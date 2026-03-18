package igdb

import (
	"testing"
	"time"
)

func TestMetrics_RecordPost_NilReceiverDoesNotPanic(t *testing.T) {
	var m *Metrics
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RecordPost panicked: %v", r)
		}
	}()
	m.RecordPost("games", 1, 100*time.Millisecond)
}

func TestMetrics_Snapshot_NilReceiver(t *testing.T) {
	var m *Metrics
	entity, posts := m.Snapshot()
	if entity != "" {
		t.Fatalf("expected empty entity, got %q", entity)
	}
	if posts != nil {
		t.Fatalf("expected nil posts, got %v", posts)
	}
}

func TestMetrics_RecordPost_SnapshotCopy(t *testing.T) {
	m := NewMetrics("games")
	m.RecordPost("games", 2, 150*time.Millisecond)

	entity, posts := m.Snapshot()
	if entity != "games" {
		t.Fatalf("expected entity games, got %q", entity)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Endpoint != "games" || posts[0].Retries != 2 || posts[0].Duration != 150*time.Millisecond {
		t.Fatalf("unexpected post record: %+v", posts[0])
	}

	// Mutate the snapshot output; Metrics internals should not change.
	posts[0].Retries = 999

	_, posts2 := m.Snapshot()
	if posts2[0].Retries != 2 {
		t.Fatalf("expected Metrics internal retries=2, got %d", posts2[0].Retries)
	}
}

func TestMetrics_Totals(t *testing.T) {
	m := NewMetrics("platforms")
	m.RecordPost("platforms", 0, 100*time.Millisecond)
	m.RecordPost("platforms", 2, 250*time.Millisecond)
	m.RecordPost("platforms", 1, 50*time.Millisecond)

	requests, totalRetries, totalDuration := m.Totals()
	if requests != 3 {
		t.Fatalf("expected requests=3, got %d", requests)
	}
	if totalRetries != 3 {
		t.Fatalf("expected totalRetries=3, got %d", totalRetries)
	}
	if totalDuration != (100*time.Millisecond + 250*time.Millisecond + 50*time.Millisecond) {
		t.Fatalf("expected totalDuration=%v, got %v", 100*time.Millisecond+250*time.Millisecond+50*time.Millisecond, totalDuration)
	}
}

