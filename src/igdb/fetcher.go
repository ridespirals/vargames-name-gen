package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// FetcherOptions configures parallel page fetching.
type FetcherOptions struct {
	// Limit is the page size (max items per request). Default from config MaxLimit.
	Limit int
	// MaxPages is the maximum number of pages to fetch (0 = use default).
	// Total items cap is Limit * MaxPages.
	MaxPages int
	// MaxConcurrent limits concurrent page requests (0 = use default).
	MaxConcurrent int
	// Logger is optional; when set, progress (pages completed, total items) is logged.
	Logger Logger
}

const (
	defaultMaxPages     = 20
	defaultMaxConcurrent = 4
)

// Fetcher fetches all pages for an IGDB entity in parallel and combines results.
type Fetcher struct {
	client   *Client
	entity   Entity
	limit    int
	maxPages int
	sem      chan struct{}
	log      Logger
}

// NewFetcher creates a fetcher for the given entity using the client's config for limit when not set in opts.
func NewFetcher(client *Client, entity Entity, opts FetcherOptions) *Fetcher {
	limit := opts.Limit
	if limit <= 0 {
		limit = client.MaxLimit()
	}
	if limit <= 0 {
		limit = 500
	}
	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = defaultMaxPages
	}
	concurrent := opts.MaxConcurrent
	if concurrent <= 0 {
		concurrent = defaultMaxConcurrent
	}
	return &Fetcher{
		client:   client,
		entity:   entity,
		limit:    limit,
		maxPages: maxPages,
		sem:      make(chan struct{}, concurrent),
		log:      opts.Logger,
	}
}

// FetchAll fetches up to Limit*MaxPages items for the entity by requesting one page per goroutine,
// then concatenates all results. Pages that return fewer than Limit items do not trigger further pages.
func (f *Fetcher) FetchAll(ctx context.Context) ([]json.RawMessage, error) {
	if f.log != nil {
		f.log.Logf("fetcher %s: starting %d pages (limit %d)", f.entity, f.maxPages, f.limit)
	}
	var mu sync.Mutex
	var combined []json.RawMessage
	var firstErr error
	var pagesDone int
	var wg sync.WaitGroup
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	for page := 0; page < f.maxPages; page++ {
		page := page
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case f.sem <- struct{}{}:
				defer func() { <-f.sem }()
			case <-ctx.Done():
				mu.Lock()
				if firstErr == nil {
					firstErr = ctx.Err()
				}
				mu.Unlock()
				return
			}
			offset := page * f.limit
			body := fmt.Sprintf("fields *; limit %d; offset %d;", f.limit, offset)
			out, err := f.client.Post(ctx, string(f.entity), []byte(body))
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("page %d (offset %d): %w", page, offset, err)
				}
				mu.Unlock()
				return
			}
			var pageResults []json.RawMessage
			if err := json.Unmarshal(out, &pageResults); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("page %d decode: %w", page, err)
				}
				mu.Unlock()
				return
			}
			if len(pageResults) > 0 {
				mu.Lock()
				combined = append(combined, pageResults...)
				pagesDone++
				total := len(combined)
				mu.Unlock()
				if f.log != nil {
					f.log.Logf("fetcher %s: page %d/%d done (%d items this page, %d total)", f.entity, page+1, f.maxPages, len(pageResults), total)
				}
			}
		}()
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
	}
	mu.Lock()
	err := firstErr
	mu.Unlock()
	if err != nil {
		return nil, err
	}
	if f.log != nil {
		f.log.Logf("fetcher %s: finished %d items", f.entity, len(combined))
	}
	return combined, nil
}

// Entity returns the entity type this fetcher uses.
func (f *Fetcher) Entity() Entity { return f.entity }
