package db

import (
	"testing"
	"time"

	"github.com/example/realtime-logs/internal/models"
)

func TestMemoryStoreQueryOrdersNewestFirst(t *testing.T) {
	store := NewMemoryStore(10)
	entries := []models.LogEntry{
		{OrgID: "acme", Level: "info", Message: "old", Timestamp: "2025-01-01T00:00:00Z"},
		{OrgID: "acme", Level: "info", Message: "new", Timestamp: "2025-01-02T00:00:00Z"},
	}

	if err := store.Append(entries); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	items, total := store.Query(QueryParams{OrgID: "acme", Limit: 10})
	if total != 2 {
		t.Fatalf("expected total 2, got %d", total)
	}
	if got := items[0].Message; got != "new" {
		t.Fatalf("expected newest item first, got %q", got)
	}
}

func TestAppendNormalizesInvalidTimestamp(t *testing.T) {
	store := NewMemoryStore(10)
	entries := []models.LogEntry{
		{OrgID: "acme", Level: "info", Message: "bad time", Timestamp: "nope"},
	}

	if err := store.Append(entries); err != nil {
		t.Fatalf("append failed: %v", err)
	}

	if _, err := time.Parse(time.RFC3339Nano, entries[0].Timestamp); err != nil {
		t.Fatalf("expected normalized timestamp, got %q", entries[0].Timestamp)
	}
	if entries[0].Ts.IsZero() {
		t.Fatal("expected Ts to be set")
	}
}
