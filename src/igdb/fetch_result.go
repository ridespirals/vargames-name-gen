package igdb

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// FetchResult holds the outcome of a fetch run for one entity, including partial progress.
type FetchResult struct {
	Entity        Entity
	Items         []json.RawMessage
	Err           error
	PagesOK       int
	PagesFailed   int
	PagesTotal    int
	CountReported int
}

// FetchAll fetches all items for the entity. Any page failure aborts and returns an error.
func (f *Fetcher) FetchAll(ctx context.Context) ([]json.RawMessage, error) {
	res := f.FetchAllResult(ctx, false)
	if res.Err != nil {
		return nil, res.Err
	}
	return res.Items, nil
}

// FetchAllResult fetches entity pages. When allowPartial is true, successful pages are kept
// and Err is set if any page failed.
func (f *Fetcher) FetchAllResult(ctx context.Context, allowPartial bool) FetchResult {
	if f.log != nil {
		f.log.Logf("fetcher %s: starting paging with limit %d", f.entity, f.limit)
	}

	count, err := f.client.Count(ctx, f.entity)
	if err != nil {
		if f.log != nil {
			f.log.Logf("fetcher %s: count unavailable (%v), falling back to sequential paging", f.entity, err)
		}
		res := f.fetchAllSequentialResult(ctx, allowPartial, 0)
		res.Entity = f.entity
		return res
	}
	if count == 0 {
		if f.log != nil {
			f.log.Logf("fetcher %s: count reported 0, skipping fetch", f.entity)
		}
		return FetchResult{Entity: f.entity, CountReported: 0}
	}
	res := f.fetchAllConcurrentResult(ctx, count, allowPartial)
	res.Entity = f.entity
	res.CountReported = count
	return res
}

func (f *Fetcher) fetchAllSequentialResult(ctx context.Context, allowPartial bool, countReported int) FetchResult {
	var combined []json.RawMessage
	pagesOK := 0

	for page := 0; ; page++ {
		select {
		case <-ctx.Done():
			return FetchResult{
				Entity:        f.entity,
				Items:         combined,
				Err:           ctx.Err(),
				PagesOK:       pagesOK,
				PagesFailed:   1,
				CountReported: countReported,
			}
		default:
		}

		offset := page * f.limit
		pageResults, err := f.fetchPage(ctx, page, offset)
		if err != nil {
			res := FetchResult{
				Entity:        f.entity,
				Items:         combined,
				Err:           err,
				PagesOK:       pagesOK,
				PagesFailed:   1,
				PagesTotal:    page + 1,
				CountReported: countReported,
			}
			if allowPartial && len(combined) > 0 {
				return res
			}
			return FetchResult{
				Entity:        f.entity,
				Err:           err,
				PagesFailed:   1,
				PagesTotal:    page + 1,
				CountReported: countReported,
			}
		}
		if len(pageResults) == 0 {
			if f.log != nil {
				f.log.Logf("fetcher %s: page %d returned 0 items, stopping", f.entity, page+1)
			}
			break
		}
		pagesOK++
		combined = append(combined, pageResults...)
		if f.log != nil {
			f.log.Logf("fetcher %s: page %d fetched %d items (total %d%s)",
				f.entity, page+1, len(pageResults), len(combined), formatPercent(len(combined), countReported))
		}
		if len(pageResults) < f.limit {
			if f.log != nil {
				f.log.Logf("fetcher %s: page %d returned %d < %d items, assuming end of data", f.entity, page+1, len(pageResults), f.limit)
			}
			break
		}
	}

	if f.log != nil {
		f.log.Logf("fetcher %s: finished %d items%s", f.entity, len(combined), formatPercent(len(combined), countReported))
	}
	return FetchResult{
		Entity:        f.entity,
		Items:         combined,
		PagesOK:       pagesOK,
		PagesTotal:    pagesOK,
		CountReported: countReported,
	}
}

func (f *Fetcher) fetchAllConcurrentResult(ctx context.Context, count int, allowPartial bool) FetchResult {
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

	for page := range pages {
		offset := page * f.limit
		wg.Go(func() {
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
		})
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	pageResults := make([]pageResult, 0, pages)
	var firstErr error
	pagesOK := 0
	pagesFailed := 0
	itemsFetched := 0
	for r := range resultsCh {
		if r.err != nil {
			pagesFailed++
			if firstErr == nil {
				firstErr = r.err
			}
			if f.log != nil {
				pagesDone := pagesOK + pagesFailed
				f.log.Logf("fetcher %s: page %d failed (%d/%d pages%s, %d/%d items%s)",
					f.entity, r.page+1, pagesDone, pages, formatPercent(pagesDone, pages), itemsFetched, count, formatPercent(itemsFetched, count))
			}
			if !allowPartial {
				return FetchResult{
					Entity:      f.entity,
					Err:         r.err,
					PagesFailed: pagesFailed,
					PagesTotal:  pages,
				}
			}
			continue
		}
		pagesOK++
		itemsFetched += len(r.items)
		pageResults = append(pageResults, r)
		if f.log != nil {
			pagesDone := pagesOK + pagesFailed
			f.log.Logf("fetcher %s: %d/%d pages complete (%d/%d items%s)",
				f.entity, pagesDone, pages, itemsFetched, count, formatPercent(itemsFetched, count))
		}
	}

	sort.Slice(pageResults, func(i, j int) bool {
		return pageResults[i].page < pageResults[j].page
	})

	var combined []json.RawMessage
	for _, pr := range pageResults {
		combined = append(combined, pr.items...)
	}

	if len(combined) != count && firstErr == nil {
		if f.log != nil {
			f.log.Logf("fetcher %s: warning: fetched %d items but count reported %d", f.entity, len(combined), count)
		}
	}

	if f.log != nil {
		f.log.Logf("fetcher %s: finished %d items%s", f.entity, len(combined), formatPercent(len(combined), count))
	}

	res := FetchResult{
		Entity:      f.entity,
		Items:       combined,
		PagesOK:     pagesOK,
		PagesFailed: pagesFailed,
		PagesTotal:  pages,
	}
	if firstErr != nil {
		res.Err = fmt.Errorf("partial fetch: %d/%d pages failed: %w", pagesFailed, pages, firstErr)
	}
	return res
}
