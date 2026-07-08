package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"vargames-name-gen/src/cli"
	"vargames-name-gen/src/config"
	"vargames-name-gen/src/igdb"
)

const dataDir = "data"

type entityFetchOutcome struct {
	entity        string
	result        igdb.FetchResult
	wallClock     time.Duration
	metrics       *igdb.Metrics
	limit         int
	maxConcurrent int
	profileName   string
	incremental   bool
	incStats      igdb.IncrementalStats
	priorMeta     *fetchMeta
}

// parseEntityList splits a comma-separated list and returns non-empty trimmed entries.
func parseEntityList(s string) []string {
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		e := strings.TrimSpace(part)
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}

func main() {
	// Generation subcommand does not require IGDB credentials.
	if cli.IsGenerateCommand(os.Args[1:]) {
		if err := cli.RunGenerate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	verbose := flag.Bool("verbose", false, "enable progress logging for IGDB client and fetchers")
	fetchEntities := flag.String("fetch", "", "fetch entity/entities (comma-separated) and save to data/<entity>.json (e.g. -fetch=games, -fetch=games,genres,platforms)")
	fetchLimit := flag.Int("fetch-limit", 0, "page size per IGDB request (default from config or IGDB_MAX_LIMIT)")
	fetchConcurrent := flag.Int("fetch-concurrent", 0, "parallel pages within one entity fetch (default from config or IGDB_MAX_CONCURRENT)")
	fetchPartial := flag.Bool("partial", false, "continue on entity/page failures; write .partial.json when some pages succeed")
	fetchProfile := flag.String("fetch-profile", "", "Apicalypse field profile: full, minimal (default), or checksum")
	fetchIncremental := flag.Bool("incremental", false, "re-fetch only changed rows via checksum scan (requires existing data/<entity>.json for updates)")
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

		profileName := strings.TrimSpace(*fetchProfile)
		if profileName == "" {
			profileName = cfg.FetchProfile
		}
		if !igdb.ValidProfileName(profileName) {
			log.Fatalf("unknown fetch profile %q (valid: full, minimal, checksum)", profileName)
		}

		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("mkdir %s: %v", dataDir, err)
		}
		ctx := context.Background()
		start := time.Now()
		outcomes := make([]entityFetchOutcome, 0, len(entities))
		var outMu sync.Mutex
		var wg sync.WaitGroup
		for _, entityStr := range entities {
			wg.Go(func() {
				metrics := igdb.NewMetrics(entityStr)
				client := igdb.NewClient(cfg, igdb.WithLogger(logger), igdb.WithMetrics(metrics))
				limit := *fetchLimit
				if limit <= 0 {
					limit = cfg.MaxLimit
				}
				maxConcurrent := *fetchConcurrent
				if maxConcurrent <= 0 {
					maxConcurrent = cfg.MaxConcurrent
				}

				profile, err := igdb.ProfileFor(igdb.Entity(entityStr), profileName)
				if err != nil {
					outMu.Lock()
					outcomes = append(outcomes, entityFetchOutcome{
						entity: entityStr,
						result: igdb.FetchResult{
							Entity: igdb.Entity(entityStr),
							Err:    err,
						},
					})
					outMu.Unlock()
					return
				}

				priorMeta, _ := loadPriorFetchMeta(entityStr, dataDir)
				if priorMeta != nil && priorMeta.CountReported > 0 && enableLog {
					logger.Logf("fetch %s: prior count_reported=%d profile=%s", entityStr, priorMeta.CountReported, priorMeta.FetchProfile)
				}

				fetcher := igdb.NewFetcher(client, igdb.Entity(entityStr), igdb.FetcherOptions{
					Limit:         limit,
					MaxConcurrent: maxConcurrent,
					QueryPrefix:   profile.QueryPrefix,
					Logger:        logger,
				})
				runStart := time.Now()

				var result igdb.FetchResult
				var incStats igdb.IncrementalStats
				if *fetchIncremental {
					existing, err := loadExistingEntityJSON(entityStr, dataDir)
					if err != nil {
						result = igdb.FetchResult{Entity: igdb.Entity(entityStr), Err: err}
					} else {
						dataProfile, err := igdb.ProfileFor(igdb.Entity(entityStr), igdb.ProfileMinimal)
						if err != nil {
							result = igdb.FetchResult{Entity: igdb.Entity(entityStr), Err: err}
						} else {
							result, incStats, _ = fetcher.FetchIncrementalResult(ctx, existing, dataProfile, *fetchPartial)
						}
					}
				} else {
					result = fetcher.FetchAllResult(ctx, *fetchPartial)
				}

				if priorMeta != nil && result.CountReported > 0 && priorMeta.CountReported != result.CountReported {
					log.Printf("fetch %s: count changed %d -> %d", entityStr, priorMeta.CountReported, result.CountReported)
				}

				outMu.Lock()
				outcomes = append(outcomes, entityFetchOutcome{
					entity:        entityStr,
					result:        result,
					wallClock:     time.Since(runStart),
					metrics:       metrics,
					limit:         limit,
					maxConcurrent: maxConcurrent,
					profileName:   profile.Name,
					incremental:   *fetchIncremental,
					incStats:      incStats,
					priorMeta:     priorMeta,
				})
				outMu.Unlock()
			})
		}
		wg.Wait()

		hadFailure := false
		for _, out := range outcomes {
			log.Print(formatFetchSummaryDetailed(out))

			if err := writeFetchMeta(out.entity, out.result, out.wallClock, out.limit, out.maxConcurrent, out.profileName, out.incremental, out.incStats, out.priorMeta, dataDir); err != nil {
				log.Printf("warning: could not write meta for %s: %v", out.entity, err)
			}

			if out.result.Err != nil && len(out.result.Items) == 0 {
				hadFailure = true
				reportPath := filepath.Join(dataDir, out.entity+"-report.html")
				if err := writeFetchReport(out.metrics, out.wallClock, 0, reportPath); err != nil {
					log.Printf("warning: could not write report for %s: %v", out.entity, err)
				}
				continue
			}

			partialFile := out.result.Err != nil
			outPath, err := writeEntityResultsJSONAt(out.entity, out.result.Items, dataDir, partialFile)
			if err != nil {
				log.Printf("%s: FAILED writing output: %v", out.entity, err)
				hadFailure = true
				continue
			}
			if out.result.Err != nil {
				hadFailure = true
				log.Printf("wrote %d items to %s (partial)", len(out.result.Items), outPath)
			} else {
				log.Printf("wrote %d items to %s", len(out.result.Items), outPath)
			}

			if out.incremental {
				if err := writeChecksumsJSON(out.entity, out.result.Items, dataDir); err != nil {
					log.Printf("warning: could not write checksums for %s: %v", out.entity, err)
				}
			}

			reportPath := filepath.Join(dataDir, out.entity+"-report.html")
			if err := writeFetchReport(out.metrics, out.wallClock, len(out.result.Items), reportPath); err != nil {
				log.Printf("warning: could not write report for %s: %v", out.entity, err)
			} else if enableLog {
				log.Printf("report written to %s", reportPath)
			}
		}

		log.Printf("fetched %d entities in %s", len(entities), time.Since(start).Round(time.Millisecond))
		if hadFailure {
			os.Exit(1)
		}
		return
	}

	if enableLog {
		log.Println("verbose logging enabled (IGDB client/fetchers)")
	}
	client := igdb.NewClient(cfg, igdb.WithLogger(logger))
	_ = client
	fmt.Printf("Hello from vargames-name-gen (IGDB config loaded, base URL: %s)\n", cfg.BaseURL)
}

func formatFetchSummaryDetailed(out entityFetchOutcome) string {
	base := formatFetchSummary(out.entity, out.result)
	if !out.incremental {
		return base
	}
	if out.incStats.FullRefetch {
		return base + " (incremental: full refetch, no prior corpus)"
	}
	return fmt.Sprintf("%s (incremental: %d updated, %d new, %d removed, %d unchanged)",
		base, out.incStats.Updated, out.incStats.New, out.incStats.Removed, out.incStats.Unchanged)
}
