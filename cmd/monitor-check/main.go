// monitor-check: re-crawls every subscribed domain and emails the watcher
// when the agentic-readiness score drops or any signal disappears.
//
// Intended to run weekly via launchd (com.foundry.nothumansearch.monitor).
// Safe to run more often — ListDueMonitors uses the cutoff timestamp to
// avoid checking the same row twice in one week.
package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/email"
	"github.com/unitedideas/nothumansearch/internal/models"
)

func main() {
	cutoffHours := flag.Int("cutoff-hours", 144, "Only check monitors last checked more than N hours ago (default 6 days)")
	limit := flag.Int("limit", 500, "Max monitors to process per run")
	dry := flag.Bool("dry-run", false, "Crawl and diff but don't send emails or update last_checked_at")
	flag.Parse()

	if err := database.Connect(); err != nil {
		log.Fatalf("database: %v", err)
	}

	var mailer *email.Client
	if !*dry {
		var err error
		mailer, err = email.NewClientFromEnv()
		if err != nil {
			log.Fatalf("mailer: %v", err)
		}
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "https://nothumansearch.ai"
	}

	cutoff := time.Now().Add(-time.Duration(*cutoffHours) * time.Hour)
	due, err := models.ListDueMonitors(database.DB, cutoff, *limit)
	if err != nil {
		log.Fatalf("list monitors: %v", err)
	}
	log.Printf("monitor-check: %d due monitors (cutoff %s)", len(due), cutoff.Format(time.RFC3339))

	// Dedup by domain so one crawl fans out to all watchers. Same domain
	// subscribed by 5 emails = 1 crawl, 5 possible alerts.
	byDomain := map[string][]models.Monitor{}
	order := []string{}
	for _, m := range due {
		if _, ok := byDomain[m.Domain]; !ok {
			order = append(order, m.Domain)
		}
		byDomain[m.Domain] = append(byDomain[m.Domain], m)
	}

	for _, domain := range order {
		site, cerr := crawler.CrawlSite("https://" + domain)
		for _, m := range byDomain[domain] {
			if err := handleOne(database.DB, mailer, baseURL, &m, site, cerr, *dry); err != nil {
				log.Printf("  %s (%s): error: %v", m.Domain, m.Email, err)
			}
		}
	}
	log.Printf("monitor-check done")
}

// handleOne applies a single crawl result to a single monitor row.
// site/cerr are the shared crawl result for this domain; we accept nil
// site when cerr != nil.
func handleOne(db *sql.DB, mailer *email.Client, baseURL string, m *models.Monitor, site *models.Site, cerr error, dry bool) error {
	if cerr != nil {
		if m.LastScore != nil && *m.LastScore > 0 {
			// Site previously worked, now unreachable — alert.
			return maybeAlert(db, mailer, baseURL, m, 0, "unreachable", dry,
				fmt.Sprintf("We couldn't reach %s during our weekly check. Error: %v", m.Domain, cerr))
		}
		// First-ever crawl failed. Record a zero-score baseline so the
		// monitor isn't retried forever on every run.
		log.Printf("  WARN %s: first crawl failed, baselining at score=0: %v", m.Domain, cerr)
		if dry {
			return nil
		}
		return models.UpdateMonitorCheck(db, m.ID, 0, "", false)
	}

	score := site.AgenticScore
	sigs := signalsString(site)
	hash := hashString(sigs)

	// First-ever check: just record, don't alert.
	if m.LastScore == nil {
		log.Printf("  %s: first check, score=%d signals=%s", m.Domain, score, sigs)
		if dry {
			return nil
		}
		return models.UpdateMonitorCheck(db, m.ID, score, hash, false)
	}

	drop := *m.LastScore - score
	log.Printf("  %s: score %d->%d (drop=%d)", m.Domain, *m.LastScore, score, drop)

	// Alert only on meaningful regressions: score drop ≥ 5 OR any signal disappeared.
	disappeared := signalsDisappeared(m, site)
	if drop >= 5 || disappeared != "" {
		reason := reasonText(drop, disappeared, *m.LastScore, score)
		return maybeAlert(db, mailer, baseURL, m, score, hash, dry, reason)
	}
	if dry {
		return nil
	}
	return models.UpdateMonitorCheck(db, m.ID, score, hash, false)
}

