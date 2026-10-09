package store

import (
	"context"

	"github.com/chrisuzor/commerce-ops-console/internal/domain"
)

type Store interface {
	SaveEvent(context.Context, domain.Event) (stored domain.Event, duplicate bool, err error)
	EventsForOrder(context.Context, string) ([]domain.Event, error)
	ListOrderIDs(context.Context) ([]string, error)
	SaveAction(context.Context, domain.Action) (stored domain.Action, duplicate bool, err error)
	AcknowledgeIssue(context.Context, string, string, string) error
	AcknowledgedIssues(context.Context, string) (map[string]bool, error)
	Close()
}
