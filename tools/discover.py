#!/usr/bin/env python3
"""
NHS discovery pipeline.

Pulls domains from external agent-ready directories (awesome-mcp-servers,
PulseMCP, llms.txt directories) and submits any unseen ones to NHS's
own /api/v1/submit endpoint. Idempotent: existing domains are no-ops
thanks to ON CONFLICT DO NOTHING in the submit handler.

Runs weekly via launchd. Compounds the index size over time without
requiring manual seed-list updates.
"""
import json
import os
import re
import shlex
import subprocess
import sys
import time
import urllib.parse
import urllib.request
import uuid
from concurrent.futures import ThreadPoolExecutor, as_completed
from urllib.error import HTTPError, URLError

NHS_BASE = "https://nothumansearch.ai"
UA = "NHS-Discovery/1.0 (+https://nothumansearch.ai)"
BATCH_SIZE = 25
SSH_TIMEOUT_SECONDS = 900
SOURCE_WORKERS = 8
INDEX_CHECK_WORKERS = 32
_GH_TOKEN = None


def github_token():
    global _GH_TOKEN
    if _GH_TOKEN is not None:
        return _GH_TOKEN
    try:
        _GH_TOKEN = subprocess.check_output(
            ["gh", "auth", "token"],
            stderr=subprocess.DEVNULL,
            text=True,
            timeout=5,
        ).strip()
    except Exception:
        _GH_TOKEN = ""
    return _GH_TOKEN

# Root domains we never want to submit — noise, platform infra, social, etc.
# Matched against registrable domain (last 2 labels), so subdomains are caught too.
SKIP_ROOT_DOMAINS = {
    "github.com", "githubusercontent.com", "gitlab.com", "bitbucket.org",
    "github.io", "gitlab.io", "netlify.app", "vercel.app", "pages.dev",
    "shields.io", "opencollective.com", "ko-fi.com", "buymeacoffee.com",
    "patreon.com", "paypal.me", "paypal.com",
    "twitter.com", "x.com", "facebook.com", "linkedin.com", "youtube.com",
    "youtu.be", "instagram.com", "reddit.com", "t.me", "discord.gg",
    "discord.com", "mastodon.social",
    "medium.com", "dev.to", "hashnode.dev", "substack.com",
    "npmjs.com", "pypi.org", "crates.io", "rubygems.org", "packagist.org",
    "docker.com", "hub.docker.com",
    "example.com", "localhost", "127.0.0.1",
    "google.com", "apple.com", "microsoft.com", "amazon.com",
    "wikipedia.org", "wikimedia.org",
    "star-history.com",
    "archive.org", "web.archive.org",
    # MCP registries themselves — we harvest them, don't submit them
    "mcp.so", "smithery.ai", "glama.ai", "pulsemcp.com",
    "llmstxt.site", "llmstxt.cloud", "llmstxt.org",
    "goo.gl", "bit.ly", "t.co", "tinyurl.com",
    # NHS-internal — never submit ourselves or test domains
    "nothumansearch.ai", "nothumansearch.com",
}


def registrable_domain(host):
    """Return the last 2 labels of host (naive — good enough for blocklist)."""
    parts = host.split(".")
    if len(parts) < 2:
        return host
    return ".".join(parts[-2:])

URL_RE = re.compile(r"https?://[a-zA-Z0-9][a-zA-Z0-9.-]*\.[a-zA-Z]{2,}(?:/[^\s\"'<>)]*)?")


def http_get(url, timeout=20, retries=2):
    last = None
    for attempt in range(retries + 1):
        try:
            headers = {"User-Agent": UA, "Accept": "*/*"}
            if url.startswith("https://api.github.com/"):
                token = github_token()
                if token:
                    headers["Authorization"] = f"Bearer {token}"
            req = urllib.request.Request(url, headers=headers)
            with urllib.request.urlopen(req, timeout=timeout) as resp:
                return resp.read().decode("utf-8", errors="replace")
        except (HTTPError, URLError) as e:
            last = e
            # 4xx is usually permanent; retry 429 + 5xx + transient 410
            if isinstance(e, HTTPError) and e.code not in (410, 429, 500, 502, 503, 504):
                raise
            if attempt < retries:
                time.sleep(1.5 * (attempt + 1))
    raise last  # type: ignore[misc]


