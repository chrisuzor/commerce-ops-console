package domain

import (
	"testing"
	"time"
)

func TestBuildOrderDetectsPaidWithoutFulfillment(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	events := []Event{{
		ID: "event-1", OrderID: "ORD-1", Type: EventPaymentSucceeded,
		OccurredAt: now.Add(-20 * time.Minute), ReceivedAt: now.Add(-19 * time.Minute),
	}}

	order := BuildOrder("ORD-1", events, now, nil)

	if order.Status != StatusPaid {
		t.Fatalf("status = %s, want %s", order.Status, StatusPaid)
	}
	if len(order.Issues) != 1 || order.Issues[0].Code != "PAID_WITHOUT_FULFILLMENT" {
		t.Fatalf("issues = %#v, want paid-without-fulfillment", order.Issues)
	}
}

func TestBuildOrderSortsEventsAndDetectsLogicalConflict(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "payment", OrderID: "ORD-2", Type: EventPaymentSucceeded, OccurredAt: now.Add(-5 * time.Minute), ReceivedAt: now.Add(-9 * time.Minute)},
		{ID: "fulfillment", OrderID: "ORD-2", Type: EventFulfillmentStarted, OccurredAt: now.Add(-10 * time.Minute), ReceivedAt: now.Add(-2 * time.Minute)},
	}

	order := BuildOrder("ORD-2", events, now, nil)

	if order.Events[0].ID != "fulfillment" {
		t.Fatalf("first event = %s, want fulfillment", order.Events[0].ID)
	}
	if !hasIssue(order, "EVENT_ORDER_CONFLICT") {
		t.Fatalf("issues = %#v, want event-order conflict", order.Issues)
	}
}

func TestBuildOrderDetectsRepeatedFailureAndAcknowledgement(t *testing.T) {
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "1", OrderID: "ORD-3", Type: EventFulfillmentFailed, OccurredAt: now.Add(-3 * time.Minute)},
		{ID: "2", OrderID: "ORD-3", Type: EventFulfillmentFailed, OccurredAt: now.Add(-2 * time.Minute)},
		{ID: "3", OrderID: "ORD-3", Type: EventFulfillmentFailed, OccurredAt: now.Add(-time.Minute)},
	}

	order := BuildOrder("ORD-3", events, now, map[string]bool{"REPEATED_FULFILLMENT_FAILURE": true})

	if len(order.Issues) != 1 || !order.Issues[0].Acknowledged {
		t.Fatalf("issues = %#v, want acknowledged repeated-failure issue", order.Issues)
	}
}

func hasIssue(order Order, code string) bool {
	for _, issue := range order.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
