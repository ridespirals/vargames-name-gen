package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vargames-name-gen/src/config"
	"vargames-name-gen/src/igdb"
)

const dataDir = "data"

// parseEntityList splits a comma-separated list and returns non-empty trimmed entries.
func parseEntityList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		e := strings.TrimSpace(part)
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}

func main() {
	verbose := flag.Bool("verbose", false, "enable progress logging for IGDB client and fetchers")
	fetchEntities := flag.String("fetch", "", "fetch entity/entities (comma-separated) and save to data/<entity>.json (e.g. -fetch=games, -fetch=games,genres,platforms)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v (set %s and %s, e.g. from .env)", err, config.EnvClientID, config.EnvClientSecret)
	}
	enableLog := *verbose || cfg.Verbose
	logger := igdb.LoggerFromVerbose(enableLog)

	if *fetchEntities != "" {
		entities := parseEntityList(*fetchEntities)
		if len(entities) == 0 {
			log.Fatal("no valid entities in -fetch list")
		}
		for _, e := range entities {
			if !igdb.ValidEntity(e) {
				log.Fatalf("unknown entity %q; valid: games, characters, genres, platforms, collections, companies, alternative_names", e)
			}
		}
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("mkdir %s: %v", dataDir, err)
		}
		ctx := context.Background()
		start := time.Now()
		var errMu sync.Mutex
		var firstErr error
		var wg sync.WaitGroup
		for _, entityStr := range entities {
			// capture local copy of entityStr to avoid race condition
			entityStr := entityStr
			wg.Add(1)
			go func() {
				defer wg.Done()
				metrics := igdb.NewMetrics(entityStr)
				client := igdb.NewClient(cfg, igdb.WithLogger(logger), igdb.WithMetrics(metrics))
				fetcher := igdb.NewFetcher(client, igdb.Entity(entityStr), igdb.FetcherOptions{
					Limit:         0,
					MaxConcurrent: 4,
					Logger:        logger,
				})
				runStart := time.Now()
				results, err := fetcher.FetchAll(ctx)
				wallClock := time.Since(runStart)
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("fetch %s: %w", entityStr, err)
					}
					errMu.Unlock()
					return
				}
				outPath := filepath.Join(dataDir, entityStr+".json")
				raw, err := json.Marshal(results)
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("marshal %s: %w", entityStr, err)
					}
					errMu.Unlock()
					return
				}
				if err := os.WriteFile(outPath, raw, 0644); err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("write %s: %w", outPath, err)
					}
					errMu.Unlock()
					return
				}
				reportPath := filepath.Join(dataDir, entityStr+"-report.html")
				if err := writeFetchReport(metrics, wallClock, len(results), reportPath); err != nil {
					log.Printf("warning: could not write report for %s: %v", entityStr, err)
				} else if enableLog {
					log.Printf("report written to %s", reportPath)
				}
				log.Printf("wrote %d items to %s", len(results), outPath)
			}()
		}
		wg.Wait()
		if firstErr != nil {
			log.Fatalf("%v", firstErr)
		}
		log.Printf("fetched %d entities in %s", len(entities), time.Since(start).Round(time.Millisecond))
		return
	}

	if enableLog {
		log.Println("verbose logging enabled (IGDB client/fetchers)")
	}
	client := igdb.NewClient(cfg, igdb.WithLogger(logger))
	_ = client
	fmt.Printf("Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n", cfg.BaseURL)
}
