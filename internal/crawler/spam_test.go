package crawler

import (
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

// TestIsSpam locks in the foreign-language SEO-farm detector.
// False positives cost us real indexed sites; false negatives let gambling
// spam rank via their llms.txt + schema.org signals. High-precision bar.
func TestIsSpam(t *testing.T) {
	cases := []struct {
		name string
		site models.Site
		want bool
	}{
		// Real spam that slipped through and motivated the rule (spell.so 2026-04-18).
		{
			name: "indonesian gambling megawin",
			site: models.Site{
				Name:        "MEGAWIN288 Harapan Bangsa Kemenangan Setiap Hari",
				Description: "MEGAWIN288 merupakan situs platform yang menjadi harapan bangsa karna sangat mudah mendapatkan kemenangan di semua jenis permainan setiap harinya.",
			},
			want: true,
		},
		// Single Indonesian gambling marker is NOT enough — avoid false positives.
		{
			name: "one-word false-positive guard",
			site: models.Site{
				Name:        "Kemenangan — a Javanese cultural history project",
				Description: "Digital archive of Kemenangan village folk records.",
			},
			want: false,
		},
		// Pairs of Indonesian markers → spam.
		{
			name: "two indonesian markers",
			site: models.Site{
				Name:        "Bandar Judi Terpercaya",
				Description: "Bandar judi online terbaik",
			},
			want: true,
		},
		// Thai gambling.
		{
			name: "thai gambling",
			site: models.Site{
				Name:        "เว็บแทงบอลออนไลน์",
				Description: "แทงบอล บาคาร่า",
			},
			want: true,
		},
		// Legit AI/dev tools must pass.
		{
			name: "legit Foundry property",
			site: models.Site{
				Name:        "Bring Your AI — harness migration for AI agents",
				Description: "Move Claude Code to Codex locally without sending harness data to a server.",
			},
			want: false,
		},
		{
			name: "legit dev tool with uncommon words",
			site: models.Site{
				Name:        "Slotify — calendar booking",
				Description: "Share availability as bookable slots.",
			},
			want: false,
		},
		// English SEO-farm brand stuffing.
		{
			name: "slot gacor seo farm",
			site: models.Site{
				Name:        "Slot Gacor Terbaru",
				Description: "Link slot gacor resmi hari ini",
			},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isSpam(&tc.site)
			if got != tc.want {
				t.Errorf("isSpam(%q)=%v, want %v", tc.site.Name, got, tc.want)
			}
		})
	}
}

// TestCategorize_SpamShortCircuit: isSpam must run before the legitimate
// category rules so a spam site is NEVER tagged as "developer" etc.
func TestCategorize_SpamShortCircuit(t *testing.T) {
	// A spam site that ALSO contains dev keywords in its description — the
	// SEO farm class sometimes stuffs API mentions alongside gambling terms.
	site := &models.Site{
		Domain:      "example-spam.tld",
		Name:        "MEGAWIN88 API Gateway",
		Description: "API developer tools kemenangan judi bandar",
	}
	got := categorize(site)
	if got != "spam" {
		t.Errorf("categorize = %q, want \"spam\" (isSpam short-circuit missed)", got)
	}
}
