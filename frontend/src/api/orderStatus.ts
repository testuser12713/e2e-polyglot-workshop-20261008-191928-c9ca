import { request } from "./client";
import type { OrderStatus } from "../components/StatusBadge";

/**
 * Customer order-status lookup — GET /api/orders/status.
 *
 * The shared interface of this sprint (order number + plate, no auth):
 *   200 {"order_number","status","history":[History],"vehicle":Vehicle,"invoice":Invoice|null}
 *   404 {"error":{"code","message"}}
 *
 * This module speaks the wire format only: amounts stay whole cents, times stay
 * ISO-8601 UTC. Turning those into Euro and the German display format is the
 * page's job (StatusPage).
 */

export interface Vehicle {
  id: number;
  plate: string;
  brand: string;
  model: string;
  mileage: number;
}

export interface OrderItem {
  id: number;
  kind: "labor" | "part";
  description: string;
  hours: number | null;
  quantity: number | null;
  unit_price_cents: number;
  total_cents: number;
}

export interface StatusHistoryEntry {
  status: OrderStatus;
  changed_at: string;
}

export interface Invoice {
  items: OrderItem[];
  net_cents: number;
  vat_cents: number;
  gross_cents: number;
}

export interface OrderStatusResponse {
  order_number: string;
  status: OrderStatus;
  history: StatusHistoryEntry[];
  vehicle: Vehicle;
  invoice: Invoice | null;
}

export interface OrderStatusQuery {
  orderNumber: string;
  plate: string;
}

/** Live plate normalisation: uppercase, collapse runs of spaces, trim. */
export function normalizePlate(value: string): string {
  return value.toUpperCase().replace(/\s+/g, " ").trim();
}

/** GET /api/orders/status?order_number=&plate= — runs only when the form is submitted. */
export function getOrderStatus({
  orderNumber,
  plate,
}: OrderStatusQuery): Promise<OrderStatusResponse> {
  const params = new URLSearchParams({
    order_number: orderNumber.trim(),
    plate: normalizePlate(plate),
  });
  return request<OrderStatusResponse>(`/api/orders/status?${params.toString()}`);
}