// signalsString produces a canonical signal list for hashing and diff display.
func signalsString(s *models.Site) string {
	parts := []string{}
	if s.HasLLMsTxt {
		parts = append(parts, "llms_txt")
	}
	if s.HasAIPlugin {
		parts = append(parts, "ai_plugin")
	}
	if s.HasOpenAPI {
		parts = append(parts, "openapi")
	}
	if s.HasStructuredAPI {
		parts = append(parts, "api")
	}
	if s.HasMCPServer {
		parts = append(parts, "mcp")
	}
	if s.HasRobotsAI {
		parts = append(parts, "robots_ai")
	}
	if s.HasSchemaOrg {
		parts = append(parts, "schema_org")
	}
	return strings.Join(parts, ",")
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// signalsDisappeared returns a human-readable string listing signals that
// are currently missing — only meaningful when the score has actually
// dropped since last check (we only stored a hash of the old signals, not
// the raw set). Empty string if no loss.
func signalsDisappeared(m *models.Monitor, current *models.Site) string {
	if m.LastScore == nil || current.AgenticScore >= *m.LastScore {
		return ""
	}
	missing := []string{}
	if !current.HasLLMsTxt {
		missing = append(missing, "llms.txt")
	}
	if !current.HasAIPlugin {
		missing = append(missing, "ai-plugin.json")
	}
	if !current.HasOpenAPI {
		missing = append(missing, "OpenAPI spec")
	}
	if !current.HasStructuredAPI {
		missing = append(missing, "structured JSON API")
	}
	if !current.HasMCPServer {
		missing = append(missing, "MCP server")
	}
	return strings.Join(missing, ", ")
}

func reasonText(drop int, disappeared string, before, after int) string {
	var sb strings.Builder
	if drop > 0 {
		fmt.Fprintf(&sb, "Your agentic-readiness score dropped from %d to %d (-%d).", before, after, drop)
	} else {
		fmt.Fprintf(&sb, "Your agent signals changed. Current score: %d.", after)
	}
	if disappeared != "" {
		fmt.Fprintf(&sb, " Currently missing: %s.", disappeared)
	}
	return sb.String()
}

func maybeAlert(db *sql.DB, mailer *email.Client, baseURL string, m *models.Monitor, score int, hash string, dry bool, reason string) error {
	subject := fmt.Sprintf("Agentic-readiness drop: %s", m.Domain)
	unsubURL := baseURL + "/monitor/unsubscribe/" + m.Token
	siteURL := baseURL + "/site/" + m.Domain
	htmlBody := fmt.Sprintf(`<p>Hi,</p>
<p>%s</p>
<p>See the full breakdown: <a href="%s">%s</a></p>
<hr>
<p style="color:#888;font-size:12px">You're subscribed to weekly alerts for %s.
<a href="%s">Unsubscribe</a></p>`, reason, siteURL, siteURL, m.Domain, unsubURL)
	textBody := fmt.Sprintf("%s\n\nFull breakdown: %s\n\nUnsubscribe: %s\n", reason, siteURL, unsubURL)

	log.Printf("  -> ALERT %s: %s", m.Email, reason)
	if dry {
		return nil
	}
	if _, err := mailer.Send(m.Email, subject, htmlBody, textBody); err != nil {
		// Still record the check so we don't re-alert every run on a mailer error.
		_ = models.UpdateMonitorCheck(db, m.ID, score, hash, false)
		return fmt.Errorf("send: %w", err)
	}
	return models.UpdateMonitorCheck(db, m.ID, score, hash, true)
}
