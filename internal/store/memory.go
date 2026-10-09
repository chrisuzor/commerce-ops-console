package store

import (
	"context"
	"sort"
	"sync"

	"github.com/chrisuzor/commerce-ops-console/internal/domain"
)

type Memory struct {
	mu               sync.RWMutex
	eventsByOrder    map[string][]domain.Event
	eventByKey       map[string]domain.Event
	actionByKey      map[string]domain.Action
	acknowledgements map[string]map[string]bool
}

func NewMemory() *Memory {
	return &Memory{
		eventsByOrder: make(map[string][]domain.Event), eventByKey: make(map[string]domain.Event),
		actionByKey: make(map[string]domain.Action), acknowledgements: make(map[string]map[string]bool),
	}
}

func (m *Memory) SaveEvent(_ context.Context, event domain.Event) (domain.Event, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.eventByKey[event.IdempotencyKey]; ok {
		return existing, true, nil
	}
	m.eventByKey[event.IdempotencyKey] = event
	m.eventsByOrder[event.OrderID] = append(m.eventsByOrder[event.OrderID], event)
	return event, false, nil
}

func (m *Memory) EventsForOrder(_ context.Context, orderID string) ([]domain.Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.Event(nil), m.eventsByOrder[orderID]...), nil
}

func (m *Memory) ListOrderIDs(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.eventsByOrder))
	for id := range m.eventsByOrder {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

func (m *Memory) SaveAction(_ context.Context, action domain.Action) (domain.Action, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.actionByKey[action.IdempotencyKey]; ok {
		return existing, true, nil
	}
	m.actionByKey[action.IdempotencyKey] = action
	return action, false, nil
}

func (m *Memory) AcknowledgeIssue(_ context.Context, orderID, issueCode, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acknowledgements[orderID] == nil {
		m.acknowledgements[orderID] = make(map[string]bool)
	}
	m.acknowledgements[orderID][issueCode] = true
	return nil
}

func (m *Memory) AcknowledgedIssues(_ context.Context, orderID string) (map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]bool)
	for code, value := range m.acknowledgements[orderID] {
		result[code] = value
	}
	return result, nil
}

func (m *Memory) Close() {}