def extract_domain(raw_url):
    """Return lowercase netloc or None if it should be skipped."""
    try:
        parsed = urllib.parse.urlparse(raw_url.strip())
    except Exception:
        return None
    host = (parsed.netloc or "").lower().strip()
    if not host or "." not in host:
        return None
    # Drop port, auth
    host = host.split("@")[-1].split(":")[0]
    if host in SKIP_ROOT_DOMAINS:
        return None
    if registrable_domain(host) in SKIP_ROOT_DOMAINS:
        return None
    return host


def from_mcp_registry():
    """Pull websiteUrl + remote URL domains from the official MCP registry."""
    url = "https://registry.modelcontextprotocol.io/v0/servers"
    try:
        body = http_get(url)
    except (HTTPError, URLError) as e:
        print(f"[mcp-registry] fetch failed: {e}", file=sys.stderr)
        return set()
    domains = set()
    try:
        data = json.loads(body)
    except json.JSONDecodeError as e:
        print(f"[mcp-registry] json error: {e}", file=sys.stderr)
        return set()
    for entry in data.get("servers", []):
        srv = entry.get("server", {})
        for field in ("websiteUrl",):
            d = extract_domain(srv.get(field, ""))
            if d:
                domains.add(d)
        for remote in srv.get("remotes", []):
            d = extract_domain(remote.get("url", ""))
            if d:
                domains.add(d)
    print(f"[mcp-registry] {len(domains)} candidate domains")
    return domains


def from_awesome_mcp():
    """Scrape domains from github.com/punkpeye/awesome-mcp-servers."""
    url = "https://raw.githubusercontent.com/punkpeye/awesome-mcp-servers/main/README.md"
    try:
        body = http_get(url)
    except (HTTPError, URLError) as e:
        print(f"[awesome-mcp] fetch failed: {e}", file=sys.stderr)
        return set()
    domains = set()
    for m in URL_RE.findall(body):
        d = extract_domain(m)
        if d:
            domains.add(d)
    print(f"[awesome-mcp] {len(domains)} candidate domains")
    return domains


def from_pulsemcp(max_pages=40):
    """Walk PulseMCP v0beta API and collect external_url per server.

    `next` field is a full URL, not a cursor. 12k+ servers total as of
    2026-04; cap pages so we don't thrash the index on every run.
    """
    domains = set()
    next_url = "https://api.pulsemcp.com/v0beta/servers?count_per_page=100"
    pages = 0
    while next_url and pages < max_pages:
        try:
            body = http_get(next_url)
        except (HTTPError, URLError) as e:
            print(f"[pulsemcp] fetch failed page={pages}: {e}", file=sys.stderr)
            break
        try:
            data = json.loads(body)
        except json.JSONDecodeError as e:
            print(f"[pulsemcp] json error: {e}", file=sys.stderr)
            break
        servers = data.get("servers", [])
        if not servers:
            break
        for s in servers:
            ext = s.get("external_url")
            if ext:
                d = extract_domain(ext)
                if d:
                    domains.add(d)
        next_url = data.get("next")
        pages += 1
        time.sleep(0.3)
    print(f"[pulsemcp] {len(domains)} candidate domains across {pages} pages")
    return domains


def from_mcpso(max_pages=10):
    """Scrape mcp.so listing pages for external host references.

    The index page + /servers?page=N pages embed many external URLs
    (project homepages, docs, etc.). Yields ~150-200 unique hosts/page
    before hitting diminishing returns.
    """
    domains = set()
    for page in range(1, max_pages + 1):
        url = f"https://mcp.so/servers?page={page}" if page > 1 else "https://mcp.so/"
        try:
            body = http_get(url)
        except (HTTPError, URLError) as e:
            print(f"[mcp.so] page={page} fetch failed: {e}", file=sys.stderr)
            continue
        page_domains = set()
        for m in URL_RE.findall(body):
            d = extract_domain(m)
            if d and d != "mcp.so":
                page_domains.add(d)
        if not page_domains:
            break
        domains |= page_domains
        time.sleep(0.3)
    print(f"[mcp.so] {len(domains)} candidate domains")
    return domains


