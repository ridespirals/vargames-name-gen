package igdb

import (
	"context"
	"encoding/json"
	"fmt"

	"vargames-name-gen/src/config"
)

// FetcherClient is the minimal surface area required by Fetcher.
// It allows unit tests to inject a mock without making HTTP calls.
type FetcherClient interface {
	MaxLimit() int
	Post(ctx context.Context, endpoint string, body []byte) ([]byte, error)
}

// FetcherOptions configures parallel page fetching.
type FetcherOptions struct {
	// Limit is the page size (max items per request). Default from config MaxLimit.
	Limit int
	// MaxConcurrent limits concurrent page requests (0 = use default).
	MaxConcurrent int
	// Logger is optional; when set, progress (pages completed, total items) is logged.
	Logger Logger
}

const (
	defaultMaxConcurrent = 4
)

// Fetcher fetches all pages for an IGDB entity and combines results.
type Fetcher struct {
	client FetcherClient
	entity Entity
	limit  int
	sem    chan struct{}
	log    Logger
}

// NewFetcher creates a fetcher for the given entity using the client's config for limit when not set in opts.
func NewFetcher(client FetcherClient, entity Entity, opts FetcherOptions) *Fetcher {
	limit := opts.Limit
	if limit <= 0 {
		limit = client.MaxLimit()
	}
	if limit <= 0 {
		limit = config.DefaultMaxLimit
	}
	concurrent := opts.MaxConcurrent
	if concurrent <= 0 {
		concurrent = defaultMaxConcurrent
	}
	return &Fetcher{
		client: client,
		entity: entity,
		limit:  limit,
		sem:    make(chan struct{}, concurrent),
		log:    opts.Logger, //.With("entity", entity),
	}
}

// FetchAll fetches items for the entity starting at offset 0 and walking forward in page-sized
// increments until:
//   - a page returns zero results, or
//   - a page returns fewer than Limit results.
//
// This guarantees that the first request always has offset 0, and that we stop once the end
// of the dataset is reached.
func (f *Fetcher) FetchAll(ctx context.Context) ([]json.RawMessage, error) {
	if f.log != nil {
		// f.log.Logf("fetcher %s: starting paging with limit %d", f.entity, f.limit)
		f.log.Logf("starting fetcher", "entity", f.entity, "limit", f.limit)
	}
	var combined []json.RawMessage

	for page := 0; ; page++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		offset := page * f.limit
		body := fmt.Sprintf("fields *; limit %d; offset %d;", f.limit, offset)
		out, err := f.client.Post(ctx, string(f.entity), []byte(body))
		if err != nil {
			return nil, fmt.Errorf("page %d (offset %d): %w", page, offset, err)
		}
		var pageResults []json.RawMessage
		if err := json.Unmarshal(out, &pageResults); err != nil {
			return nil, fmt.Errorf("page %d decode: %w", page, err)
		}
		if len(pageResults) == 0 {
			if f.log != nil {
				// f.log.Logf("fetcher %s: page %d returned 0 items, stopping", f.entity, page+1)
				f.log.Logf("stopping fetcher, returned 0 items", "entity", f.entity, "page", page+1)
			}
			break
		}
		combined = append(combined, pageResults...)
		if f.log != nil {
			// f.log.Logf("fetcher %s: page %d fetched %d items (total %d)", f.entity, page+1, len(pageResults), len(combined))
			f.log.Logf("fetcher", "entity", f.entity, "page", page+1, "fetched", len(pageResults), "total", len(combined))
		}
		if len(pageResults) < f.limit {
			if f.log != nil {
				// f.log.Logf("fetcher %s: page %d returned %d < %d items, assuming end of data", f.entity, page+1, len(pageResults), f.limit)
				f.log.Logf("stopping fetcher, returned less than limit items, assuming end of data", "entity", f.entity, "page", page+1, "fetched", len(pageResults), "limit", f.limit)
			}
			break
		}
	}

	if f.log != nil {
		// f.log.Logf("fetcher %s: finished %d items", f.entity, len(combined))
		f.log.Logf("finished fetcher", "entity", f.entity, "items", len(combined))
	}
	return combined, nil
}

// Entity returns the entity type this fetcher uses.
func (f *Fetcher) Entity() Entity { return f.entity }
