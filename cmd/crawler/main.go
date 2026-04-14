package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/unitedideas/nothumansearch/internal/crawler"
	"github.com/unitedideas/nothumansearch/internal/database"
	"github.com/unitedideas/nothumansearch/internal/models"
)

func main() {
	seed := flag.Bool("seed", false, "Crawl seed sites")
	recrawl := flag.Bool("recrawl", false, "Re-crawl all sites in the database")
	url := flag.String("url", "", "Crawl a single URL")
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
	}

	if *url != "" {
		crawlOne(*url, *dryRun)
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

	fmt.Println("Usage:")
	fmt.Println("  crawler -seed              Crawl all seed sites")
	fmt.Println("  crawler -recrawl           Re-crawl all DB sites")
	fmt.Println("  crawler -url https://...   Crawl a single URL")
	fmt.Println("  crawler -dry-run -seed     Crawl without saving")
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
	// First, process any pending submissions
	rows, err := database.DB.Query("SELECT url FROM submissions WHERE status='pending' LIMIT 50")
	if err == nil {
		defer rows.Close()
		var pendingURLs []string
		for rows.Next() {
			var u string
			rows.Scan(&u)
			pendingURLs = append(pendingURLs, u)
		}
		if len(pendingURLs) > 0 {
			log.Printf("Processing %d pending submissions...", len(pendingURLs))
			for _, u := range pendingURLs {
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
