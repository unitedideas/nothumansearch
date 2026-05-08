package handlers

import (
	"testing"

	"github.com/unitedideas/nothumansearch/internal/models"
)

func TestScoreFixEligibleRequiresHardSignal(t *testing.T) {
	if scoreFixEligible(nil) {
		t.Fatal("nil site should not be score-fix eligible")
	}
	if scoreFixEligible(&models.Site{HasLLMsTxt: true, HasRobotsAI: true, HasSchemaOrg: true}) {
		t.Fatal("passive-only site should not be score-fix eligible")
	}
	if !scoreFixEligible(&models.Site{HasOpenAPI: true}) {
		t.Fatal("hard-signal site should be score-fix eligible")
	}
}
