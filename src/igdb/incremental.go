package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// IncrementalStats summarizes an incremental fetch run.
type IncrementalStats struct {
	RemoteCount   int
	ExistingCount int
	Updated       int
	New           int
	Removed       int
	Unchanged     int
	FullRefetch   bool
}

// idChecksumRecord is a minimal IGDB row with id and checksum.
type idChecksumRecord struct {
	ID       int    `json:"id"`
	Checksum string `json:"checksum"`
}

// ParseIDChecksum extracts id and checksum from a raw IGDB JSON object.
func ParseIDChecksum(raw json.RawMessage) (id int, checksum string, err error) {
	var rec idChecksumRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		return 0, "", err
	}
	if rec.ID <= 0 {
		return 0, "", fmt.Errorf("missing or invalid id")
	}
	return rec.ID, rec.Checksum, nil
}

// IDsNeedingFullFetch compares remote checksums to existing records and returns IDs to pull.
func IDsNeedingFullFetch(existing []json.RawMessage, remote map[int]string) (need []int, unchanged int) {
	existingByID := make(map[int]string, len(existing))
	for _, raw := range existing {
		id, checksum, err := ParseIDChecksum(raw)
		if err != nil {
			continue
		}
		existingByID[id] = checksum
	}

	need = make([]int, 0)
	for id, remoteChecksum := range remote {
		localChecksum, ok := existingByID[id]
		if !ok || localChecksum != remoteChecksum {
			need = append(need, id)
			continue
		}
		unchanged++
	}
	return need, unchanged
}

// MergeRecordsByID merges updates into existing, drops IDs absent from remote, sorts by id.
func MergeRecordsByID(existing []json.RawMessage, updates []json.RawMessage, remoteIDs map[int]struct{}) ([]json.RawMessage, error) {
	byID := make(map[int]json.RawMessage, len(existing)+len(updates))

	for _, raw := range existing {
		id, _, err := ParseIDChecksum(raw)
		if err != nil {
			continue
		}
		if _, ok := remoteIDs[id]; ok {
			byID[id] = raw
		}
	}
	for _, raw := range updates {
		id, _, err := ParseIDChecksum(raw)
		if err != nil {
			return nil, fmt.Errorf("update record: %w", err)
		}
		byID[id] = raw
	}

	ids := make([]int, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Ints(ids)

	out := make([]json.RawMessage, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out, nil
}

// FetchIncrementalResult updates a corpus using checksum scan + selective full pulls.
// When existing is empty, performs a full fetch using dataProfile.
func (f *Fetcher) FetchIncrementalResult(ctx context.Context, existing []json.RawMessage, dataProfile Profile, allowPartial bool) (FetchResult, IncrementalStats, error) {
	stats := IncrementalStats{ExistingCount: len(existing)}

	if len(existing) == 0 {
		stats.FullRefetch = true
		fetcher := NewFetcher(f.client, f.entity, FetcherOptions{
			Limit:         f.limit,
			MaxConcurrent: cap(f.sem),
			QueryPrefix:   dataProfile.QueryPrefix,
			Logger:        f.log,
		})
		res := fetcher.FetchAllResult(ctx, allowPartial)
		stats.RemoteCount = len(res.Items)
		stats.New = len(res.Items)
		return res, stats, nil
	}

	checksumFetcher := NewFetcher(f.client, f.entity, FetcherOptions{
		Limit:         f.limit,
		MaxConcurrent: cap(f.sem),
		QueryPrefix:   "fields id,checksum;",
		Logger:        f.log,
	})
	scan := checksumFetcher.FetchAllResult(ctx, allowPartial)
	if scan.Err != nil && len(scan.Items) == 0 {
		return scan, stats, scan.Err
	}

	remote := make(map[int]string, len(scan.Items))
	remoteIDs := make(map[int]struct{}, len(scan.Items))
	for _, raw := range scan.Items {
		id, checksum, err := ParseIDChecksum(raw)
		if err != nil {
			continue
		}
		remote[id] = checksum
		remoteIDs[id] = struct{}{}
	}
	stats.RemoteCount = len(remote)

	needIDs, unchanged := IDsNeedingFullFetch(existing, remote)
	stats.Unchanged = unchanged
	stats.Removed = len(existing) - countExistingInRemote(existing, remoteIDs)

	existingByID := make(map[int]struct{}, len(existing))
	for _, raw := range existing {
		if id, _, err := ParseIDChecksum(raw); err == nil {
			existingByID[id] = struct{}{}
		}
	}
	for _, id := range needIDs {
		if _, ok := existingByID[id]; ok {
			stats.Updated++
		} else {
			stats.New++
		}
	}

	if len(needIDs) == 0 {
		merged, err := MergeRecordsByID(existing, nil, remoteIDs)
		if err != nil {
			return FetchResult{Entity: f.entity, Err: err}, stats, err
		}
		return FetchResult{
			Entity:        f.entity,
			Items:         merged,
			CountReported: scan.CountReported,
			PagesOK:       scan.PagesOK,
			PagesTotal:    scan.PagesTotal,
		}, stats, scan.Err
	}

	updates, err := f.fetchByIDs(ctx, needIDs, dataProfile.QueryPrefix, allowPartial)
	if err != nil && len(updates) == 0 {
		return FetchResult{Entity: f.entity, Err: err}, stats, err
	}

	merged, err := MergeRecordsByID(existing, updates, remoteIDs)
	if err != nil {
		return FetchResult{Entity: f.entity, Err: err}, stats, err
	}

	res := FetchResult{
		Entity:        f.entity,
		Items:         merged,
		CountReported: scan.CountReported,
		PagesOK:       scan.PagesOK,
		PagesTotal:    scan.PagesTotal,
	}
	if err != nil {
		res.Err = fmt.Errorf("incremental: %w", err)
	} else if scan.Err != nil {
		res.Err = scan.Err
	}
	return res, stats, res.Err
}

func countExistingInRemote(existing []json.RawMessage, remoteIDs map[int]struct{}) int {
	n := 0
	for _, raw := range existing {
		id, _, err := ParseIDChecksum(raw)
		if err != nil {
			continue
		}
		if _, ok := remoteIDs[id]; ok {
			n++
		}
	}
	return n
}

func (f *Fetcher) fetchByIDs(ctx context.Context, ids []int, queryPrefix string, allowPartial bool) ([]json.RawMessage, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const batchSize = 500
	var combined []json.RawMessage
	var firstErr error

	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]
		body := fmt.Sprintf("%s where id = (%s); limit %d;", queryPrefix, joinIDs(batch), len(batch))
		out, err := f.client.Post(ctx, string(f.entity), []byte(body))
		if err != nil {
			if !allowPartial {
				return nil, fmt.Errorf("fetch ids batch at %d: %w", start, err)
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		var page []json.RawMessage
		if err := json.Unmarshal(out, &page); err != nil {
			return nil, fmt.Errorf("decode ids batch: %w", err)
		}
		combined = append(combined, page...)
	}
	return combined, firstErr
}

func joinIDs(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}
