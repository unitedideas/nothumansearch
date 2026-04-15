package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/unitedideas/nothumansearch/internal/models"
)

// BadgeHandler serves embeddable SVG score badges at /badge/{domain}.svg.
// Sites proud of their agentic readiness can embed the badge on their docs;
// every impression is an NHS referral.
//
//   <a href="https://nothumansearch.ai/site/stripe.com">
//     <img src="https://nothumansearch.ai/badge/stripe.com.svg" alt="Agentic Ready">
//   </a>
type BadgeHandler struct {
	DB *sql.DB
}

func NewBadgeHandler(db *sql.DB) *BadgeHandler {
	return &BadgeHandler{DB: db}
}

func (h *BadgeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/badge/")
	domain := strings.TrimSuffix(path, ".svg")
	domain = strings.TrimPrefix(domain, "www.")
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" || strings.ContainsAny(domain, "/?&#") {
		http.Error(w, "invalid domain", http.StatusBadRequest)
		return
	}

	score := -1
	site, err := models.GetSiteByDomain(h.DB, domain)
	if err == nil {
		score = site.AgenticScore
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	// Short cache: badges should reflect recent recrawls within a day.
	w.Header().Set("Cache-Control", "public, max-age=3600, s-maxage=3600")
	w.Write([]byte(renderBadgeSVG(score)))
}

func renderBadgeSVG(score int) string {
	// Colour buckets match the /site/{domain} page palette.
	// score == -1 means "unknown" (site not in index).
	var right, rightText string
	switch {
	case score < 0:
		right = "#6b7280"
		rightText = "not indexed"
	case score >= 70:
		right = "#22c55e"
		rightText = fmt.Sprintf("%d / 100", score)
	case score >= 40:
		right = "#eab308"
		rightText = fmt.Sprintf("%d / 100", score)
	default:
		right = "#ef4444"
		rightText = fmt.Sprintf("%d / 100", score)
	}

	const leftText = "agentic ready"
	leftWidth := 102
	rightWidth := 82
	totalWidth := leftWidth + rightWidth

	// Standard shields.io-ish layout. Monospace-ish numbers, readable left label.
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20" role="img" aria-label="agentic ready: %s">
  <title>agentic ready: %s</title>
  <linearGradient id="s" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <clipPath id="r"><rect width="%d" height="20" rx="3" fill="#fff"/></clipPath>
  <g clip-path="url(#r)">
    <rect width="%d" height="20" fill="#0d0d0e"/>
    <rect x="%d" width="%d" height="20" fill="%s"/>
    <rect width="%d" height="20" fill="url(#s)"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">%s</text>
    <text x="%d" y="14">%s</text>
  </g>
</svg>`,
		totalWidth, rightText, rightText,
		totalWidth,
		leftWidth,
		leftWidth, rightWidth, right,
		totalWidth,
		leftWidth/2, leftText,
		leftWidth/2, leftText,
		leftWidth+rightWidth/2, rightText,
		leftWidth+rightWidth/2, rightText,
	)
}
