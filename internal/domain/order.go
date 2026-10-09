package domain

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

type EventType string

const (
	EventOrderCreated         EventType = "order.created"
	EventPaymentSucceeded     EventType = "payment.succeeded"
	EventPaymentFailed        EventType = "payment.failed"
	EventFulfillmentStarted   EventType = "fulfillment.started"
	EventFulfillmentFailed    EventType = "fulfillment.failed"
	EventFulfillmentSucceeded EventType = "fulfillment.succeeded"
	EventFulfillmentRetry     EventType = "fulfillment.retry_requested"
	EventNotificationSent     EventType = "notification.sent"
	EventNotificationFailed   EventType = "notification.failed"
)

var validEventTypes = map[EventType]struct{}{
	EventOrderCreated: {}, EventPaymentSucceeded: {}, EventPaymentFailed: {},
	EventFulfillmentStarted: {}, EventFulfillmentFailed: {}, EventFulfillmentSucceeded: {},
	EventFulfillmentRetry: {}, EventNotificationSent: {}, EventNotificationFailed: {},
}

func (e EventType) Valid() bool {
	_, ok := validEventTypes[e]
	return ok
}

type Event struct {
	ID             string         `json:"id"`
	OrderID        string         `json:"orderId"`
	Type           EventType      `json:"type"`
	Source         string         `json:"source"`
	IdempotencyKey string         `json:"idempotencyKey"`
	OccurredAt     time.Time      `json:"occurredAt"`
	ReceivedAt     time.Time      `json:"receivedAt"`
	Metadata       map[string]any `json:"metadata"`
}

type Status string

const (
	StatusPending    Status = "PENDING"
	StatusPaid       Status = "PAID"
	StatusProcessing Status = "PROCESSING"
	StatusAttention  Status = "NEEDS_ATTENTION"
	StatusFulfilled  Status = "FULFILLED"
)

type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
)

type Issue struct {
	Code         string   `json:"code"`
	Title        string   `json:"title"`
	Explanation  string   `json:"explanation"`
	Severity     Severity `json:"severity"`
	Evidence     []string `json:"evidence"`
	Acknowledged bool     `json:"acknowledged"`
}

type Order struct {
	ID      string  `json:"id"`
	Status  Status  `json:"status"`
	Events  []Event `json:"events"`
	Issues  []Issue `json:"issues"`
	Summary string  `json:"summary"`
}

func BuildOrder(orderID string, events []Event, now time.Time, acknowledged map[string]bool) Order {
	ordered := append([]Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].OccurredAt.Equal(ordered[j].OccurredAt) {
			return ordered[i].ReceivedAt.Before(ordered[j].ReceivedAt)
		}
		return ordered[i].OccurredAt.Before(ordered[j].OccurredAt)
	})

	order := Order{ID: orderID, Status: StatusPending, Events: ordered}
	var paymentAt, fulfillmentAt time.Time
	fulfillmentFailures := 0
	notificationFailures := 0
	logicalOutOfOrder := false

	for _, event := range ordered {
		switch event.Type {
		case EventPaymentSucceeded:
			paymentAt = event.OccurredAt
			if order.Status != StatusFulfilled {
				order.Status = StatusPaid
			}
		case EventFulfillmentStarted:
			fulfillmentAt = event.OccurredAt
			if paymentAt.IsZero() || event.OccurredAt.Before(paymentAt) {
				logicalOutOfOrder = true
			}
			order.Status = StatusProcessing
		case EventFulfillmentFailed:
			fulfillmentFailures++
			order.Status = StatusAttention
		case EventFulfillmentSucceeded:
			fulfillmentAt = event.OccurredAt
			if paymentAt.IsZero() || event.OccurredAt.Before(paymentAt) {
				logicalOutOfOrder = true
			}
			order.Status = StatusFulfilled
		case EventPaymentFailed:
			order.Status = StatusAttention
		case EventNotificationFailed:
			notificationFailures++
		}
	}

	if !paymentAt.IsZero() && fulfillmentAt.IsZero() && now.Sub(paymentAt) >= 15*time.Minute {
		order.Issues = append(order.Issues, Issue{
			Code: "PAID_WITHOUT_FULFILLMENT", Title: "Fulfillment has not started",
			Explanation: "Payment succeeded, but no fulfillment event followed within 15 minutes.",
			Severity:    SeverityCritical, Evidence: []string{eventEvidence(EventPaymentSucceeded, paymentAt)},
		})
	}
	if fulfillmentFailures >= 3 {
		order.Issues = append(order.Issues, Issue{
			Code: "REPEATED_FULFILLMENT_FAILURE", Title: "Fulfillment repeatedly failed",
			Explanation: "Three or more fulfillment attempts failed and require operator review.",
			Severity:    SeverityCritical, Evidence: []string{countEvidence(EventFulfillmentFailed, fulfillmentFailures)},
		})
	}
	if notificationFailures > 0 {
		order.Issues = append(order.Issues, Issue{
			Code: "NOTIFICATION_FAILURE", Title: "Customer notification failed",
			Explanation: "At least one customer notification could not be delivered.",
			Severity:    SeverityWarning, Evidence: []string{countEvidence(EventNotificationFailed, notificationFailures)},
		})
	}
	if logicalOutOfOrder {
		order.Issues = append(order.Issues, Issue{
			Code: "EVENT_ORDER_CONFLICT", Title: "Event sequence is inconsistent",
			Explanation: "Fulfillment was recorded before a successful payment event.",
			Severity:    SeverityWarning, Evidence: []string{"Compare payment and fulfillment timestamps in the event timeline."},
		})
	}
	for i := range order.Issues {
		order.Issues[i].Acknowledged = acknowledged[order.Issues[i].Code]
	}
	return order
}

func eventEvidence(kind EventType, at time.Time) string {
	return string(kind) + " at " + at.UTC().Format(time.RFC3339)
}

func countEvidence(kind EventType, count int) string {
	return strings.Join([]string{string(kind), "count", strconv.Itoa(count)}, ": ")
}
