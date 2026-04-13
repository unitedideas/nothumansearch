package crawler

// SeedSites are the initial sites to crawl.
// Foundry businesses first (featured), then well-known agent-ready sites.
var SeedSites = []struct {
	URL      string
	Featured bool
}{
	// Foundry businesses (featured)
	{"https://aidevboard.com", true},
	{"https://8bitconcepts.com", true},
	{"https://agentcanary.com", true},

	// Major AI platforms (known agent-ready)
	{"https://platform.openai.com", false},
	{"https://docs.anthropic.com", false},
	{"https://ai.google.dev", false},
	{"https://docs.cohere.com", false},
	{"https://docs.mistral.ai", false},

	// Developer tools with APIs
	{"https://github.com", false},
	{"https://api.github.com", false},
	{"https://vercel.com", false},
	{"https://fly.io", false},
	{"https://stripe.com", false},
	{"https://resend.com", false},
	{"https://supabase.com", false},
	{"https://neon.tech", false},
	{"https://planetscale.com", false},
	{"https://cloudflare.com", false},
	{"https://render.com", false},

	// Data / APIs
	{"https://newsapi.org", false},
	{"https://openweathermap.org", false},
	{"https://exchangeratesapi.io", false},

	// AI tools
	{"https://huggingface.co", false},
	{"https://replicate.com", false},
	{"https://stability.ai", false},
	{"https://elevenlabs.io", false},

	// Search / Knowledge
	{"https://exa.ai", false},
	{"https://tavily.com", false},
	{"https://serper.dev", false},
	{"https://wikipedia.org", false},

	// Agent frameworks
	{"https://langchain.com", false},
	{"https://docs.llamaindex.ai", false},
	{"https://www.crewai.com", false},

	// Productivity
	{"https://linear.app", false},
	{"https://notion.so", false},
	{"https://slack.com", false},
	{"https://discord.com", false},

	// E-commerce
	{"https://shopify.dev", false},
	{"https://bigcommerce.com", false},

	// Finance
	{"https://plaid.com", false},
	{"https://mercury.com", false},

	// Healthcare
	{"https://health.gov", false},

	// MCP-ready sites
	{"https://modelcontextprotocol.io", false},
	{"https://smithery.ai", false},
	{"https://glama.ai", false},
}
