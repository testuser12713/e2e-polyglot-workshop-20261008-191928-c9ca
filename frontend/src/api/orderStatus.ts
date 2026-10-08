import { request } from "./client";
import type { OrderStatus } from "../components/StatusBadge";

/**
 * Customer order-status lookup — GET /api/orders/status.
 *
 * The shared interface of this sprint (order number + plate, no auth):
 *   200 {"order_number","status","history":[History],"vehicle":Vehicle,"invoice":Invoice|null}
 *   404 {"error":{"code","message"}}
 *
 * All amounts arrive as whole cents and are rendered as Euro by the helpers
 * below — never in the api layer itself (the api layer deals in cents only).
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

/* ------------------------------------------------------------------ */
/* Formatting — the UI renders Euro, never cents (AC-12).              */
/* ------------------------------------------------------------------ */

/**
 * Whole cents to German Euro: `123456` -> `1.234,56 €`, two decimals, comma
 * decimal separator, dot thousands separator (DESIGN.md number formatting).
 */
export function formatEuro(cents: number): string {
  const sign = cents < 0 ? "-" : "";
  const absolute = Math.abs(Math.round(cents));
  const euros = Math.floor(absolute / 100);
  const remainder = absolute % 100;
  return `${sign}${euros.toLocaleString("de-DE")},${String(remainder).padStart(2, "0")} €`;
}

/** ISO-8601 UTC instant to the fixed display time zone Europe/Berlin. */
export function formatDateTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "–";
  }
  return new Intl.DateTimeFormat("de-DE", {
    timeZone: "Europe/Berlin",
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

/** Labour time: `2.5` -> `2,5 Std.` (comma, one decimal, German unit). */
export function formatHours(hours: number): string {
  return `${hours.toFixed(1).replace(".", ",")} Std.`;
}

/** The Menge column: hours for labour, the count for a part, else an en dash. */
export function formatMenge(item: OrderItem): string {
  if (item.kind === "labor" && item.hours != null) {
    return formatHours(item.hours);
  }
  if (item.quantity != null) {
    return String(item.quantity);
  }
  return "–";
}
