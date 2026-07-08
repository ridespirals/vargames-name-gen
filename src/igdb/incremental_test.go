package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestIDsNeedingFullFetch(t *testing.T) {
	existing := []json.RawMessage{
		json.RawMessage(`{"id":1,"checksum":"aaa","name":"A"}`),
		json.RawMessage(`{"id":2,"checksum":"bbb","name":"B"}`),
	}
	remote := map[int]string{1: "aaa", 2: "ccc", 3: "ddd"}

	need, unchanged := IDsNeedingFullFetch(existing, remote)
	if unchanged != 1 {
		t.Fatalf("unchanged = %d, want 1", unchanged)
	}
	if len(need) != 2 {
		t.Fatalf("need %v, want ids 2 and 3", need)
	}
}

func TestMergeRecordsByID(t *testing.T) {
	existing := []json.RawMessage{
		json.RawMessage(`{"id":1,"checksum":"a","name":"Old"}`),
		json.RawMessage(`{"id":2,"checksum":"b","name":"Remove"}`),
	}
	updates := []json.RawMessage{
		json.RawMessage(`{"id":1,"checksum":"a2","name":"New"}`),
		json.RawMessage(`{"id":3,"checksum":"c","name":"Added"}`),
	}
	remoteIDs := map[int]struct{}{1: {}, 3: {}}

	merged, err := MergeRecordsByID(existing, updates, remoteIDs)
	if err != nil {
		t.Fatalf("MergeRecordsByID: %v", err)
	}
	if len(merged) != 2 {
		t.Fatalf("got %d records, want 2", len(merged))
	}
	var first idChecksumRecord
	if err := json.Unmarshal(merged[0], &first); err != nil {
		t.Fatal(err)
	}
	if first.ID != 1 {
		t.Fatalf("first id = %d, want 1", first.ID)
	}
}

func TestFetchIncrementalResult_emptyExistingFullFetch(t *testing.T) {
	fake := &fakeFetcherClient{
		maxLimit: 10,
		count:    intPtr(2),
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			return json.Marshal([]json.RawMessage{
				json.RawMessage(`{"id":1,"name":"A","checksum":"x"}`),
				json.RawMessage(`{"id":2,"name":"B","checksum":"y"}`),
			})
		},
	}
	profile, _ := ProfileFor(EntityGenres, ProfileMinimal)
	fetcher := NewFetcher(fake, EntityGenres, FetcherOptions{Limit: 10})
	res, stats, err := fetcher.FetchIncrementalResult(context.Background(), nil, profile, false)
	if err != nil {
		t.Fatalf("FetchIncrementalResult: %v", err)
	}
	if !stats.FullRefetch {
		t.Fatal("expected full refetch")
	}
	if len(res.Items) != 2 {
		t.Fatalf("got %d items", len(res.Items))
	}
}

func TestFetchIncrementalResult_checksumDiffPullsUpdates(t *testing.T) {
	existing := []json.RawMessage{
		json.RawMessage(`{"id":1,"name":"Old","checksum":"aaa"}`),
	}
	var postBodies []string

	fake := &fakeFetcherClient{
		maxLimit: 10,
		count:    intPtr(1),
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			postBodies = append(postBodies, string(body))
			s := string(body)
			if strings.Contains(s, "where id =") {
				return json.Marshal([]json.RawMessage{
					json.RawMessage(`{"id":1,"name":"Updated","checksum":"bbb"}`),
				})
			}
			return json.Marshal([]json.RawMessage{
				json.RawMessage(`{"id":1,"checksum":"bbb"}`),
			})
		},
	}

	profile, _ := ProfileFor(EntityGenres, ProfileMinimal)
	fetcher := NewFetcher(fake, EntityGenres, FetcherOptions{Limit: 10})
	res, stats, err := fetcher.FetchIncrementalResult(context.Background(), existing, profile, false)
	if err != nil {
		t.Fatalf("FetchIncrementalResult: %v", err)
	}
	if stats.Updated != 1 || stats.Unchanged != 0 {
		t.Fatalf("stats: %+v", stats)
	}
	if len(res.Items) != 1 {
		t.Fatalf("got %d items", len(res.Items))
	}
	var rec struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(res.Items[0], &rec); err != nil {
		t.Fatal(err)
	}
	if rec.Name != "Updated" {
		t.Fatalf("name = %q", rec.Name)
	}
}

func TestFetchIncrementalResult_noChanges(t *testing.T) {
	existing := []json.RawMessage{
		json.RawMessage(`{"id":1,"name":"Same","checksum":"aaa"}`),
	}
	fake := &fakeFetcherClient{
		maxLimit: 10,
		count:    intPtr(1),
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			return json.Marshal([]json.RawMessage{
				json.RawMessage(`{"id":1,"checksum":"aaa"}`),
			})
		},
	}
	profile, _ := ProfileFor(EntityGenres, ProfileMinimal)
	fetcher := NewFetcher(fake, EntityGenres, FetcherOptions{Limit: 10})
	res, stats, err := fetcher.FetchIncrementalResult(context.Background(), existing, profile, false)
	if err != nil {
		t.Fatalf("FetchIncrementalResult: %v", err)
	}
	if stats.Updated != 0 || stats.New != 0 {
		t.Fatalf("stats: %+v", stats)
	}
	if len(res.Items) != 1 {
		t.Fatalf("got %d items", len(res.Items))
	}
}

func TestFetchIncrementalResult_fetchByIDsError(t *testing.T) {
	wantErr := errors.New("api down")
	fake := &fakeFetcherClient{
		maxLimit: 10,
		count:    intPtr(1),
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			if strings.Contains(string(body), "where id =") {
				return nil, wantErr
			}
			return json.Marshal([]json.RawMessage{
				json.RawMessage(`{"id":1,"checksum":"new"}`),
			})
		},
	}
	existing := []json.RawMessage{json.RawMessage(`{"id":1,"checksum":"old"}`)}
	profile, _ := ProfileFor(EntityGenres, ProfileMinimal)
	fetcher := NewFetcher(fake, EntityGenres, FetcherOptions{Limit: 10})
	_, _, err := fetcher.FetchIncrementalResult(context.Background(), existing, profile, false)
	if err == nil {
		t.Fatal("expected error")
	}
}
