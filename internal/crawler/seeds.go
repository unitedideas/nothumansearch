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
	{"https://nothumansearch.ai", true},

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

	// CI/CD / DevOps
	{"https://circleci.com", false},
	{"https://buildkite.com", false},
	{"https://jenkins.io", false},
	{"https://gitlab.com", false},
	{"https://bitbucket.org", false},
	{"https://travis-ci.com", false},
	{"https://semaphoreci.com", false},
	{"https://argo-cd.readthedocs.io", false},
	{"https://terraform.io", false},
	{"https://pulumi.com", false},

	// Package / Registry
	{"https://npmjs.com", false},
	{"https://pypi.org", false},
	{"https://crates.io", false},
	{"https://pkg.go.dev", false},
	{"https://rubygems.org", false},
	{"https://hub.docker.com", false},

	// Databases / Data stores
	{"https://redis.io", false},
	{"https://mongodb.com", false},
	{"https://elastic.co", false},
	{"https://cockroachlabs.com", false},
	{"https://fauna.com", false},
	{"https://timescale.com", false},
	{"https://influxdata.com", false},
	{"https://dgraph.io", false},

	// AI Agents / Orchestration
	{"https://fixie.ai", false},
	{"https://superagent.sh", false},
	{"https://relevanceai.com", false},
	{"https://dust.tt", false},
	{"https://retool.com", false},
	{"https://n8n.io", false},
	{"https://temporal.io", false},
	{"https://inngest.com", false},
	{"https://trigger.dev", false},

	// Marketing / Email
	{"https://mailchimp.com", false},
	{"https://convertkit.com", false},
	{"https://brevo.com", false},
	{"https://activecampaign.com", false},
	{"https://drip.com", false},
	{"https://customer.io", false},
	{"https://mailgun.com", false},

	// SEO / Search marketing
	{"https://ahrefs.com", false},
	{"https://semrush.com", false},
	{"https://moz.com", false},
	{"https://screaminFrog.co.uk", false},
	{"https://searchconsole.google.com", false},

	// Payments / Billing (more)
	{"https://chargebee.com", false},
	{"https://recurly.com", false},
	{"https://razorpay.com", false},
	{"https://mollie.com", false},
	{"https://paypal.com", false},

	// News / Media APIs
	{"https://developer.nytimes.com", false},
	{"https://guardian.co.uk/open-platform", false},
	{"https://mediastack.com", false},
	{"https://gnews.io", false},
	{"https://rss2json.com", false},

	// Weather / Environment
	{"https://weatherapi.com", false},
	{"https://tomorrow.io", false},
	{"https://visualcrossing.com", false},
	{"https://developer.accuweather.com", false},

	// Government / Open Data (intl)
	{"https://api.worldbank.org", false},
	{"https://api.nasa.gov", false},
	{"https://usaspending.gov", false},
	{"https://catalog.data.gov", false},
	{"https://api.fda.gov", false},

	// Legal / Compliance (more)
	{"https://ironclad.com", false},
	{"https://clio.com", false},
	{"https://legalzoom.com", false},

	// Customer Support / Chat
	{"https://intercom.com", false},
	{"https://zendesk.com", false},
	{"https://freshdesk.com", false},
	{"https://crisp.chat", false},
	{"https://helpscout.com", false},
	{"https://chatwoot.com", false},

	// Accounting / Finance
	{"https://quickbooks.intuit.com", false},
	{"https://xero.com", false},
	{"https://freshbooks.com", false},
	{"https://wave.com", false},

	// Automation / Integration (more)
	{"https://pipedream.com", false},
	{"https://tray.io", false},
	{"https://workato.com", false},
	{"https://ifttt.com", false},

	// Image / Media processing
	{"https://imgix.com", false},
	{"https://tinypng.com", false},
	{"https://remove.bg", false},
	{"https://bannerbear.com", false},

	// Geo / Location (more)
	{"https://ipinfo.io", false},
	{"https://ipgeolocation.io", false},
	{"https://opencagedata.com", false},
	{"https://positionstack.com", false},

	// SMS / Messaging
	{"https://vonage.com", false},
	{"https://messagebird.com", false},
	{"https://sinch.com", false},
	{"https://plivo.com", false},

	// Video / Conferencing
	{"https://daily.co", false},
	{"https://100ms.live", false},
	{"https://livekit.io", false},
	{"https://whereby.com", false},
	{"https://zoom.us/developers", false},

	// OCR / Document AI
	{"https://ocr.space", false},
	{"https://mindee.com", false},
	{"https://nanonets.com", false},
	{"https://rossum.ai", false},

	// Scraping / Browser automation
	{"https://scrapingbee.com", false},
	{"https://apify.com", false},
	{"https://crawlee.dev", false},
	{"https://firecrawl.dev", false},
	{"https://brightdata.com", false},

	// Code / Development
	{"https://replit.com", false},
	{"https://codesandbox.io", false},
	{"https://stackblitz.com", false},
	{"https://sourcegraph.com", false},
	{"https://codeium.com", false},
	{"https://cursor.com", false},

	// MCP Infrastructure
	{"https://mcpservers.org", false},
	{"https://mcpmarket.com", false},
	{"https://stagehand.dev", false},

	// AI Agent Observability / LLMOps
	{"https://agentops.ai", false},
	{"https://langfuse.com", false},
	{"https://helicone.ai", false},
	{"https://braintrust.dev", false},
	{"https://arize.com", false},
	{"https://openpipe.ai", false},
	{"https://galileo.ai", false},

	// AI Agent Orchestration
	{"https://langflow.org", false},
	{"https://flowiseai.com", false},
	{"https://botpress.com", false},
	{"https://voiceflow.com", false},
	{"https://activepieces.com", false},
	{"https://dify.ai", false},
	{"https://agno.ai", false},
	{"https://mastra.ai", false},
	{"https://vellum.ai", false},

	// AI Search APIs
	{"https://brave.com/search/api", false},
	{"https://you.com", false},

	// Document AI / OCR
	{"https://reducto.ai", false},
	{"https://mathpix.com", false},

	// Voice / Speech APIs
	{"https://cartesia.ai", false},
	{"https://lmnt.com", false},

	// Image / Video / 3D Generation
	{"https://fal.ai", false},
	{"https://pika.art", false},
	{"https://tripo3d.ai", false},

	// Music / Audio APIs
	{"https://loudly.com", false},
	{"https://lalal.ai", false},
	{"https://mubert.com", false},

	// Vector Databases
	{"https://milvus.io", false},
	{"https://zilliz.com", false},

	// Real Estate APIs
	{"https://attomdata.com", false},
	{"https://rentcast.io", false},
	{"https://housecanary.com", false},
	{"https://estated.com", false},

	// Legal Tech APIs
	{"https://sec-api.io", false},
	{"https://courtlistener.org", false},

	// Sports Data APIs
	{"https://sportradar.com", false},
	{"https://sportsdata.io", false},
	{"https://api-sports.io", false},
	{"https://sportmonks.com", false},
	{"https://thesportsdb.com", false},

	// Wearables / Fitness APIs
	{"https://tryterra.co", false},
	{"https://sahha.ai", false},

	// Food / Nutrition APIs
	{"https://spoonacular.com", false},
	{"https://edamam.com", false},
	{"https://nutritionix.com", false},

	// Environmental / Climate APIs
	{"https://climatiq.io", false},
	{"https://open-meteo.com", false},
	{"https://getambee.com", false},

	// Supply Chain / Freight APIs
	{"https://flexport.com", false},
	{"https://shipengine.com", false},
	{"https://fleetbase.io", false},

	// Bioinformatics APIs
	{"https://uniprot.org", false},
	{"https://rcsb.org", false},

	// Fintech / Banking Infrastructure
	{"https://moderntreasury.com", false},
	{"https://moov.io", false},
	{"https://lithic.com", false},
	{"https://column.com", false},
	{"https://increase.com", false},
	{"https://mangopay.com", false},
	{"https://getlago.com", false},
	{"https://tryfinch.com", false},

	// Identity / KYC / Fraud APIs
	{"https://withpersona.com", false},
	{"https://onfido.com", false},
	{"https://socure.com", false},
	{"https://alloy.com", false},

	// Data Enrichment / B2B APIs
	{"https://clay.com", false},
	{"https://apollo.io", false},
	{"https://peopledatalabs.com", false},
	{"https://proxycurl.com", false},

	// Healthcare APIs
	{"https://www.metriport.com", false},

	// Cybersecurity / Threat Intel
	{"https://shodan.io", false},
	{"https://greynoise.io", false},

	// Maps / Geospatial
	{"https://protomaps.com", false},
	{"https://overturemaps.org", false},
	{"https://felt.com", false},

	// Travel APIs
	{"https://duffel.com", false},
	{"https://kiwi.com", false},

	// HR / Payroll Unified APIs
	{"https://merge.dev", false},

	// E-commerce Product Data APIs
	{"https://serpapi.com", false},

	// Email (newer)
	{"https://loops.so", false},

	// Agentic Infra / Code Execution
	{"https://www.browsercat.com", false},

	// llms.txt early adopters
	{"https://mintlify.com", false},
	{"https://tinybird.co", false},
	{"https://flatfile.com", false},
	{"https://plain.com", false},
	{"https://inkeep.com", false},
	{"https://axiom.co", false},
	{"https://openphone.com", false},
	{"https://smartcar.com", false},
	{"https://stedi.com", false},
	{"https://infisical.com", false},
	{"https://screenshotone.com", false},
	{"https://buildwithfern.com", false},
	{"https://tryvital.io", false},
	{"https://projectdiscovery.io", false},
	{"https://conductor.is", false},
	{"https://ionq.com", false},

	// llms.txt confirmed (batch 2 — 2026-04-13)
	{"https://speakeasy.com", false},
	{"https://scalar.com", false},
	{"https://readme.com", false},
	{"https://dub.co", false},
	{"https://writer.com", false},
	{"https://frigade.com", false},
	{"https://basehub.com", false},
	{"https://openpipe.ai", false},
	{"https://dotenvx.com", false},
	{"https://datafold.com", false},
	{"https://dynamic.xyz", false},
	{"https://velt.dev", false},
	{"https://salesbricks.com", false},
	{"https://hyperline.co", false},
	{"https://aporia.com", false},
	{"https://pinata.cloud", false},
	{"https://wordlift.io", false},
	{"https://micro1.ai", false},
	{"https://campsite.com", false},

	// MCP server providers
	{"https://mastra.ai", false},
	{"https://portkey.ai", false},
	{"https://context7.com", false},
	{"https://stainlessapi.com", false},

	// MCP directories and registries
	{"https://pulsemcp.com", false},
	{"https://mcp.so", false},
	{"https://opentools.com", false},

	// llms.txt directories
	{"https://llmstxthub.com", false},

	// llms.txt verified batch 3 (from directory.llmstxt.cloud, 2026-04-14)
	{"https://play.ai", false},           // AI voice agents
	{"https://svelte.dev", false},         // web framework
	{"https://answer.ai", false},          // AI research
	{"https://fastht.ml", false},          // Python web framework
	{"https://ongoody.com", false},        // gifting API
	{"https://embedchain.ai", false},      // RAG framework
	{"https://argil.ai", false},           // AI video
	{"https://axle.insure", false},        // insurance verification API
	{"https://unifygtm.com", false},       // sales automation
	{"https://fabric.inc", false},         // ecommerce platform
	{"https://meshconnect.com", false},    // fintech aggregation
	{"https://skip.build", false},         // dev tools
	{"https://flowx.ai", false},           // workflow automation
	{"https://solidfi.com", false},        // banking as a service API
	{"https://cobo.com", false},           // crypto custody API
	{"https://dopp.finance", false},       // DeFi API
	{"https://sardine.ai", false},         // fraud detection API
	{"https://oxla.com", false},           // analytics database
	{"https://aptible.com", false},        // secure cloud hosting
	{"https://rubric.com", false},         // AI evaluation
	{"https://sitespeak.ai", false},       // accessibility API

	// llms.txt verified batch 4 (from llmstxt.site, 2026-04-14)
	{"https://docs.adyen.com", false},     // payments API
	{"https://uploadcare.com", false},     // file upload API
	{"https://configcat.com", false},      // feature flags API
	{"https://mariadb.com", false},        // database
	{"https://hydrolix.io", false},        // streaming analytics
	{"https://printify.com", false},       // print-on-demand API
	{"https://readwise.io", false},        // reading/highlights API
	{"https://nuxt.com", false},           // web framework
	{"https://nextjs.org", false},         // web framework
	{"https://www.postman.com", false},    // API testing platform
	{"https://developer.nvidia.com", false}, // GPU/AI platform
	{"https://docs.retool.com", false},    // internal tools platform
	{"https://www.dreamhost.com", false},  // hosting
	{"https://vite.dev", false},           // build tool
	{"https://www.nextiva.com", false},    // communications platform
	{"https://claspo.io", false},          // widget builder API
	{"https://brandefense.io", false},     // cybersecurity platform
}
