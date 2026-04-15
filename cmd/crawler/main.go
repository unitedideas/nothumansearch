package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/models"
)

func main() {
	seed := flag.Bool("seed", false, "Crawl seed sites")
	recrawl := flag.Bool("recrawl", false, "Re-crawl all sites in the database")
	recategorize := flag.Bool("recategorize", false, "Re-apply categorize() + tags to all DB sites (no HTTP)")
	url := flag.String("url", "", "Crawl a single URL")
	file := flag.String("file", "", "Crawl URLs from a file (one per line)")
	dryRun := flag.Bool("dry-run", false, "Crawl but don't save to DB")
	workers := flag.Int("workers", 5, "Number of concurrent crawlers")
	flag.Parse()

	if !*dryRun {
		if err := database.Connect(); err != nil {
			log.Fatalf("database: %v", err)
		}
		log.Println("connected to database")

		// Run migrations
		migrationsDir := "migrations"
		if root := os.Getenv("APP_ROOT"); root != "" {
			migrationsDir = root + "/migrations"
		}
		if err := database.RunMigrations(migrationsDir); err != nil {
			log.Printf("WARNING: migration: %v", err)
		}
		// Belt-and-braces: ensure favicon columns exist regardless of file-based migration state.
		if _, err := database.DB.Exec(`ALTER TABLE sites ADD COLUMN IF NOT EXISTS has_favicon BOOLEAN DEFAULT FALSE`); err != nil {
			log.Printf("ensure has_favicon: %v", err)
		} else {
			log.Println("ensured column: has_favicon")
		}
		if _, err := database.DB.Exec(`ALTER TABLE sites ADD COLUMN IF NOT EXISTS favicon_url TEXT DEFAULT ''`); err != nil {
			log.Printf("ensure favicon_url: %v", err)
		} else {
			log.Println("ensured column: favicon_url")
		}
	}

	if *url != "" {
		crawlOne(*url, *dryRun)
		return
	}

	if *file != "" {
		crawlFile(*file, *workers, *dryRun)
		return
	}

	if *seed {
		crawlSeeds(*workers, *dryRun)
		return
	}

	if *recrawl {
		recrawlAll(*workers)
		return
	}

	if *recategorize {
		recategorizeAll()
		return
	}

	fmt.Println("Usage:")
	fmt.Println("  crawler -seed              Crawl all seed sites")
	fmt.Println("  crawler -recrawl           Re-crawl all DB sites")
	fmt.Println("  crawler -recategorize      Re-apply categorize() + tags (no HTTP)")
	fmt.Println("  crawler -url https://...   Crawl a single URL")
	fmt.Println("  crawler -file urls.txt     Crawl URLs from file")
	fmt.Println("  crawler -dry-run -seed     Crawl without saving")
}

// recategorizeAll re-applies the current categorize() and generateTags() rules
// to every site in the DB in a single pass — no HTTP, no re-scoring. Useful
// after adding new domainRules/keyword rules when recrawl is slow or blocked.
func recategorizeAll() {
	rows, err := database.DB.Query(`SELECT id, domain, name, description, category,
		has_llms_txt, has_ai_plugin, has_openapi, has_robots_ai, has_structured_api, has_mcp_server, has_schema_org
		FROM sites`)
	if err != nil {
		log.Fatalf("query sites: %v", err)
	}
	defer rows.Close()

	type row struct {
		id       string
		oldCat   string
		site     models.Site
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.site.Domain, &r.site.Name, &r.site.Description, &r.oldCat,
			&r.site.HasLLMsTxt, &r.site.HasAIPlugin, &r.site.HasOpenAPI, &r.site.HasRobotsAI,
			&r.site.HasStructuredAPI, &r.site.HasMCPServer, &r.site.HasSchemaOrg); err != nil {
			log.Printf("scan: %v", err)
			continue
		}
		all = append(all, r)
	}
	log.Printf("Recategorizing %d sites...", len(all))

	var changed, unchanged, failed int
	moves := map[string]int{}
	for _, r := range all {
		newCat := crawler.Categorize(&r.site)
		newTags := crawler.GenerateTags(&r.site)
		if newTags == nil {
			newTags = pq.StringArray{}
		}
		if newCat == r.oldCat {
			unchanged++
			// still update tags in case they changed
			if _, err := database.DB.Exec(`UPDATE sites SET tags=$1, updated_at=NOW() WHERE id=$2`, newTags, r.id); err != nil {
				log.Printf("UPDATE %s tags: %v", r.site.Domain, err)
				failed++
			}
			continue
		}
		if _, err := database.DB.Exec(`UPDATE sites SET category=$1, tags=$2, updated_at=NOW() WHERE id=$3`, newCat, newTags, r.id); err != nil {
			log.Printf("UPDATE %s: %v", r.site.Domain, err)
			failed++
			continue
		}
		changed++
		key := r.oldCat + " -> " + newCat
		moves[key]++
		log.Printf("  %s: %s -> %s", r.site.Domain, r.oldCat, newCat)
	}
	log.Printf("Done. Changed: %d, Unchanged: %d, Failed: %d", changed, unchanged, failed)
	for k, v := range moves {
		log.Printf("  %s: %d", k, v)
	}
}

