package service

import (
	"context"
	"testing"
	"time"

	"github.com/chrisuzor/commerce-ops-console/internal/ai"
	"github.com/chrisuzor/commerce-ops-console/internal/store"
)

func TestIngestIsIdempotent(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	repository := store.NewMemory()
	orders := NewOrders(repository, ai.EvidenceSummarizer{}, func() time.Time { return now })
	input := IngestInput{
		OrderID: "ORD-1", Type: "payment.succeeded", Source: "payments",
		IdempotencyKey: "provider-event-123", OccurredAt: now.Add(-time.Minute),
	}

	first, err := orders.Ingest(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := orders.Ingest(ctx, input)
	if err != nil {
		t.Fatal(err)
	}

	if first.Duplicate || !second.Duplicate {
		t.Fatalf("duplicate flags first=%v second=%v", first.Duplicate, second.Duplicate)
	}
	if first.Event.ID != second.Event.ID {
		t.Fatalf("ids differ: %s != %s", first.Event.ID, second.Event.ID)
	}
	events, _ := repository.EventsForOrder(ctx, "ORD-1")
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
}

func TestRetryFulfillmentIsIdempotentAndAudited(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	repository := store.NewMemory()
	orders := NewOrders(repository, ai.EvidenceSummarizer{}, func() time.Time { return now })
	_, err := orders.Ingest(ctx, IngestInput{
		OrderID: "ORD-2", Type: "payment.succeeded", Source: "payments",
		IdempotencyKey: "payment-2", OccurredAt: now.Add(-20 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	first, err := orders.RetryFulfillment(ctx, "ORD-2", "operator@example.com", "retry-click-1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := orders.RetryFulfillment(ctx, "ORD-2", "operator@example.com", "retry-click-1")
	if err != nil {
		t.Fatal(err)
	}

	if first.Replayed || !second.Replayed || first.Action.ID != second.Action.ID {
		t.Fatalf("unexpected retry results first=%#v second=%#v", first, second)
	}
	events, _ := repository.EventsForOrder(ctx, "ORD-2")
	if len(events) != 2 {
		t.Fatalf("event count = %d, want payment plus one retry event", len(events))
	}
}

func TestRejectsFutureEvent(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	orders := NewOrders(store.NewMemory(), ai.EvidenceSummarizer{}, func() time.Time { return now })
	_, err := orders.Ingest(context.Background(), IngestInput{
		OrderID: "ORD-3", Type: "order.created", Source: "store", IdempotencyKey: "future",
		OccurredAt: now.Add(6 * time.Minute),
	})
	if err == nil {
		t.Fatal("expected future event to be rejected")
	}
}
