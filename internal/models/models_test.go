package models

import "testing"

func TestSiteHasHardAgentSignal(t *testing.T) {
	tests := []struct {
		name string
		site Site
		want bool
	}{
		{
			name: "passive signals do not qualify",
			site: Site{HasLLMsTxt: true, HasRobotsAI: true, HasSchemaOrg: true},
			want: false,
		},
		{name: "structured API qualifies", site: Site{HasStructuredAPI: true}, want: true},
		{name: "OpenAPI qualifies", site: Site{HasOpenAPI: true}, want: true},
		{name: "ai plugin qualifies", site: Site{HasAIPlugin: true}, want: true},
		{name: "MCP qualifies", site: Site{HasMCPServer: true}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.site.HasHardAgentSignal(); got != tt.want {
				t.Fatalf("HasHardAgentSignal() = %v, want %v", got, tt.want)
			}
		})
	}
}
