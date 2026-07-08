package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestFetchAllResult_PartialConcurrentKeepsSuccessfulPages(t *testing.T) {
	responses := map[int][]json.RawMessage{
		0: {json.RawMessage(`{"id":1}`), json.RawMessage(`{"id":2}`)},
		2: {json.RawMessage(`{"id":3}`), json.RawMessage(`{"id":4}`)},
	}
	pageErr := errors.New("page failed")

	fake := &fakeFetcherClient{
		maxLimit: 10,
		count:    new(4),
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			offset := parseOffset(string(body))
			if offset == 2 {
				return nil, pageErr
			}
			items, ok := responses[offset]
			if !ok {
				return []byte(`[]`), nil
			}
			return json.Marshal(items)
		},
	}

	fetcher := NewFetcher(fake, EntityGames, FetcherOptions{Limit: 2, MaxConcurrent: 2})
	res := fetcher.FetchAllResult(context.Background(), true)
	if res.Err == nil {
		t.Fatal("expected partial error")
	}
	if len(res.Items) != 2 {
		t.Fatalf("got %d items, want 2 from successful page 0", len(res.Items))
	}
	if res.PagesOK != 1 || res.PagesFailed != 1 {
		t.Fatalf("pagesOK=%d pagesFailed=%d, want 1 and 1", res.PagesOK, res.PagesFailed)
	}
}

func TestFetchAllResult_PartialSequentialKeepsProgress(t *testing.T) {
	responses := map[int][]json.RawMessage{
		0: {json.RawMessage(`{"id":1}`), json.RawMessage(`{"id":2}`)},
	}
	wantErr := errors.New("boom")

	fake := &fakeFetcherClient{
		maxLimit: 10,
		countErr: errCountUnavailable,
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			offset := parseOffset(string(body))
			if offset == 2 {
				return nil, wantErr
			}
			items, ok := responses[offset]
			if !ok {
				return []byte(`[]`), nil
			}
			return json.Marshal(items)
		},
	}

	fetcher := NewFetcher(fake, EntityGames, FetcherOptions{Limit: 2})
	res := fetcher.FetchAllResult(context.Background(), true)
	if !errors.Is(res.Err, wantErr) && res.Err == nil {
		t.Fatalf("expected error, got %v", res.Err)
	}
	if len(res.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(res.Items))
	}
}

func TestFetchAllResult_NonPartialFailsFast(t *testing.T) {
	wantErr := errors.New("fail")

	fake := &fakeFetcherClient{
		maxLimit: 10,
		count:    new(4),
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			offset := parseOffset(string(body))
			if offset == 2 {
				return nil, wantErr
			}
			return json.Marshal([]json.RawMessage{json.RawMessage(`{"id":1}`), json.RawMessage(`{"id":2}`)})
		},
	}

	fetcher := NewFetcher(fake, EntityGames, FetcherOptions{Limit: 2, MaxConcurrent: 2})
	res := fetcher.FetchAllResult(context.Background(), false)
	if res.Err == nil {
		t.Fatal("expected error")
	}
	if len(res.Items) != 0 {
		t.Fatalf("expected no items on fail-fast, got %d", len(res.Items))
	}
}
