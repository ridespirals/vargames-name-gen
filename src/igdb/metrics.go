package igdb

import (
	"sync"
	"time"
)

// PostRecord holds metrics for a single POST request (one page or one call).
type PostRecord struct {
	Endpoint string        `json:"endpoint"`
	Retries  int           `json:"retries"`
	Duration time.Duration `json:"duration_ms"`
}

// Metrics collects per-request and run metrics for reporting.
type Metrics struct {
	mu     sync.Mutex
	posts  []PostRecord
	entity string
}

// NewMetrics returns metrics for a fetch run (entity is the entity being fetched).
func NewMetrics(entity string) *Metrics {
	return &Metrics{entity: entity}
}

// RecordPost records one POST call (endpoint, retries used, duration).
func (m *Metrics) RecordPost(endpoint string, retries int, duration time.Duration) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.posts = append(m.posts, PostRecord{
		Endpoint: endpoint,
		Retries:  retries,
		Duration: duration,
	})
}

// Snapshot returns a copy of all post records and the entity name.
func (m *Metrics) Snapshot() (entity string, posts []PostRecord) {
	if m == nil {
		return "", nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	entity = m.entity
	posts = make([]PostRecord, len(m.posts))
	copy(posts, m.posts)
	return entity, posts
}

// Totals returns total request count, total retries, and sum of durations (not wall clock).
func (m *Metrics) Totals() (requests int, totalRetries int, totalDuration time.Duration) {
	_, posts := m.Snapshot()
	for _, p := range posts {
		requests++
		totalRetries += p.Retries
		totalDuration += p.Duration
	}
	return requests, totalRetries, totalDuration
}
