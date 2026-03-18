package main

import (
	"encoding/json"
	"os"
	"testing"
)

const (
	alternativeNamesExpectedCount = 1000
)

func TestWriteEntityResultsJSON_AlternativeNames(t *testing.T) {
	// Unit test only: avoids any HTTP/IGDB calls by using fake in-memory results.
	tmp := t.TempDir()
	entity := "alternative_names"

	results := make([]json.RawMessage, alternativeNamesExpectedCount)
	for i := range results {
		results[i] = nil
	}

	outPath, err := writeEntityResultsJSON(entity, results, tmp)
	if err != nil {
		t.Fatalf("writeEntityResultsJSON: %v", err)
	}
	if outPath == "" {
		t.Fatalf("expected non-empty outPath")
	}

	// Verify the file content length.
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var decoded []json.RawMessage
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(decoded) != alternativeNamesExpectedCount {
		t.Fatalf("got %d decoded items, want %d", len(decoded), alternativeNamesExpectedCount)
	}
}
