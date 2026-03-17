package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"vargames-name-gen/src/config"
	"vargames-name-gen/src/igdb"
)

const (
	alternativeNamesExpectedCount = 195850
	alternativeNamesOutputFile    = "data/alternative_names.json"
)

// TestFetchAlternativeNames fetches all alternative_names from IGDB, verifies the count
// is 195850, and writes the results to data/alternative_names.json.
// Skip with: go test -short
// Requires IGDB credentials in .env or environment.
func TestFetchAlternativeNames(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in -short mode")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("config not available (set IGDB_CLIENT_ID and IGDB_CLIENT_SECRET): %v", err)
	}
	logger := igdb.LoggerFromVerbose(cfg.Verbose)
	client := igdb.NewClient(cfg, igdb.WithLogger(logger))
	fetcher := igdb.NewFetcher(client, igdb.EntityAlternativeNames, igdb.FetcherOptions{
		Limit:  0,
		Logger: logger,
	})
	ctx := context.Background()
	results, err := fetcher.FetchAll(ctx)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(results) != alternativeNamesExpectedCount {
		t.Errorf("got %d alternative_names, want %d", len(results), alternativeNamesExpectedCount)
	}
	raw, err := json.Marshal(results)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("MkdirAll %s: %v", dataDir, err)
	}
	if err := os.WriteFile(alternativeNamesOutputFile, raw, 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", alternativeNamesOutputFile, err)
	}
	t.Logf("wrote %d items to %s", len(results), alternativeNamesOutputFile)
}
