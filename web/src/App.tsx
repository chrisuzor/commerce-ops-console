import { useCallback, useEffect, useMemo, useState } from "react";
import { acknowledgeIssue, listOrders, retryFulfillment } from "./api";
import { OrderDetail } from "./components/OrderDetail";
import { OrderList } from "./components/OrderList";
import type { Order } from "./types";
import "./styles.css";

export default function App() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [selectedId, setSelectedId] = useState<string>();
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string>();

  const load = useCallback(async () => {
    try {
      setError(undefined);
      const next = await listOrders();
      setOrders(next);
      setSelectedId((current) => current && next.some((item) => item.id === current) ? current : next[0]?.id);
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : "Unable to load orders");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    // Loading remote data is the synchronization this effect owns; state changes occur after the request resolves.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load();
  }, [load]);
  const selected = useMemo(() => orders.find((order) => order.id === selectedId), [orders, selectedId]);

  async function run(action: () => Promise<unknown>) {
    setBusy(true);
    setError(undefined);
    try { await action(); await load(); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "Action failed"); }
    finally { setBusy(false); }
  }

  if (loading) return <div className="state-page">Loading operations data…</div>;

  return (
    <div className="app-shell">
      <header className="topbar">
        <a className="brand" href="/">Commerce<span>Ops</span></a>
        <div className="topbar__meta"><span className="live-dot" /> Demo environment</div>
      </header>
      {error && <div className="error-banner" role="alert">{error}<button onClick={() => setError(undefined)}>Dismiss</button></div>}
      <div className="workspace">
        <OrderList orders={orders} selectedId={selectedId} onSelect={setSelectedId} />
        {selected ? (
          <OrderDetail
            order={selected}
            busy={busy}
            onRetry={() => run(() => retryFulfillment(selected.id))}
            onAcknowledge={(code) => run(() => acknowledgeIssue(selected.id, code))}
          />
        ) : <main className="state-page">Select an order to investigate.</main>}
      </div>
    </div>
  );
}
