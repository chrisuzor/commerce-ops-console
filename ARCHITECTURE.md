# Architecture

## Request and data flow

```text
commerce systems
      |
      | signed events
      v
HTTP webhook / GraphQL mutation
      |
      v
Orders service -----> idempotent event store (memory or PostgreSQL)
      |                           |
      |                           v
      +-----------------> ordered event history
                                  |
                                  v
                        deterministic state reducer
                                  |
                 +----------------+----------------+
                 v                                 v
          issue detection                 evidence summarizer
                 |                                 |
                 +----------------+----------------+
                                  v
                         GraphQL API + React UI
```

## Main decisions

### Events are the source of truth

The store retains immutable events rather than overwriting a mutable order row. The domain reducer sorts by business occurrence time, with receipt time as a tie-breaker, and derives the current state. This makes an incident explainable and allows a late event to correct the projection.

### At-least-once delivery is expected

Webhook providers retry. `idempotency_key` is unique in both stores, so receiving the same logical event twice returns the original event and does not change the projection. Operator actions use the same pattern to make retry clicks safe.

### The AI-shaped feature is constrained by evidence

The summarizer is intentionally deterministic. Every statement is derived from stored events and each summary exposes its evidence. This demonstrates the reliability boundary an LLM implementation should preserve: retrieve a bounded evidence set, require structured output, validate references, and fall back to deterministic logic. A model can later improve phrasing without becoming the authority on order state.

### Storage is behind an interface

The service depends on `store.Store`, not PostgreSQL. The in-memory implementation makes domain and service behavior fast to test; PostgreSQL supplies durable constraints for deployment.

## Consistency and failure handling

- Unknown event types are rejected at the boundary.
- Future-dated events beyond the accepted clock-skew window are rejected.
- Webhook bodies are limited to 1 MiB.
- HMAC comparison uses constant-time equality.
- Duplicate-key handling only treats PostgreSQL `23505` errors as idempotent replays; unrelated database failures remain visible.
- HTTP timeouts and graceful shutdown bound resource use and in-flight work.
- Recovery actions are written before their derived events, providing an auditable operator trail.

## Trade-offs

- Replaying an order's full history is simple and auditable but becomes expensive for very long streams. A production version would snapshot projections after a measured threshold.
- The demo executes recovery locally. A production adapter would use an outbox table plus a worker so database state and external requests cannot diverge.
- Listing orders currently discovers distinct order IDs and projects each order. At larger scale, a continuously updated read model would support pagination and filtering efficiently.
- CORS is intentionally narrow and configurable. Production authentication and per-action authorization are outside this sample's scope.

## Production follow-ups

1. OIDC authentication and role-based authorization for operator actions.
2. Transactional outbox, worker retries with jitter, and a dead-letter queue.
3. Schema-managed migrations and automated rollback checks.
4. OpenTelemetry traces, RED metrics, SLOs, and alerting.
5. Paginated materialized projections and retention/archival policies.
6. Model-backed summaries behind evaluation, schema validation, citations, and deterministic fallback.
