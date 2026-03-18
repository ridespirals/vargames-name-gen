package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// parseOffset extracts "offset N" from body for test handlers.
func parseOffset(body string) int {
	const pref = "offset "
	i := strings.Index(body, pref)
	if i < 0 {
		return -1
	}
	i += len(pref)
	j := i
	for j < len(body) && body[j] >= '0' && body[j] <= '9' {
		j++
	}
	n, _ := strconv.Atoi(body[i:j])
	return n
}

type fakeFetcherClient struct {
	maxLimit int

	mu sync.Mutex
	seen        bool
	firstOffset int

	post func(ctx context.Context, endpoint string, body []byte) ([]byte, error)
}

func (f *fakeFetcherClient) MaxLimit() int { return f.maxLimit }

func (f *fakeFetcherClient) Post(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
	f.mu.Lock()
	offset := parseOffset(string(body))
	if !f.seen {
		f.seen = true
		f.firstOffset = offset
	}
	post := f.post
	f.mu.Unlock()

	if post == nil {
		return []byte(`[]`), nil
	}
	return post(ctx, endpoint, body)
}

func TestFetcher_FetchAll_CombinesPages(t *testing.T) {
	t.Helper()

	type offsetResponse struct {
		items []json.RawMessage
	}

	// Limit=2:
	//  - offset 0  => 2 items
	//  - offset 2  => 2 items
	//  - offset 4  => 1 item (< limit, stop)
	responses := map[int]offsetResponse{
		0: {items: []json.RawMessage{json.RawMessage(`{"id":1}`), json.RawMessage(`{"id":2}`)}},
		2: {items: []json.RawMessage{json.RawMessage(`{"id":3}`), json.RawMessage(`{"id":4}`)}},
		4: {items: []json.RawMessage{json.RawMessage(`{"id":5}`)}},
	}

	var calledOffsets []int

	fake := &fakeFetcherClient{
		maxLimit:  10,
		firstOffset: -1,
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			offset := parseOffset(string(body))
			calledOffsets = append(calledOffsets, offset)

			resp, ok := responses[offset]
			if !ok {
				return []byte(`[]`), nil
			}
			raw, err := json.Marshal(resp.items)
			if err != nil {
				return nil, err
			}
			return raw, nil
		},
	}

	fetcher := NewFetcher(fake, EntityGames, FetcherOptions{
		Limit: 2,
	})
	results, err := fetcher.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 combined results, got %d", len(results))
	}
	wantOffsets := []int{0, 2, 4}
	if len(calledOffsets) != len(wantOffsets) {
		t.Fatalf("expected %d requests, got %d (offsets: %v)", len(wantOffsets), len(calledOffsets), calledOffsets)
	}
	for i := range wantOffsets {
		if calledOffsets[i] != wantOffsets[i] {
			t.Fatalf("expected request offset[%d]=%d, got %d (all offsets: %v)", i, wantOffsets[i], calledOffsets[i], calledOffsets)
		}
	}
}

func TestFetcher_FirstRequestOffsetZero(t *testing.T) {
	fake := &fakeFetcherClient{
		maxLimit:    100,
		firstOffset: -1,
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			return []byte(`[]`), nil
		},
	}

	fetcher := NewFetcher(fake, EntityGames, FetcherOptions{
		Limit: 2,
	})
	if _, err := fetcher.FetchAll(context.Background()); err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if fake.firstOffset != 0 {
		t.Fatalf("expected first request offset 0, got %d", fake.firstOffset)
	}
}

func TestFetcher_FirstRequestOffsetZero_MultiEntities(t *testing.T) {
	fakeGenres := &fakeFetcherClient{
		maxLimit:    100,
		firstOffset: -1,
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			return []byte(`[]`), nil
		},
	}
	fakePlatforms := &fakeFetcherClient{
		maxLimit:    100,
		firstOffset: -1,
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			return []byte(`[]`), nil
		},
	}

	errCh := make(chan error, 2)
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		fetcher := NewFetcher(fakeGenres, EntityGenres, FetcherOptions{Limit: 2})
		_, err := fetcher.FetchAll(context.Background())
		errCh <- err
	}()
	go func() {
		defer wg.Done()
		fetcher := NewFetcher(fakePlatforms, EntityPlatforms, FetcherOptions{Limit: 2})
		_, err := fetcher.FetchAll(context.Background())
		errCh <- err
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("FetchAll: %v", err)
		}
	}

	if fakeGenres.firstOffset != 0 {
		t.Fatalf("expected first request offset 0 for genres, got %d", fakeGenres.firstOffset)
	}
	if fakePlatforms.firstOffset != 0 {
		t.Fatalf("expected first request offset 0 for platforms, got %d", fakePlatforms.firstOffset)
	}
}

func TestFetcher_FetchAll_EmptyResponse(t *testing.T) {
	fake := &fakeFetcherClient{
		maxLimit:    500,
		firstOffset: -1,
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			return []byte(`[]`), nil
		},
	}
	fetcher := NewFetcher(fake, EntityGenres, FetcherOptions{Limit: 2})
	results, err := fetcher.FetchAll(context.Background())
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestFetcher_FetchAll_PropagatesError(t *testing.T) {
	wantErr := errors.New("unauthorized")
	fake := &fakeFetcherClient{
		maxLimit:    500,
		firstOffset: -1,
		post: func(ctx context.Context, endpoint string, body []byte) ([]byte, error) {
			return nil, wantErr
		},
	}
	fetcher := NewFetcher(fake, EntityCharacters, FetcherOptions{Limit: 2})
	_, err := fetcher.FetchAll(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestFetcher_Entity(t *testing.T) {
	fetcher := NewFetcher(&fakeFetcherClient{maxLimit: 100}, EntityPlatforms, FetcherOptions{})
	if fetcher.Entity() != EntityPlatforms {
		t.Errorf("Entity() = %s, want platforms", fetcher.Entity())
	}
}

func TestAllEntities(t *testing.T) {
	all := AllEntities()
	seen := make(map[Entity]bool)
	for _, e := range all {
		if seen[e] {
			t.Errorf("duplicate entity %s", e)
		}
		seen[e] = true
	}
	want := []Entity{
		EntityGames, EntityCharacters, EntityGenres, EntityPlatforms,
		EntityCollections, EntityCompanies, EntityAlternativeNames,
	}
	if len(all) != len(want) {
		t.Errorf("AllEntities() length = %d, want %d", len(all), len(want))
	}
	for _, e := range want {
		if !seen[e] {
			t.Errorf("missing entity %s", e)
		}
	}
}
