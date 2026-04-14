package models

import (
	"time"

	"github.com/lib/pq"
)

type Site struct {
	ID          string         `json:"id"`
	Domain      string         `json:"domain"`
	URL         string         `json:"url"`
	Name        string         `json:"name"`
	Description string         `json:"description"`

	HasLLMsTxt      bool `json:"has_llms_txt"`
	HasAIPlugin     bool `json:"has_ai_plugin"`
	HasOpenAPI      bool `json:"has_openapi"`
	HasRobotsAI     bool `json:"has_robots_ai"`
	HasStructuredAPI bool `json:"has_structured_api"`
	HasMCPServer    bool `json:"has_mcp_server"`
	HasSchemaOrg    bool `json:"has_schema_org"`

	LLMsTxtContent string `json:"llms_txt_content,omitempty"`
	OpenAPISummary string `json:"openapi_summary,omitempty"`
	MCPEndpoint    string `json:"mcp_endpoint,omitempty"`

	AgenticScore int            `json:"agentic_score"`
	Category     string         `json:"category"`
	Tags         pq.StringArray `json:"tags"`

	IsVerified  bool   `json:"is_verified"`
	IsFeatured  bool   `json:"is_featured"`

	LastCrawledAt *time.Time `json:"last_crawled_at,omitempty"`
	CrawlStatus   string     `json:"crawl_status"`
	CrawlError    string     `json:"crawl_error,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgenticScore weights
const (
	ScoreLLMsTxt      = 25
	ScoreAIPlugin     = 20
	ScoreOpenAPI      = 20
	ScoreRobotsAI     = 5
	ScoreStructuredAPI = 15
	ScoreMCPServer    = 10
	ScoreSchemaOrg    = 5
)

func CalculateScore(s *Site) int {
	score := 0
	if s.HasLLMsTxt {
		score += ScoreLLMsTxt
	}
	if s.HasAIPlugin {
		score += ScoreAIPlugin
	}
	if s.HasOpenAPI {
		score += ScoreOpenAPI
	}
	if s.HasRobotsAI {
		score += ScoreRobotsAI
	}
	if s.HasStructuredAPI {
		score += ScoreStructuredAPI
	}
	if s.HasMCPServer {
		score += ScoreMCPServer
	}
	if s.HasSchemaOrg {
		score += ScoreSchemaOrg
	}
	return score
}
