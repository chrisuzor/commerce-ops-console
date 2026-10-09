package store

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/chrisuzor/commerce-ops-console/internal/domain"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) SaveEvent(ctx context.Context, event domain.Event) (domain.Event, bool, error) {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return domain.Event{}, false, err
	}
	_, err = p.pool.Exec(ctx, `
		INSERT INTO order_events
			(id, order_id, event_type, source, idempotency_key, occurred_at, received_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, event.ID, event.OrderID, event.Type, event.Source, event.IdempotencyKey, event.OccurredAt, event.ReceivedAt, metadata)
	if err == nil {
		return event, false, nil
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23505" {
		return domain.Event{}, false, err
	}

	var existing domain.Event
	var raw []byte
	queryErr := p.pool.QueryRow(ctx, `
		SELECT id, order_id, event_type, source, idempotency_key, occurred_at, received_at, metadata
		FROM order_events WHERE idempotency_key = $1
	`, event.IdempotencyKey).Scan(
		&existing.ID, &existing.OrderID, &existing.Type, &existing.Source, &existing.IdempotencyKey,
		&existing.OccurredAt, &existing.ReceivedAt, &raw,
	)
	if queryErr != nil {
		return domain.Event{}, false, err
	}
	if unmarshalErr := json.Unmarshal(raw, &existing.Metadata); unmarshalErr != nil {
		return domain.Event{}, false, unmarshalErr
	}
	return existing, true, nil
}

func (p *Postgres) EventsForOrder(ctx context.Context, orderID string) ([]domain.Event, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, order_id, event_type, source, idempotency_key, occurred_at, received_at, metadata
		FROM order_events WHERE order_id = $1 ORDER BY occurred_at, received_at
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]domain.Event, 0)
	for rows.Next() {
		var event domain.Event
		var raw []byte
		if err = rows.Scan(&event.ID, &event.OrderID, &event.Type, &event.Source, &event.IdempotencyKey, &event.OccurredAt, &event.ReceivedAt, &raw); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &event.Metadata); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (p *Postgres) ListOrderIDs(ctx context.Context) ([]string, error) {
	rows, err := p.pool.Query(ctx, `SELECT DISTINCT order_id FROM order_events ORDER BY order_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (p *Postgres) SaveAction(ctx context.Context, action domain.Action) (domain.Action, bool, error) {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO operator_actions (id, order_id, action_type, actor, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, action.ID, action.OrderID, action.Type, action.Actor, action.IdempotencyKey, action.CreatedAt)
	if err == nil {
		return action, false, nil
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) || postgresError.Code != "23505" {
		return domain.Action{}, false, err
	}
	var existing domain.Action
	queryErr := p.pool.QueryRow(ctx, `
		SELECT id, order_id, action_type, actor, idempotency_key, created_at
		FROM operator_actions WHERE idempotency_key = $1
	`, action.IdempotencyKey).Scan(
		&existing.ID, &existing.OrderID, &existing.Type, &existing.Actor, &existing.IdempotencyKey, &existing.CreatedAt,
	)
	if queryErr != nil {
		return domain.Action{}, false, err
	}
	return existing, true, nil
}

func (p *Postgres) AcknowledgeIssue(ctx context.Context, orderID, issueCode, actor string) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO issue_acknowledgements (order_id, issue_code, actor)
		VALUES ($1, $2, $3)
		ON CONFLICT (order_id, issue_code) DO UPDATE SET actor = EXCLUDED.actor, acknowledged_at = now()
	`, orderID, issueCode, actor)
	return err
}

func (p *Postgres) AcknowledgedIssues(ctx context.Context, orderID string) (map[string]bool, error) {
	rows, err := p.pool.Query(ctx, `SELECT issue_code FROM issue_acknowledgements WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]bool)
	for rows.Next() {
		var code string
		if err = rows.Scan(&code); err != nil {
			return nil, err
		}
		result[code] = true
	}
	return result, rows.Err()
}

func (p *Postgres) Close() { p.pool.Close() }
