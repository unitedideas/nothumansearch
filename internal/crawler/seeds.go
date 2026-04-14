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
	{"https://nothumansearch.fly.dev", true},

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

	// Translation / Language
	{"https://deepl.com", false},
	{"https://cloud.google.com/translate", false},
	{"https://libretranslate.com", false},

	// Shipping / Logistics
	{"https://easypost.com", false},
	{"https://goshippo.com", false},
	{"https://shipstation.com", false},
	{"https://aftership.com", false},

	// Calendar / Scheduling
	{"https://cal.com", false},
	{"https://calendly.com", false},
	{"https://cronofy.com", false},

	// Travel / Flights
	{"https://amadeus.com", false},
	{"https://developers.amadeus.com", false},
	{"https://travelport.com", false},

	// Social Media APIs
	{"https://developer.x.com", false},
	{"https://developers.facebook.com", false},
	{"https://developers.reddit.com", false},
	{"https://buffer.com", false},
	{"https://hootsuite.com", false},

	// Image / Video Generation
	{"https://leonardo.ai", false},
	{"https://ideogram.ai", false},
	{"https://runway.com", false},
	{"https://luma.ai", false},

	// Music / Audio
	{"https://developer.spotify.com", false},
	{"https://soundcloud.com", false},
	{"https://suno.com", false},

	// PDF / Document Processing
	{"https://docparser.com", false},
	{"https://pdf.co", false},
	{"https://smallpdf.com", false},

	// CRM / Sales
	{"https://hubspot.com", false},
	{"https://pipedrive.com", false},
	{"https://close.com", false},

	// Forms / Surveys
	{"https://typeform.com", false},
	{"https://tally.so", false},

	// DNS / Domain
	{"https://dnsimple.com", false},
	{"https://name.com", false},

	// Testing / QA
	{"https://browserstack.com", false},
	{"https://lambdatest.com", false},

	// Notifications / Push
	{"https://onesignal.com", false},
	{"https://pushover.net", false},
	{"https://ntfy.sh", false},

	// Education / Learning
	{"https://coursera.org", false},
	{"https://udemy.com", false},
	{"https://edx.org", false},
	{"https://khanacademy.org", false},
	{"https://duolingo.com", false},

	// Healthcare / Biotech (expanding thin category)
	{"https://developer.apple.com/healthkit", false},
	{"https://fhir.org", false},
	{"https://openfda.gov", false},
	{"https://medlineplus.gov", false},
	{"https://rxnav.nlm.nih.gov", false},

	// Security (expanding thin category)
	{"https://letsencrypt.org", false},
	{"https://vault.hashicorp.com", false},
	{"https://virustotal.com", false},
	{"https://haveibeenpwned.com", false},
	{"https://cve.mitre.org", false},

	// Jobs / HR (expanding thin category)
	{"https://smartrecruiters.com", false},
	{"https://breezy.hr", false},
	{"https://recruitee.com", false},
	{"https://bamboohr.com", false},
	{"https://gusto.com", false},

	// Ecommerce (expanding thin category)
	{"https://stripe.com/payments", false},
	{"https://paddle.com", false},
	{"https://lemon.squeezy.com", false},
	{"https://gumroad.com", false},
	{"https://printful.com", false},

	// Design / Creative
	{"https://figma.com", false},
	{"https://canva.com", false},
	{"https://dribbble.com", false},

	// Legal / Compliance
	{"https://docuseal.co", false},
	{"https://termly.io", false},

	// Real Estate
	{"https://developer.zillow.com", false},
	{"https://www.realtor.com", false},

	// Food / Restaurant
	{"https://developer.doordash.com", false},
	{"https://docs.yelp.com", false},

	// IoT / Hardware
	{"https://particle.io", false},
	{"https://arduino.cc", false},

	// Cryptocurrency / Web3
	{"https://docs.etherscan.io", false},
	{"https://alchemy.com", false},
	{"https://moralis.io", false},

	// Productivity (expanding)
	{"https://clickup.com", false},
	{"https://todoist.com", false},
	{"https://jira.atlassian.com", false},
}
