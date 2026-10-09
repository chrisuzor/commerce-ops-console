export type OrderStatus = "PENDING" | "PAID" | "PROCESSING" | "NEEDS_ATTENTION" | "FULFILLED";

export interface OrderEvent {
  id: string;
  type: string;
  source: string;
  occurredAt: string;
  receivedAt: string;
  metadata: Record<string, unknown>;
}

export interface Issue {
  code: string;
  title: string;
  explanation: string;
  severity: "INFO" | "WARNING" | "CRITICAL";
  evidence: string[];
  acknowledged: boolean;
}

export interface Order {
  id: string;
  status: OrderStatus;
  summary: string;
  events: OrderEvent[];
  issues: Issue[];
}

