import type { Order } from "../types";
import { StatusBadge } from "./StatusBadge";

interface Props {
  order: Order;
  busy: boolean;
  onRetry: () => Promise<void>;
  onAcknowledge: (issueCode: string) => Promise<void>;
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

export function OrderDetail({ order, busy, onRetry, onAcknowledge }: Props) {
  return (
    <main className="order-detail">
      <header className="detail-header">
        <div>
          <p className="eyebrow">Order investigation</p>
          <h1>{order.id}</h1>
        </div>
        <StatusBadge status={order.status} />
      </header>

      <section className="summary" aria-labelledby="summary-heading">
        <div className="summary__icon">AI</div>
        <div>
          <p className="eyebrow" id="summary-heading">Evidence-grounded summary</p>
          <p>{order.summary}</p>
        </div>
      </section>

      <section className="section-block" aria-labelledby="issues-heading">
        <div className="section-heading">
          <div><p className="eyebrow">Detected automatically</p><h2 id="issues-heading">Operational issues</h2></div>
          {order.issues.length > 0 && (
            <button className="button button--primary" disabled={busy} onClick={() => void onRetry()}>
              {busy ? "Working…" : "Retry fulfillment"}
            </button>
          )}
        </div>
        {order.issues.length === 0 ? (
          <p className="empty">No active issues were detected from the available evidence.</p>
        ) : (
          <div className="issue-grid">
            {order.issues.map((issue) => (
              <article className={`issue issue--${issue.severity.toLowerCase()}`} key={issue.code}>
                <div className="issue__top">
                  <span>{issue.severity}</span>
                  <button
                    className="button button--quiet"
                    disabled={busy || issue.acknowledged}
                    onClick={() => void onAcknowledge(issue.code)}
                  >
                    {issue.acknowledged ? "Acknowledged" : "Acknowledge"}
                  </button>
                </div>
                <h3>{issue.title}</h3>
                <p>{issue.explanation}</p>
                <ul>{issue.evidence.map((item) => <li key={item}>{item}</li>)}</ul>
              </article>
            ))}
          </div>
        )}
      </section>

      <section className="section-block" aria-labelledby="timeline-heading">
        <div className="section-heading"><div><p className="eyebrow">Source evidence</p><h2 id="timeline-heading">Event timeline</h2></div></div>
        <ol className="timeline">
          {order.events.map((event) => (
            <li key={event.id}>
              <span className="timeline__dot" aria-hidden="true" />
              <div className="timeline__content">
                <div><strong>{event.type}</strong><span>{event.source}</span></div>
                <time dateTime={event.occurredAt}>{formatTime(event.occurredAt)}</time>
                {Object.keys(event.metadata).length > 0 && <code>{JSON.stringify(event.metadata)}</code>}
              </div>
            </li>
          ))}
        </ol>
      </section>
    </main>
  );
}