def from_llmstxt_site():
    """Scrape llms.txt index sites for referenced domains."""
    domains = set()
    for url in ("https://llmstxt.site/", "https://directory.llmstxt.cloud/"):
        try:
            body = http_get(url)
        except (HTTPError, URLError) as e:
            print(f"[{url}] fetch failed: {e}", file=sys.stderr)
            continue
        for m in URL_RE.findall(body):
            d = extract_domain(m)
            if d:
                domains.add(d)
    # Drop the index sites themselves
    domains.discard("llmstxt.site")
    domains.discard("directory.llmstxt.cloud")
    domains.discard("llmstxt.org")
    print(f"[llmstxt] {len(domains)} candidate domains")
    return domains


def from_apis_guru():
    """Harvest domains from apis.guru — the OpenAPI directory."""
    domains = set()
    try:
        body = http_get("https://api.apis.guru/v2/list.json")
        data = json.loads(body)
        for key in data:
            for ver_key, ver in data[key].get("versions", {}).items():
                info = ver.get("info", {})
                contact = info.get("contact", {})
                url = contact.get("url", "") or info.get("x-origin", [{}])[0].get("url", "") if isinstance(info.get("x-origin"), list) else ""
                swagger_url = ver.get("swaggerUrl", "")
                for u in (url, swagger_url):
                    d = extract_domain(u)
                    if d:
                        domains.add(d)
    except (HTTPError, URLError, json.JSONDecodeError) as e:
        print(f"[apis.guru] fetch failed: {e}", file=sys.stderr)
    print(f"[apis.guru] {len(domains)} candidate domains")
    return domains


def from_smithery():
    """Harvest domains from smithery.ai MCP server registry."""
    domains = set()
    for page in range(1, 20):
        try:
            body = http_get(f"https://registry.smithery.ai/servers?page={page}&pageSize=100")
            data = json.loads(body)
            servers = data.get("servers", [])
            if not servers:
                break
            for s in servers:
                homepage = s.get("homepage", "")
                d = extract_domain(homepage)
                if d:
                    domains.add(d)
        except (HTTPError, URLError, json.JSONDecodeError) as e:
            print(f"[smithery p{page}] fetch failed: {e}", file=sys.stderr)
            break
    print(f"[smithery] {len(domains)} candidate domains")
    return domains


def from_github_mcp_topic(max_pages=33):
    """Harvest homepage domains from GitHub repos tagged with MCP topics.

    Two topics: `model-context-protocol` (7k+ repos) and `mcp-server`.
    Unauthenticated GitHub search: 10 rpm limit, 30 results/page, 1000 max.
    Many MCP server repos list their product's homepage — stronger signal
    than the README scrape used by `from_awesome_mcp`.
    """
    domains = set()
    for topic in ("model-context-protocol", "mcp-server"):
        for page in range(1, max_pages + 1):
            url = (
                f"https://api.github.com/search/repositories?"
                f"q=topic:{topic}&sort=stars&order=desc&per_page=30&page={page}"
            )
            try:
                body = http_get(url)
            except (HTTPError, URLError) as e:
                print(f"[github:{topic} p{page}] fetch failed: {e}", file=sys.stderr)
                break
            try:
                data = json.loads(body)
            except json.JSONDecodeError as e:
                print(f"[github:{topic} p{page}] json error: {e}", file=sys.stderr)
                break
            items = data.get("items", [])
            if not items:
                break
            for x in items:
                hp = x.get("homepage") or ""
                d = extract_domain(hp)
                if d:
                    domains.add(d)
            time.sleep(0.2 if github_token() else 6.5)  # stay under 10 rpm when unauthenticated
    print(f"[github] {len(domains)} candidate domains")
    return domains


