import type { Order } from "../types";
import { StatusBadge } from "./StatusBadge";

interface Props {
  orders: Order[];
  selectedId?: string;
  onSelect: (orderId: string) => void;
}

export function OrderList({ orders, selectedId, onSelect }: Props) {
  return (
    <aside className="order-list" aria-label="Orders">
      <div className="panel-heading">
        <div>
          <p className="eyebrow">Live operations</p>
          <h2>Orders</h2>
        </div>
        <span className="count">{orders.length}</span>
      </div>
      <div className="order-list__items">
        {orders.map((order) => (
          <button
            className={`order-card ${selectedId === order.id ? "order-card--selected" : ""}`}
            key={order.id}
            onClick={() => onSelect(order.id)}
          >
            <span>
              <strong>{order.id}</strong>
              <small>{order.events.length} timeline events</small>
            </span>
            <span className="order-card__meta">
              <StatusBadge status={order.status} />
              {order.issues.length > 0 && <small>{order.issues.length} issue{order.issues.length > 1 ? "s" : ""}</small>}
            </span>
          </button>
        ))}
      </div>
    </aside>
  );
}