func crawlOne(rawURL string, dryRun bool) {
	if !strings.HasPrefix(rawURL, "http") {
		rawURL = "https://" + rawURL
	}

	log.Printf("Crawling %s...", rawURL)
	site, err := crawler.CrawlSite(rawURL)
	if err != nil {
		log.Fatalf("crawl error: %v", err)
	}

	printSite(site)

	if !dryRun {
		if err := models.UpsertSite(database.DB, site); err != nil {
			log.Fatalf("save error: %v", err)
		}
		log.Println("Saved to database")
	}
}

func crawlFile(path string, numWorkers int, dryRun bool) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("open file: %v", err)
	}
	defer f.Close()

	var urls []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "http") {
			line = "https://" + line
		}
		urls = append(urls, line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("read file: %v", err)
	}

	log.Printf("Crawling %d URLs from %s with %d workers...", len(urls), path, numWorkers)

	jobs := make(chan string, len(urls))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var success, failed int

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range jobs {
				site, err := crawler.CrawlSite(u)
				if err != nil {
					mu.Lock()
					failed++
					mu.Unlock()
					log.Printf("FAIL: %s: %v", u, err)
					continue
				}
				if !dryRun {
					if err := models.UpsertSite(database.DB, site); err != nil {
						log.Printf("SAVE FAIL: %s: %v", u, err)
						mu.Lock()
						failed++
						mu.Unlock()
						continue
					}
				}
				mu.Lock()
				success++
				mu.Unlock()
				printSite(site)
			}
		}()
	}

	for _, u := range urls {
		jobs <- u
	}
	close(jobs)
	wg.Wait()

	log.Printf("Done. Success: %d, Failed: %d, Total: %d", success, failed, len(urls))
}

func crawlSeeds(numWorkers int, dryRun bool) {
	seeds := crawler.SeedSites
	log.Printf("Crawling %d seed sites with %d workers...", len(seeds), numWorkers)

	type job struct {
		url      string
		featured bool
	}

	jobs := make(chan job, len(seeds))
	var wg sync.WaitGroup

	var mu sync.Mutex
	var success, failed int

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				site, err := crawler.CrawlSite(j.url)
				if err != nil {
					mu.Lock()
					failed++
					mu.Unlock()
					log.Printf("FAIL: %s: %v", j.url, err)
					continue
				}
				site.IsFeatured = j.featured

				if !dryRun {
					if err := models.UpsertSite(database.DB, site); err != nil {
						log.Printf("SAVE FAIL: %s: %v", j.url, err)
						mu.Lock()
						failed++
						mu.Unlock()
						continue
					}
				}

				mu.Lock()
				success++
				mu.Unlock()
				printSite(site)
			}
		}()
	}

	for _, s := range seeds {
		jobs <- job{url: s.URL, featured: s.Featured}
	}
	close(jobs)

	wg.Wait()

	log.Printf("Done. Success: %d, Failed: %d, Total: %d", success, failed, len(seeds))
}

