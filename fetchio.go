package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"vargames-name-gen/src/igdb"
)

func loadExistingEntityJSON(entity, dataDir string) ([]json.RawMessage, error) {
	path := filepath.Join(dataDir, entity+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return items, nil
}

func loadPriorFetchMeta(entity, dataDir string) (*fetchMeta, error) {
	path := filepath.Join(dataDir, entity+"-meta.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var meta fetchMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func writeChecksumsJSON(entity string, items []json.RawMessage, dataDir string) error {
	records := make([]idChecksumRecord, 0, len(items))
	for _, raw := range items {
		id, checksum, err := igdb.ParseIDChecksum(raw)
		if err != nil {
			continue
		}
		records = append(records, idChecksumRecord{ID: id, Checksum: checksum})
	}
	path := filepath.Join(dataDir, entity+"-checksums.json")
	raw, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal checksums: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}

type idChecksumRecord struct {
	ID       int    `json:"id"`
	Checksum string `json:"checksum"`
}
