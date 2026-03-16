package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"vargames-name-gen/src/config"
	"vargames-name-gen/src/igdb"
)

const dataDir = "data"

func main() {
	verbose := flag.Bool("verbose", false, "enable progress logging for IGDB client and fetchers")
	fetchEntity := flag.String("fetch", "", "fetch entity and save to data/<entity>.json (e.g. -fetch=games, -fetch=alternative_names)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v (set %s and %s, e.g. from .env)", err, config.EnvClientID, config.EnvClientSecret)
	}
	enableLog := *verbose || cfg.Verbose
	logger := igdb.LoggerFromVerbose(enableLog)
	client := igdb.NewClient(cfg, igdb.WithLogger(logger))

	if *fetchEntity != "" {
		if !igdb.ValidEntity(*fetchEntity) {
			log.Fatalf("unknown entity %q; valid: games, characters, genres, platforms, collections, companies, alternative_names", *fetchEntity)
		}
		entity := igdb.Entity(*fetchEntity)
		metrics := igdb.NewMetrics(*fetchEntity)
		client = igdb.NewClient(cfg, igdb.WithLogger(logger), igdb.WithMetrics(metrics))
		fetcher := igdb.NewFetcher(client, entity, igdb.FetcherOptions{
			Limit:         0, // use config MaxLimit
			MaxPages:      500,
			MaxConcurrent: 4,
			Logger:        logger,
		})
		ctx := context.Background()
		start := time.Now()
		results, err := fetcher.FetchAll(ctx)
		wallClock := time.Since(start)
		if err != nil {
			log.Fatalf("fetch %s: %v", entity, err)
		}
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("mkdir %s: %v", dataDir, err)
		}
		outPath := filepath.Join(dataDir, *fetchEntity+".json")
		raw, err := json.Marshal(results)
		if err != nil {
			log.Fatalf("marshal: %v", err)
		}
		if err := os.WriteFile(outPath, raw, 0644); err != nil {
			log.Fatalf("write %s: %v", outPath, err)
		}
		reportPath := filepath.Join(dataDir, *fetchEntity+"-report.html")
		if err := writeFetchReport(metrics, wallClock, len(results), reportPath); err != nil {
			log.Printf("warning: could not write report: %v", err)
		} else {
			log.Printf("report written to %s", reportPath)
		}
		log.Printf("wrote %d items to %s", len(results), outPath)
		return
	}

	if enableLog {
		log.Println("verbose logging enabled (IGDB client/fetchers)")
	}
	_ = client
	fmt.Printf("Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n", cfg.BaseURL)
}

