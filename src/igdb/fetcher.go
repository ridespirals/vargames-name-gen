package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"vargames-name-gen/src/config"
)

// errCountUnavailable signals that Count is not implemented or failed; FetchAll falls back to sequential paging.
var errCountUnavailable = errors.New("count unavailable")

// FetcherClient is the minimal surface area required by Fetcher.
// It allows unit tests to inject a mock without making HTTP calls.
type FetcherClient interface {
	MaxLimit() int
	Post(ctx context.Context, endpoint string, body []byte) ([]byte, error)
	Count(ctx context.Context, entity Entity) (int, error)
}

// FetcherOptions configures parallel page fetching.
type FetcherOptions struct {
	// Limit is the page size (max items per request). Default from config MaxLimit.
	Limit int
	// MaxConcurrent limits concurrent page requests (0 = use default).
	MaxConcurrent int
	// QueryPrefix is the Apicalypse query fragment inserted before `limit` and `offset`.
	// It should be semicolon-terminated (e.g. `fields *;`), but we'll normalize if not.
	//
	// If empty, defaults to `igdb.DefaultQueryPrefix` for the entity.
	QueryPrefix string
	// Logger is optional; when set, progress (pages completed, total items) is logged.
	Logger Logger
}

const (
	defaultMaxConcurrent = 4
)

// Fetcher fetches all pages for an IGDB entity and combines results.
type Fetcher struct {
	client      FetcherClient
	entity      Entity
	limit       int
	queryPrefix string
	sem         chan struct{}
	log         Logger
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
	queryPrefix := strings.TrimSpace(opts.QueryPrefix)
	if queryPrefix == "" {
		queryPrefix = QueryPrefixForEntity(entity)
	}
	if !strings.HasSuffix(queryPrefix, ";") {
		queryPrefix += ";"
	}
	return &Fetcher{
		client:      client,
		entity:      entity,
		limit:       limit,
		queryPrefix: queryPrefix,
		sem:         make(chan struct{}, concurrent),
		log:         opts.Logger,
	}
}

// FetchAll fetches all items for the entity. When Count is available, pages are fetched concurrently
// using the reported total; otherwise it walks forward sequentially until an empty or short page.
func (f *Fetcher) FetchAll(ctx context.Context) ([]json.RawMessage, error) {
	if f.log != nil {
		f.log.Logf("fetcher %s: starting paging with limit %d", f.entity, f.limit)
	}

	count, err := f.client.Count(ctx, f.entity)
	if err != nil {
		if f.log != nil {
			f.log.Logf("fetcher %s: count unavailable (%v), falling back to sequential paging", f.entity, err)
		}
		return f.fetchAllSequential(ctx)
	}
	if count == 0 {
		if f.log != nil {
			f.log.Logf("fetcher %s: count reported 0, skipping fetch", f.entity)
		}
		return nil, nil
	}
	return f.fetchAllConcurrent(ctx, count)
}

func (f *Fetcher) fetchAllSequential(ctx context.Context) ([]json.RawMessage, error) {
	var combined []json.RawMessage

	for page := 0; ; page++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		offset := page * f.limit
		pageResults, err := f.fetchPage(ctx, page, offset)
		if err != nil {
			return nil, err
		}
		if len(pageResults) == 0 {
			if f.log != nil {
				f.log.Logf("fetcher %s: page %d returned 0 items, stopping", f.entity, page+1)
			}
			break
		}
		combined = append(combined, pageResults...)
		if f.log != nil {
			f.log.Logf("fetcher %s: page %d fetched %d items (total %d)", f.entity, page+1, len(pageResults), len(combined))
		}
		if len(pageResults) < f.limit {
			if f.log != nil {
				f.log.Logf("fetcher %s: page %d returned %d < %d items, assuming end of data", f.entity, page+1, len(pageResults), f.limit)
			}
			break
		}
	}

	if f.log != nil {
		f.log.Logf("fetcher %s: finished %d items", f.entity, len(combined))
	}
	return combined, nil
}

func (f *Fetcher) fetchAllConcurrent(ctx context.Context, count int) ([]json.RawMessage, error) {
	pages := pageCountForTotal(count, f.limit)
	if f.log != nil {
		f.log.Logf("fetcher %s: count %d, fetching %d pages concurrently", f.entity, count, pages)
	}

	type pageResult struct {
		page  int
		items []json.RawMessage
		err   error
	}

	resultsCh := make(chan pageResult, pages)
	var wg sync.WaitGroup

	for page := 0; page < pages; page++ {
		page := page
		offset := page * f.limit
		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case f.sem <- struct{}{}:
				defer func() { <-f.sem }()
			case <-ctx.Done():
				resultsCh <- pageResult{page: page, err: ctx.Err()}
				return
			}

			select {
			case <-ctx.Done():
				resultsCh <- pageResult{page: page, err: ctx.Err()}
				return
			default:
			}

			items, err := f.fetchPage(ctx, page, offset)
			resultsCh <- pageResult{page: page, items: items, err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	pageResults := make([]pageResult, 0, pages)
	for r := range resultsCh {
		if r.err != nil {
			return nil, r.err
		}
		pageResults = append(pageResults, r)
	}

	sort.Slice(pageResults, func(i, j int) bool {
		return pageResults[i].page < pageResults[j].page
	})

	var combined []json.RawMessage
	for _, pr := range pageResults {
		combined = append(combined, pr.items...)
		if f.log != nil {
			f.log.Logf("fetcher %s: page %d fetched %d items (total %d)", f.entity, pr.page+1, len(pr.items), len(combined))
		}
	}

	if len(combined) != count {
		if f.log != nil {
			f.log.Logf("fetcher %s: warning: fetched %d items but count reported %d", f.entity, len(combined), count)
		}
	}

	if f.log != nil {
		f.log.Logf("fetcher %s: finished %d items", f.entity, len(combined))
	}
	return combined, nil
}

func (f *Fetcher) fetchPage(ctx context.Context, page, offset int) ([]json.RawMessage, error) {
	body := fmt.Sprintf("%s limit %d; offset %d;", f.queryPrefix, f.limit, offset)
	out, err := f.client.Post(ctx, string(f.entity), []byte(body))
	if err != nil {
		return nil, fmt.Errorf("page %d (offset %d): %w", page, offset, err)
	}
	var pageResults []json.RawMessage
	if err := json.Unmarshal(out, &pageResults); err != nil {
		return nil, fmt.Errorf("page %d decode: %w", page, err)
	}
	return pageResults, nil
}

func pageCountForTotal(count, limit int) int {
	if count <= 0 || limit <= 0 {
		return 0
	}
	return (count + limit - 1) / limit
}

// Entity returns the entity type this fetcher uses.
func (f *Fetcher) Entity() Entity { return f.entity }
