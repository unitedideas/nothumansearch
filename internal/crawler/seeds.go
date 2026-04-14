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
	{"https://autogen.microsoft.com", false},
	{"https://www.phidata.com", false},

	// Productivity / Collaboration
	{"https://linear.app", false},
	{"https://notion.so", false},
	{"https://slack.com", false},
	{"https://discord.com", false},
	{"https://airtable.com", false},
	{"https://zapier.com", false},
	{"https://make.com", false},
	{"https://asana.com", false},
	{"https://trello.com", false},
	{"https://monday.com", false},

	// E-commerce
	{"https://shopify.dev", false},
	{"https://bigcommerce.com", false},
	{"https://woocommerce.com", false},
	{"https://snipcart.com", false},
	{"https://square.com", false},

	// Finance / Fintech
	{"https://plaid.com", false},
	{"https://mercury.com", false},
	{"https://wise.com", false},
	{"https://brex.com", false},
	{"https://alpaca.markets", false},
	{"https://polygon.io", false},
	{"https://coinbase.com", false},

	// Healthcare / Science
	{"https://health.gov", false},
	{"https://pubmed.ncbi.nlm.nih.gov", false},
	{"https://clinicaltrials.gov", false},

	// MCP-ready / Agent infrastructure
	{"https://modelcontextprotocol.io", false},
	{"https://smithery.ai", false},
	{"https://glama.ai", false},
	{"https://composio.dev", false},
	{"https://e2b.dev", false},
	{"https://browserbase.com", false},

	// Cloud / Infrastructure
	{"https://railway.app", false},
	{"https://deno.com", false},
	{"https://bun.sh", false},
	{"https://turso.tech", false},
	{"https://upstash.com", false},
	{"https://convex.dev", false},
	{"https://modal.com", false},

	// Auth / Identity
	{"https://clerk.com", false},
	{"https://auth0.com", false},
	{"https://workos.com", false},

	// Monitoring / Observability
	{"https://sentry.io", false},
	{"https://posthog.com", false},
	{"https://betterstack.com", false},
	{"https://grafana.com", false},
	{"https://datadog.com", false},

	// Communication APIs
	{"https://twilio.com", false},
	{"https://sendgrid.com", false},
	{"https://postmark.com", false},
	{"https://pusher.com", false},

	// AI / ML tools
	{"https://deepgram.com", false},
	{"https://assemblyai.com", false},
	{"https://unstructured.io", false},
	{"https://pinecone.io", false},
	{"https://weaviate.io", false},
	{"https://qdrant.tech", false},
	{"https://chroma.dev", false},
	{"https://together.ai", false},
	{"https://groq.com", false},
	{"https://fireworks.ai", false},
	{"https://perplexity.ai", false},

	// Document / Content
	{"https://docusign.com", false},
	{"https://contentful.com", false},
	{"https://sanity.io", false},
	{"https://storyblok.com", false},

	// Data / Analytics
	{"https://snowflake.com", false},
	{"https://databricks.com", false},
	{"https://dbt.com", false},
	{"https://fivetran.com", false},
	{"https://segment.com", false},
	{"https://mixpanel.com", false},
	{"https://amplitude.com", false},

	// Security
	{"https://snyk.io", false},
	{"https://1password.com", false},

	// Maps / Geo
	{"https://mapbox.com", false},
	{"https://here.com", false},

	// Job / HR / Recruiting
	{"https://lever.co", false},
	{"https://greenhouse.io", false},
	{"https://ashbyhq.com", false},
	{"https://workable.com", false},

	// Media / Files
	{"https://cloudinary.com", false},
	{"https://mux.com", false},
	{"https://uploadthing.com", false},

	// Government / Open data
	{"https://data.gov", false},
	{"https://api.census.gov", false},
	{"https://developer.nrel.gov", false},
}