def already_indexed(domain):
    """Return True if NHS already has this domain."""
    try:
        req = urllib.request.Request(
            f"{NHS_BASE}/api/v1/site/{domain}",
            headers={"User-Agent": UA},
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            return resp.status == 200
    except HTTPError as e:
        if e.code == 404:
            return False
        return True  # fail-safe: assume indexed, skip submission
    except URLError:
        return True


def submit(domain):
    payload = json.dumps({"url": f"https://{domain}"}).encode()
    req = urllib.request.Request(
        f"{NHS_BASE}/api/v1/submit",
        data=payload,
        headers={"User-Agent": UA, "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            return resp.status in (200, 201)
    except (HTTPError, URLError) as e:
        print(f"[submit] {domain} failed: {e}", file=sys.stderr)
        return False


def crawl_via_ssh(domains):
    """Pipe domains to the crawler on Fly via SSH, bypassing HTTP rate limits."""
    if not domains:
        return 0, 0
    print(f"Piping {len(domains)} domains to crawler via fly ssh in batches of {BATCH_SIZE}...")
    submitted = 0
    errors = 0
    try:
        fly_bin = "/opt/homebrew/bin/fly"
        token = subprocess.check_output(
            ["/usr/bin/security", "find-generic-password", "-a", "foundry", "-s", "fly-api-token", "-w"],
            stderr=subprocess.DEVNULL,
        ).decode("utf-8").strip()
        env = dict(os.environ)
        env["FLY_ACCESS_TOKEN"] = token
        env.setdefault("HOME", "/Users/owlassist")

        for idx in range(0, len(domains), BATCH_SIZE):
            batch = domains[idx : idx + BATCH_SIZE]
            remote_file = f"/tmp/nhs-discover-{uuid.uuid4().hex}.txt"
            payload = "\n".join(f"https://{domain}" for domain in batch) + "\n"
            remote_script = (
                f"cat > {remote_file} <<'EOF'\n"
                f"{payload}"
                "EOF\n"
                f"/usr/bin/timeout {SSH_TIMEOUT_SECONDS - 30} /app/crawler -file {remote_file} -workers 5\n"
                f"rc=$?\n"
                f"rm -f {remote_file}\n"
                f"exit $rc"
            )
            remote_cmd = f"/bin/sh -lc {shlex.quote(remote_script)}"
            print(f"  [batch {idx // BATCH_SIZE + 1}] {len(batch)} domains")
            try:
                result = subprocess.run(
                    [fly_bin, "ssh", "console", "-a", "nothumansearch", "-C",
                     remote_cmd],
                    timeout=SSH_TIMEOUT_SECONDS,
                    env=env,
                )
            except subprocess.TimeoutExpired:
                print(f"  [crawler] batch timeout after {SSH_TIMEOUT_SECONDS}s")
                errors += len(batch)
                continue
            if result.returncode == 0:
                submitted += len(batch)
            else:
                errors += len(batch)
        return submitted, errors
    except FileNotFoundError:
        print("  [crawler] fly CLI not found, falling back to HTTP submit")
        return -1, 0
    except subprocess.CalledProcessError:
        print("  [crawler] fly token missing from Keychain", file=sys.stderr)
        return 0, len(domains)


def main():
    print(f"=== NHS discovery run @ {time.strftime('%Y-%m-%d %H:%M:%S')} ===")
    candidates = set()
    sources = [
        (from_mcp_registry, 100),
        (from_pulsemcp, 95),
        (from_mcpso, 90),
        (from_smithery, 85),
        (from_github_mcp_topic, 80),
        (from_awesome_mcp, 75),
        (from_apis_guru, 35),
        (from_llmstxt_site, 10),
    ]
    weights = {}
    with ThreadPoolExecutor(max_workers=SOURCE_WORKERS) as pool:
        futures = {pool.submit(source): (source.__name__, weight) for source, weight in sources}
        for future in as_completed(futures):
            name, weight = futures[future]
            try:
                domains = future.result()
            except Exception as e:
                print(f"[{name}] failed: {e}", file=sys.stderr)
                continue
            candidates |= domains
            for domain in domains:
                weights[domain] = max(weights.get(domain, 0), weight)
    print(f"Total unique candidates: {len(candidates)}")

    new_domains = set()
    skipped = 0
    with ThreadPoolExecutor(max_workers=INDEX_CHECK_WORKERS) as pool:
        futures = {pool.submit(already_indexed, domain): domain for domain in candidates}
        for future in as_completed(futures):
            domain = futures[future]
            try:
                indexed = future.result()
            except Exception:
                indexed = True
            if indexed:
                skipped += 1
            else:
                new_domains.add(domain)
    new_domains = sorted(new_domains, key=lambda d: (-weights.get(d, 0), d))

    print(f"New domains to crawl: {len(new_domains)} (already indexed: {skipped})")

    if not new_domains:
        print("=== done: nothing new ===")
        return 0

    submitted, errors = crawl_via_ssh(new_domains)
    if submitted == -1:
        submitted = 0
        errors = 0
        for i, domain in enumerate(new_domains):
            if submit(domain):
                submitted += 1
                print(f"  [+] {domain}")
            else:
                errors += 1
            if i % 10 == 9:
                time.sleep(1.0)

    print(f"=== done: submitted={submitted} already_indexed={skipped} errors={errors} ===")
    return 0 if errors == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
