CREATE TABLE IF NOT EXISTS order_events (
    id UUID PRIMARY KEY,
    order_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    source TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    occurred_at TIMESTAMPTZ NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS order_events_order_timeline_idx
    ON order_events (order_id, occurred_at, received_at);

CREATE TABLE IF NOT EXISTS operator_actions (
    id UUID PRIMARY KEY,
    order_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    actor TEXT NOT NULL,
    idempotency_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS operator_actions_order_idx ON operator_actions (order_id, created_at);

CREATE TABLE IF NOT EXISTS issue_acknowledgements (
    order_id TEXT NOT NULL,
    issue_code TEXT NOT NULL,
    actor TEXT NOT NULL,
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id, issue_code)
);

