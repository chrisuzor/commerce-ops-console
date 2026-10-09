# Commerce Operations Console

A production-minded code sample for investigating and recovering failed commerce orders. It ingests events from payment, fulfillment, and notification systems, reconstructs each order's state, identifies operational issues, and gives an operator safe, idempotent recovery actions.

The project is deliberately more than CRUD: it demonstrates event-driven modeling, duplicate and out-of-order delivery handling, evidence-grounded summaries, GraphQL, PostgreSQL, observability, and a useful React interface.

## What it does

- Rebuilds order state from an append-only event history.
- Accepts events through a signed webhook or GraphQL mutation.
- Deduplicates deliveries with idempotency keys.
- Detects failures such as paid orders that were never fulfilled or customers that were never notified.
- Produces a deterministic, evidence-grounded incident summary without inventing facts.
- Lets operators retry fulfillment or acknowledge an issue, with an audit trail.
- Uses PostgreSQL in production mode and a seeded in-memory store for an instant demo.

## Quick start

Requirements: Go 1.25+ and Node.js 20+.

```bash
make install
make test
make run
```

Open [http://localhost:8080](http://localhost:8080). The default mode uses an in-memory repository and loads three representative orders.

For frontend development with hot reload, run `make dev-api` and `make dev-web` in separate terminals, then open [http://localhost:5173](http://localhost:5173).

## Run with PostgreSQL

```bash
docker compose up --build
```

The database initializes from `migrations/001_init.sql`; the application is available on [http://localhost:8080](http://localhost:8080).

## API examples

Query orders:

```bash
curl http://localhost:8080/graphql \
  -H 'Content-Type: application/json' \
  --data '{"query":"{ orders { id status issues { code message } summary { headline evidence } } }"}'
```

Ingest an event:

```bash
curl http://localhost:8080/webhooks/events \
  -H 'Content-Type: application/json' \
  -H 'X-Idempotency-Key: demo-payment-1' \
  --data '{"orderId":"ORD-5000","type":"payment.succeeded","source":"payments","occurredAt":"2026-10-09T10:00:00Z","metadata":{"amount":"49.00","currency":"USD"}}'
```

When `WEBHOOK_SECRET` is configured, sign the exact request bytes with HMAC-SHA256 and send the hex digest in `X-Webhook-Signature` (an optional `sha256=` prefix is accepted).

## Quality checks

```bash
make test       # Go and React tests
make lint       # go vet and ESLint
make build      # production frontend and Go binary
```

The CI workflow runs the same checks on every push and pull request.

## Design notes

See [ARCHITECTURE.md](ARCHITECTURE.md) for component boundaries, delivery semantics, trade-offs, and production follow-ups.

This is a portfolio project, not a hosted order processor. Authentication, authorization, retention policies, migrations tooling, and external fulfillment adapters would be required before production use.

## License

MIT
