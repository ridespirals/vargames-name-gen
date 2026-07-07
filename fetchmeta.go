package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"vargames-name-gen/src/igdb"
)

// fetchMeta is machine-readable metadata written after each entity fetch.
type fetchMeta struct {
	Entity         string    `json:"entity"`
	FetchedAt      time.Time `json:"fetched_at"`
	CountReported  int       `json:"count_reported"`
	CountFetched   int       `json:"count_fetched"`
	Limit          int       `json:"limit"`
	MaxConcurrent  int       `json:"max_concurrent"`
	DurationMs     int64     `json:"duration_ms"`
	PagesOK        int       `json:"pages_ok,omitempty"`
	PagesFailed    int       `json:"pages_failed,omitempty"`
	PagesTotal     int       `json:"pages_total,omitempty"`
	Partial        bool      `json:"partial,omitempty"`
	Error          string    `json:"error,omitempty"`
	ChecksumSample *string   `json:"checksum_sample"`
}

func writeEntityResultsJSON(entity string, results []json.RawMessage, dataDir string) (string, error) {
	return writeEntityResultsJSONAt(entity, results, dataDir, false)
}

func writeEntityResultsJSONAt(entity string, results []json.RawMessage, dataDir string, partial bool) (string, error) {
	filename := entity + ".json"
	if partial {
		filename = entity + ".partial.json"
	}
	outPath := filepath.Join(dataDir, filename)
	raw, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("marshal %s: %w", entity, err)
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir %s: %w", dataDir, err)
	}
	if err := os.WriteFile(outPath, raw, 0644); err != nil {
		return "", fmt.Errorf("write %s: %w", outPath, err)
	}
	return outPath, nil
}

func writeFetchMeta(entity string, res igdb.FetchResult, wallClock time.Duration, limit, maxConcurrent int, dataDir string) error {
	meta := fetchMeta{
		Entity:        entity,
		FetchedAt:     time.Now().UTC(),
		CountReported: res.CountReported,
		CountFetched:  len(res.Items),
		Limit:         limit,
		MaxConcurrent: maxConcurrent,
		DurationMs:    wallClock.Milliseconds(),
		PagesOK:       res.PagesOK,
		PagesFailed:   res.PagesFailed,
		PagesTotal:    res.PagesTotal,
		Partial:       res.Err != nil && len(res.Items) > 0,
	}
	if res.Err != nil {
		meta.Error = res.Err.Error()
	}
	path := filepath.Join(dataDir, entity+"-meta.json")
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal meta %s: %w", entity, err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dataDir, err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func formatFetchSummary(entity string, res igdb.FetchResult) string {
	if res.Err == nil {
		return fmt.Sprintf("%s: OK %d items", entity, len(res.Items))
	}
	if len(res.Items) > 0 {
		if res.PagesTotal > 0 {
			return fmt.Sprintf("%s: PARTIAL %d items (%d/%d pages failed)", entity, len(res.Items), res.PagesFailed, res.PagesTotal)
		}
		return fmt.Sprintf("%s: PARTIAL %d items (%v)", entity, len(res.Items), res.Err)
	}
	if res.PagesTotal > 0 {
		return fmt.Sprintf("%s: FAILED (%d/%d pages failed)", entity, res.PagesFailed, res.PagesTotal)
	}
	return fmt.Sprintf("%s: FAILED (%v)", entity, res.Err)
}
