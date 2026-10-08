/**
 * Workshop order list API (sprint interface).
 *
 * GET  /api/workshop/orders?status=&plate=  -> 200 {"orders":[Order]}
 * POST /api/workshop/orders/{id}/confirm    -> 200 Order
 *
 * All requests go through the shared typed client so the Bearer token and the
 * uniform error body `{"error":{"code","message"}}` are handled in one place.
 */

import type { OrderStatus } from "../components/StatusBadge";
import { request } from "./client";

export interface Vehicle {
  id: number;
  plate: string;
  brand: string;
  model: string;
  mileage: number;
}

export interface Customer {
  id: number;
  name: string;
  email: string;
  phone: string;
}

export interface WorkshopOrder {
  id: number;
  order_number: string;
  status: OrderStatus;
  next_status: OrderStatus | null;
  preferred_date: string | null;
  description: string;
  created_at: string;
  vehicle: Vehicle;
  customer: Customer;
  labor_cents: number;
  parts_cents: number;
  net_cents: number;
  vat_cents: number;
  gross_cents: number;
}

export interface WorkshopOrderListResponse {
  orders: WorkshopOrder[];
}

/** `"all"` is the UI's "no status filter" sentinel and is never sent to the API. */
export interface WorkshopOrderQuery {
  status?: OrderStatus | "all";
  plate?: string;
}

function buildQuery({ status, plate }: WorkshopOrderQuery): string {
  const params = new URLSearchParams();
  if (status && status !== "all") {
    params.set("status", status);
  }
  const trimmed = plate?.trim();
  if (trimmed) {
    params.set("plate", trimmed);
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}

/** Lists the workshop's orders, optionally narrowed by status and plate. */
export async function listWorkshopOrders(query: WorkshopOrderQuery = {}): Promise<WorkshopOrder[]> {
  // The declared route is exactly "/api/workshop/orders"; the status and plate
  // filters travel as query parameters appended to that same path.
  const payload = await request<WorkshopOrderListResponse>("/api/workshop/orders" + buildQuery(query));
  // A dev server without the API in front of it answers the SPA fallback with
  // 200 text/html; only a real `orders` array proves the API responded.
  return Array.isArray(payload?.orders) ? payload.orders : [];
}

/** Confirms a requested order and returns its fresh state. */
export async function confirmWorkshopOrder(id: number): Promise<WorkshopOrder> {
  return request<WorkshopOrder>(`/api/workshop/orders/${id}/confirm`, { method: "POST" });
}
