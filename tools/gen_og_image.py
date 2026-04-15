#!/usr/bin/env python3
"""Generate 1200x630 OG image for NHS homepage.

Run: python3 tools/gen_og_image.py
Output: static/img/og-default.png
"""
import os
from PIL import Image, ImageDraw, ImageFont

W, H = 1200, 630
BG = (13, 13, 14)           # #0d0d0e
SURFACE = (20, 20, 22)      # #141416
BORDER = (31, 31, 35)       # #1f1f23
TEXT = (224, 224, 224)      # #e0e0e0
MUTED = (136, 136, 136)     # #888
ACCENT = (217, 119, 87)     # #d97757

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "..", "static", "img", "og-default.png")


def load_font(size, bold=False):
    candidates = []
    if bold:
        candidates += [
            "/System/Library/Fonts/Supplemental/Arial Bold.ttf",
            "/System/Library/Fonts/Helvetica.ttc",
        ]
    candidates += [
        "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/System/Library/Fonts/Helvetica.ttc",
        "/Library/Fonts/Arial.ttf",
    ]
    for p in candidates:
        if os.path.exists(p):
            try:
                return ImageFont.truetype(p, size)
            except Exception:
                continue
    return ImageFont.load_default()


def text_w(draw, txt, font):
    bbox = draw.textbbox((0, 0), txt, font=font)
    return bbox[2] - bbox[0]


def main():
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)

    # Subtle accent bar top
    d.rectangle([(0, 0), (W, 6)], fill=ACCENT)

    # Border frame
    pad = 48
    d.rectangle([(pad, pad + 20), (W - pad, H - pad)], outline=BORDER, width=2)

    # Logo-style wordmark
    f_logo = load_font(64, bold=True)
    logo_left = "not human"
    logo_right = " search"
    lw1 = text_w(d, logo_left, f_logo)
    lw2 = text_w(d, logo_right, f_logo)
    lx = (W - (lw1 + lw2)) // 2
    ly = 110
    d.text((lx, ly), logo_left, font=f_logo, fill=TEXT)
    d.text((lx + lw1, ly), logo_right, font=f_logo, fill=ACCENT)

    # Tagline
    f_tag = load_font(44, bold=True)
    tag = "Google for AI Agents"
    tw = text_w(d, tag, f_tag)
    d.text(((W - tw) // 2, 230), tag, font=f_tag, fill=TEXT)

    # Subline
    f_sub = load_font(26)
    sub = "Search engine that indexes agent-first tools, ranked by agentic readiness."
    sw = text_w(d, sub, f_sub)
    d.text(((W - sw) // 2, 310), sub, font=f_sub, fill=MUTED)

    # Stats pills
    pill_y = 400
    pills = [
        ("736", "verified sites"),
        ("11", "categories"),
        ("MCP", "live"),
    ]
    f_n = load_font(56, bold=True)
    f_l = load_font(22)
    gap = 60
    # measure total
    widths = []
    for n, l in pills:
        nw = text_w(d, n, f_n)
        lw = text_w(d, l, f_l)
        widths.append(max(nw, lw) + 60)  # inner padding
    total = sum(widths) + gap * (len(pills) - 1)
    x = (W - total) // 2
    for (n, l), pw in zip(pills, widths):
        # pill box
        d.rounded_rectangle(
            [(x, pill_y), (x + pw, pill_y + 130)],
            radius=14, fill=SURFACE, outline=BORDER, width=2,
        )
        nw = text_w(d, n, f_n)
        lw = text_w(d, l, f_l)
        d.text((x + (pw - nw) // 2, pill_y + 16), n, font=f_n, fill=ACCENT)
        d.text((x + (pw - lw) // 2, pill_y + 86), l, font=f_l, fill=MUTED)
        x += pw + gap

    # Footer URL
    f_url = load_font(24)
    url = "nothumansearch.ai"
    uw = text_w(d, url, f_url)
    d.text(((W - uw) // 2, H - pad - 36), url, font=f_url, fill=MUTED)

    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    img.save(OUT, "PNG", optimize=True)
    print(f"wrote {OUT} ({os.path.getsize(OUT)} bytes)")


if __name__ == "__main__":
    main()
