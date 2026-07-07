package igdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
