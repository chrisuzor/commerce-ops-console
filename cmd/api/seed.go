package main

import (
	"context"
	"time"

	"github.com/chrisuzor/commerce-ops-console/internal/service"
)

func seedDemo(ctx context.Context, orders *service.Orders, now time.Time) error {
	events := []service.IngestInput{
		{OrderID: "ORD-1042", Type: "order.created", Source: "store", IdempotencyKey: "demo-1042-created", OccurredAt: now.Add(-45 * time.Minute)},
		{OrderID: "ORD-1042", Type: "payment.succeeded", Source: "payments", IdempotencyKey: "demo-1042-paid", OccurredAt: now.Add(-42 * time.Minute), Metadata: map[string]any{"amount": 84.50, "currency": "USD"}},
		{OrderID: "ORD-1042", Type: "notification.failed", Source: "notify", IdempotencyKey: "demo-1042-email", OccurredAt: now.Add(-40 * time.Minute), Metadata: map[string]any{"channel": "email", "reason": "mailbox_unavailable"}},
		{OrderID: "ORD-2077", Type: "order.created", Source: "store", IdempotencyKey: "demo-2077-created", OccurredAt: now.Add(-35 * time.Minute)},
		{OrderID: "ORD-2077", Type: "payment.succeeded", Source: "payments", IdempotencyKey: "demo-2077-paid", OccurredAt: now.Add(-33 * time.Minute)},
		{OrderID: "ORD-2077", Type: "fulfillment.started", Source: "factory", IdempotencyKey: "demo-2077-started", OccurredAt: now.Add(-30 * time.Minute)},
		{OrderID: "ORD-2077", Type: "fulfillment.failed", Source: "factory", IdempotencyKey: "demo-2077-failed-1", OccurredAt: now.Add(-27 * time.Minute), Metadata: map[string]any{"reason": "address_rejected"}},
		{OrderID: "ORD-2077", Type: "fulfillment.failed", Source: "factory", IdempotencyKey: "demo-2077-failed-2", OccurredAt: now.Add(-24 * time.Minute), Metadata: map[string]any{"reason": "address_rejected"}},
		{OrderID: "ORD-2077", Type: "fulfillment.failed", Source: "factory", IdempotencyKey: "demo-2077-failed-3", OccurredAt: now.Add(-21 * time.Minute), Metadata: map[string]any{"reason": "address_rejected"}},
		{OrderID: "ORD-3091", Type: "order.created", Source: "store", IdempotencyKey: "demo-3091-created", OccurredAt: now.Add(-20 * time.Minute)},
		{OrderID: "ORD-3091", Type: "payment.succeeded", Source: "payments", IdempotencyKey: "demo-3091-paid", OccurredAt: now.Add(-18 * time.Minute)},
		{OrderID: "ORD-3091", Type: "fulfillment.started", Source: "factory", IdempotencyKey: "demo-3091-started", OccurredAt: now.Add(-15 * time.Minute)},
		{OrderID: "ORD-3091", Type: "fulfillment.succeeded", Source: "factory", IdempotencyKey: "demo-3091-shipped", OccurredAt: now.Add(-8 * time.Minute)},
		{OrderID: "ORD-3091", Type: "notification.sent", Source: "notify", IdempotencyKey: "demo-3091-notified", OccurredAt: now.Add(-6 * time.Minute)},
	}
	for _, event := range events {
		if _, err := orders.Ingest(ctx, event); err != nil {
			return err
		}
	}
	return nil
}
