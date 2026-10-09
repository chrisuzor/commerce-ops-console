import type { Order } from "./types";

const GRAPHQL_URL = import.meta.env.VITE_GRAPHQL_URL ?? "/graphql";

const ORDER_FIELDS = `
  id
  status
  summary
  issues { code title explanation severity evidence acknowledged }
  events { id type source occurredAt receivedAt metadata }
`;

async function execute<T>(query: string, variables: Record<string, unknown> = {}): Promise<T> {
  const response = await fetch(GRAPHQL_URL, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });
  const payload = (await response.json()) as { data?: T; errors?: Array<{ message: string }> };
  if (!response.ok || payload.errors?.length) {
    throw new Error(payload.errors?.map((item) => item.message).join("; ") || "GraphQL request failed");
  }
  if (!payload.data) throw new Error("GraphQL response did not include data");
  return payload.data;
}

export async function listOrders(): Promise<Order[]> {
  const data = await execute<{ orders: Order[] }>(`query Orders { orders { ${ORDER_FIELDS} } }`);
  return data.orders;
}

export async function retryFulfillment(orderId: string): Promise<boolean> {
  const idempotencyKey = `console-retry:${orderId}:${crypto.randomUUID()}`;
  const data = await execute<{ retryFulfillment: { replayed: boolean } }>(
    `mutation Retry($orderId: ID!, $actor: String!, $key: String!) {
      retryFulfillment(orderId: $orderId, actor: $actor, idempotencyKey: $key) { replayed }
    }`,
    { orderId, actor: "demo.operator@example.com", key: idempotencyKey },
  );
  return data.retryFulfillment.replayed;
}

export async function acknowledgeIssue(orderId: string, issueCode: string): Promise<void> {
  await execute<{ acknowledgeIssue: boolean }>(
    `mutation Acknowledge($orderId: ID!, $issueCode: String!, $actor: String!) {
      acknowledgeIssue(orderId: $orderId, issueCode: $issueCode, actor: $actor)
    }`,
    { orderId, issueCode, actor: "demo.operator@example.com" },
  );
}
