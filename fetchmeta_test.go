package main

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"vargames-name-gen/src/igdb"
)

func TestWriteEntityResultsJSONAt_PartialFilename(t *testing.T) {
	tmp := t.TempDir()
	results := []json.RawMessage{json.RawMessage(`{"id":1}`)}

	outPath, err := writeEntityResultsJSONAt("games", results, tmp, true)
	if err != nil {
		t.Fatalf("writeEntityResultsJSONAt: %v", err)
	}
	if outPath != tmp+"/games.partial.json" {
		t.Fatalf("got path %q", outPath)
	}
}

func TestWriteFetchMeta(t *testing.T) {
	tmp := t.TempDir()
	res := igdb.FetchResult{
		CountReported: 100,
		Items:         []json.RawMessage{json.RawMessage(`{"id":1}`)},
		PagesOK:       1,
		PagesTotal:    1,
	}
	if err := writeFetchMeta("games", res, 1500*time.Millisecond, 500, 4, tmp); err != nil {
		t.Fatalf("writeFetchMeta: %v", err)
	}
	raw, err := os.ReadFile(tmp + "/games-meta.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var meta fetchMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if meta.Entity != "games" || meta.CountFetched != 1 || meta.Limit != 500 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestFormatFetchSummary(t *testing.T) {
	ok := formatFetchSummary("genres", igdb.FetchResult{Items: make([]json.RawMessage, 23)})
	if ok != "genres: OK 23 items" {
		t.Fatalf("got %q", ok)
	}
}