func recrawlAll(numWorkers int) {
	// First, process any pending submissions — parallelized to match the recrawl
	// worker count. Serial processing made 500 submissions take ~25 min at 3s/site,
	// blocking the rest of the recrawl.
	rows, err := database.DB.Query("SELECT url FROM submissions WHERE status='pending' LIMIT 500")
	if err == nil {
		var pendingURLs []string
		for rows.Next() {
			var u string
			rows.Scan(&u)
			pendingURLs = append(pendingURLs, u)
		}
		rows.Close()
		if len(pendingURLs) > 0 {
			log.Printf("Processing %d pending submissions with %d workers...", len(pendingURLs), numWorkers)
			submitCh := make(chan string, len(pendingURLs))
			var subWG sync.WaitGroup
			for w := 0; w < numWorkers; w++ {
				subWG.Add(1)
				go func() {
					defer subWG.Done()
					for u := range submitCh {
						site, err := crawler.CrawlSite(u)
						if err != nil {
							log.Printf("FAIL submission %s: %v", u, err)
							database.DB.Exec("UPDATE submissions SET status='failed' WHERE url=$1", u)
							continue
						}
						if err := models.UpsertSite(database.DB, site); err != nil {
							log.Printf("SAVE FAIL %s: %v", u, err)
							continue
						}
						database.DB.Exec("UPDATE submissions SET status='crawled' WHERE url=$1", u)
						printSite(site)
					}
				}()
			}
			for _, u := range pendingURLs {
				submitCh <- u
			}
			close(submitCh)
			subWG.Wait()
		}
	}

	// Then re-crawl all existing sites
	siteRows, err := database.DB.Query("SELECT url, is_featured FROM sites ORDER BY last_crawled_at ASC NULLS FIRST")
	if err != nil {
		log.Fatalf("query sites: %v", err)
	}
	defer siteRows.Close()

	type job struct {
		url      string
		featured bool
	}

	var sites []job
	for siteRows.Next() {
		var j job
		siteRows.Scan(&j.url, &j.featured)
		sites = append(sites, j)
	}

	log.Printf("Re-crawling %d sites with %d workers...", len(sites), numWorkers)

	jobs := make(chan job, len(sites))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var success, failed int

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				site, err := crawler.CrawlSite(j.url)
				if err != nil {
					mu.Lock()
					failed++
					mu.Unlock()
					log.Printf("FAIL: %s: %v", j.url, err)
					continue
				}
				site.IsFeatured = j.featured
				if err := models.UpsertSite(database.DB, site); err != nil {
					log.Printf("SAVE FAIL: %s: %v", j.url, err)
					mu.Lock()
					failed++
					mu.Unlock()
					continue
				}
				mu.Lock()
				success++
				mu.Unlock()
				printSite(site)
			}
		}()
	}

	for _, s := range sites {
		jobs <- s
	}
	close(jobs)
	wg.Wait()

	log.Printf("Done. Success: %d, Failed: %d, Total: %d", success, failed, len(sites))
}

func printSite(s *models.Site) {
	signals := []string{}
	if s.HasLLMsTxt {
		signals = append(signals, "llms.txt")
	}
	if s.HasAIPlugin {
		signals = append(signals, "ai-plugin")
	}
	if s.HasOpenAPI {
		signals = append(signals, "OpenAPI")
	}
	if s.HasStructuredAPI {
		signals = append(signals, "API")
	}
	if s.HasRobotsAI {
		signals = append(signals, "AI-bots")
	}
	if s.HasMCPServer {
		signals = append(signals, "MCP")
	}
	if s.HasSchemaOrg {
		signals = append(signals, "schema.org")
	}

	featured := ""
	if s.IsFeatured {
		featured = " [FEATURED]"
	}

	log.Printf("  %s score=%d cat=%s signals=[%s]%s",
		s.Domain, s.AgenticScore, s.Category,
		strings.Join(signals, ", "), featured)
	_ = time.Now() // prevent unused import
}
