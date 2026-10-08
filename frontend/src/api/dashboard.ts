import type { OrderStatus } from "../components/StatusBadge";
import { request } from "./client";

/**
 * Workshop dashboard data (shared interface: GET /api/workshop/dashboard).
 *
 * The dashboard endpoint returns only the three KPI figures. The "Neueste
 * Aufträge" section is fed by the existing workshop order list endpoint, so the
 * most recent orders are fetched separately and trimmed to the newest few.
 */

export interface DashboardSummary {
  open_orders: number;
  done_today: number;
  revenue_month_cents: number;
}

export interface DashboardVehicle {
  id: number;
  plate: string;
  brand: string;
  model: string;
  mileage: number;
}

export interface DashboardCustomer {
  id: number;
  name: string;
  email: string;
  phone: string;
}

/** The subset of the shared Order shape the dashboard renders. */
export interface DashboardOrder {
  id: number;
  order_number: string;
  status: OrderStatus;
  next_status: OrderStatus | null;
  preferred_date: string;
  description: string;
  created_at: string;
  vehicle: DashboardVehicle;
  customer: DashboardCustomer;
  labor_cents: number;
  parts_cents: number;
  net_cents: number;
  vat_cents: number;
  gross_cents: number;
}

interface OrdersResponse {
  orders: DashboardOrder[];
}

/** How many recent orders the dashboard lists. */
export const RECENT_ORDERS_LIMIT = 5;

export async function getDashboard(): Promise<DashboardSummary> {
  return request<DashboardSummary>("/api/workshop/dashboard");
}

function createdAtTime(value: string | null | undefined): number {
  const parsed = value ? Date.parse(value) : Number.NaN;
  return Number.isNaN(parsed) ? 0 : parsed;
}

export async function getRecentOrders(limit: number = RECENT_ORDERS_LIMIT): Promise<DashboardOrder[]> {
  const payload = await request<OrdersResponse>("/api/workshop/orders");
  const orders = Array.isArray(payload?.orders) ? payload.orders : [];
  return [...orders]
    .sort((a, b) => createdAtTime(b.created_at) - createdAtTime(a.created_at))
    .slice(0, Math.max(0, limit));
}
