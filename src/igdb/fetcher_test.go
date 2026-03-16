package igdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"vargames-name-gen/src/config"
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

func TestFetcher_FetchAll_CombinesPages(t *testing.T) {
	var mu sync.Mutex
	pageCalls := make(map[int]int)
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/games" {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body := make([]byte, 512)
		n, _ := r.Body.Read(body)
		bodyStr := string(body[:n])
		offset := parseOffset(bodyStr)
		mu.Lock()
		pageCalls[offset]++
		mu.Unlock()
		// Limit=2: page 0 offset 0 → 2 items, page 1 offset 2 → 2 items, page 2 offset 4 → 1 item
		switch {
		case offset == 0:
			w.Write([]byte(`[{"id":1},{"id":2}]`))
		case offset == 2:
			w.Write([]byte(`[{"id":3},{"id":4}]`))
		case offset == 4:
			w.Write([]byte(`[{"id":5}]`))
		default:
			w.Write([]byte(`[]`))
		}
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:     "cid",
		BaseURL:      igdb.URL,
		AccessToken:  "test-token",
		MaxLimit:     2,
	}
	client := NewClient(cfg)
	fetcher := NewFetcher(client, EntityGames, FetcherOptions{
		Limit:         2,
		MaxPages:      4,
		MaxConcurrent: 4,
	})
	ctx := context.Background()
	results, err := fetcher.FetchAll(ctx)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 combined results, got %d", len(results))
	}
	mu.Lock()
	if len(pageCalls) < 3 {
		t.Errorf("expected at least 3 pages called, got %v", pageCalls)
	}
	mu.Unlock()
}

func TestFetcher_FetchAll_EmptyResponse(t *testing.T) {
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:    "cid",
		BaseURL:     igdb.URL,
		AccessToken: "test-token",
		MaxLimit:    500,
	}
	client := NewClient(cfg)
	fetcher := NewFetcher(client, EntityGenres, FetcherOptions{MaxPages: 2})
	ctx := context.Background()
	results, err := fetcher.FetchAll(ctx)
	if err != nil {
		t.Fatalf("FetchAll: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFetcher_FetchAll_PropagatesError(t *testing.T) {
	igdb := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
	}))
	defer igdb.Close()

	cfg := config.Config{
		ClientID:    "cid",
		BaseURL:     igdb.URL,
		AccessToken: "test-token",
		MaxLimit:    500,
	}
	client := NewClient(cfg, WithRetries(0))
	fetcher := NewFetcher(client, EntityCharacters, FetcherOptions{MaxPages: 2})
	ctx := context.Background()
	_, err := fetcher.FetchAll(ctx)
	if err == nil {
		t.Fatal("expected error from 401")
	}
}

func TestFetcher_Entity(t *testing.T) {
	client := NewClient(config.Config{MaxLimit: 100})
	fetcher := NewFetcher(client, EntityPlatforms, FetcherOptions{})
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
