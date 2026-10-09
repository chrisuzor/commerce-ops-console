package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chrisuzor/commerce-ops-console/internal/ai"
	"github.com/chrisuzor/commerce-ops-console/internal/domain"
	"github.com/chrisuzor/commerce-ops-console/internal/store"
	"github.com/google/uuid"
)

var (
	ErrInvalidEvent  = errors.New("invalid event")
	ErrOrderNotFound = errors.New("order not found")
)

type Clock func() time.Time

type Orders struct {
	store      store.Store
	summarizer ai.Summarizer
	now        Clock
}

func NewOrders(repository store.Store, summarizer ai.Summarizer, now Clock) *Orders {
	if now == nil {
		now = time.Now
	}
	return &Orders{store: repository, summarizer: summarizer, now: now}
}

type IngestInput struct {
	OrderID        string         `json:"orderId"`
	Type           string         `json:"type"`
	Source         string         `json:"source"`
	IdempotencyKey string         `json:"idempotencyKey"`
	OccurredAt     time.Time      `json:"occurredAt"`
	Metadata       map[string]any `json:"metadata"`
}

type IngestResult struct {
	Event     domain.Event `json:"event"`
	Duplicate bool         `json:"duplicate"`
}

func (s *Orders) Ingest(ctx context.Context, input IngestInput) (IngestResult, error) {
	input.OrderID = strings.TrimSpace(input.OrderID)
	input.Source = strings.TrimSpace(input.Source)
	input.IdempotencyKey = strings.TrimSpace(input.IdempotencyKey)
	eventType := domain.EventType(strings.TrimSpace(input.Type))
	if input.OrderID == "" || input.Source == "" || input.IdempotencyKey == "" || !eventType.Valid() {
		return IngestResult{}, fmt.Errorf("%w: orderId, source, idempotencyKey and a supported type are required", ErrInvalidEvent)
	}
	if input.OccurredAt.IsZero() {
		return IngestResult{}, fmt.Errorf("%w: occurredAt is required", ErrInvalidEvent)
	}
	if input.OccurredAt.After(s.now().Add(5 * time.Minute)) {
		return IngestResult{}, fmt.Errorf("%w: occurredAt cannot be more than five minutes in the future", ErrInvalidEvent)
	}

	event := domain.Event{
		ID: uuid.NewString(), OrderID: input.OrderID, Type: eventType, Source: input.Source,
		IdempotencyKey: input.IdempotencyKey, OccurredAt: input.OccurredAt.UTC(),
		ReceivedAt: s.now().UTC(), Metadata: cloneMap(input.Metadata),
	}
	stored, duplicate, err := s.store.SaveEvent(ctx, event)
	return IngestResult{Event: stored, Duplicate: duplicate}, err
}

func (s *Orders) Get(ctx context.Context, orderID string) (domain.Order, error) {
	events, err := s.store.EventsForOrder(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	if len(events) == 0 {
		return domain.Order{}, ErrOrderNotFound
	}
	acknowledged, err := s.store.AcknowledgedIssues(ctx, orderID)
	if err != nil {
		return domain.Order{}, err
	}
	order := domain.BuildOrder(orderID, events, s.now(), acknowledged)
	order.Summary, err = s.summarizer.Summarize(ctx, order)
	return order, err
}

func (s *Orders) List(ctx context.Context, status string) ([]domain.Order, error) {
	ids, err := s.store.ListOrderIDs(ctx)
	if err != nil {
		return nil, err
	}
	orders := make([]domain.Order, 0, len(ids))
	for _, id := range ids {
		order, getErr := s.Get(ctx, id)
		if getErr != nil {
			return nil, getErr
		}
		if status == "" || strings.EqualFold(string(order.Status), status) {
			orders = append(orders, order)
		}
	}
	return orders, nil
}

type ActionResult struct {
	Action   domain.Action `json:"action"`
	Replayed bool          `json:"replayed"`
}

func (s *Orders) RetryFulfillment(ctx context.Context, orderID, actor, idempotencyKey string) (ActionResult, error) {
	if _, err := s.Get(ctx, orderID); err != nil {
		return ActionResult{}, err
	}
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return ActionResult{}, errors.New("actor and idempotencyKey are required")
	}
	action := domain.Action{
		ID: uuid.NewString(), OrderID: orderID, Type: "RETRY_FULFILLMENT", Actor: actor,
		IdempotencyKey: idempotencyKey, CreatedAt: s.now().UTC(),
	}
	stored, replayed, err := s.store.SaveAction(ctx, action)
	if err != nil || replayed {
		return ActionResult{Action: stored, Replayed: replayed}, err
	}
	_, err = s.Ingest(ctx, IngestInput{
		OrderID: orderID, Type: string(domain.EventFulfillmentRetry), Source: "operator-console",
		IdempotencyKey: "action:" + stored.ID, OccurredAt: stored.CreatedAt,
		Metadata: map[string]any{"actor": actor, "actionId": stored.ID},
	})
	return ActionResult{Action: stored}, err
}

func (s *Orders) AcknowledgeIssue(ctx context.Context, orderID, issueCode, actor string) error {
	order, err := s.Get(ctx, orderID)
	if err != nil {
		return err
	}
	found := false
	for _, issue := range order.Issues {
		if issue.Code == issueCode {
			found = true
			break
		}
	}
	if !found {
		return errors.New("issue does not exist on this order")
	}
	if strings.TrimSpace(actor) == "" {
		return errors.New("actor is required")
	}
	return s.store.AcknowledgeIssue(ctx, orderID, issueCode, actor)
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
