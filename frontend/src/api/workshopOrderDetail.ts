/**
 * Typed client for a single workshop order (detail screen).
 *
 * Wraps the shared request helper from ./client. The response shapes follow
 * the sprint interface exactly:
 *   GET    /api/workshop/orders/{id}                    -> {order, items, history}
 *   POST   /api/workshop/orders/{id}/items              -> {item, order}
 *   PUT    /api/workshop/orders/{id}/items/{item_id}    -> {item, order}
 *   DELETE /api/workshop/orders/{id}/items/{item_id}    -> {order}
 *   POST   /api/workshop/orders/{id}/status             -> Order | 409
 *
 * All money is whole cents, times are ISO-8601 UTC.
 */

import { request } from "./client";
import type { OrderStatus } from "../components/StatusBadge";

export interface WorkshopCustomer {
  id: number;
  name: string;
  email: string;
  phone: string;
}

export interface WorkshopVehicle {
  id: number;
  plate: string;
  brand: string;
  model: string;
  mileage: number;
}

export interface WorkshopOrder {
  id: number;
  order_number: string;
  status: OrderStatus;
  next_status: OrderStatus | null;
  preferred_date: string;
  description: string;
  created_at: string;
  vehicle: WorkshopVehicle;
  customer: WorkshopCustomer;
  labor_cents: number;
  parts_cents: number;
  net_cents: number;
  vat_cents: number;
  gross_cents: number;
}

export type ItemKind = "labor" | "part";

export interface OrderItem {
  id: number;
  kind: ItemKind;
  description: string;
  hours: number | null;
  quantity: number | null;
  unit_price_cents: number;
  total_cents: number;
}

export interface OrderHistoryEntry {
  status: OrderStatus;
  changed_at: string;
}

export interface OrderDetail {
  order: WorkshopOrder;
  items: OrderItem[];
  history: OrderHistoryEntry[];
}

/** Body shared by the POST and PUT item endpoints. */
export interface ItemInput {
  kind: ItemKind;
  description: string;
  hours: number | null;
  quantity: number | null;
  unit_price_cents: number;
}

export interface ItemMutationResult {
  item: OrderItem;
  order: WorkshopOrder;
}

export interface OrderMutationResult {
  order: WorkshopOrder;
}

export function getOrder(orderId: string | number): Promise<OrderDetail> {
  return request<OrderDetail>(`/api/workshop/orders/${orderId}`);
}

export function addItem(orderId: string | number, input: ItemInput): Promise<ItemMutationResult> {
  return request<ItemMutationResult>(`/api/workshop/orders/${orderId}/items`, {
    method: "POST",
    body: input,
  });
}

export function updateItem(
  orderId: string | number,
  itemId: number,
  input: ItemInput,
): Promise<ItemMutationResult> {
  return request<ItemMutationResult>(`/api/workshop/orders/${orderId}/items/${itemId}`, {
    method: "PUT",
    body: input,
  });
}

export function deleteItem(
  orderId: string | number,
  itemId: number,
): Promise<OrderMutationResult> {
  return request<OrderMutationResult>(`/api/workshop/orders/${orderId}/items/${itemId}`, {
    method: "DELETE",
  });
}

export function setOrderStatus(
  orderId: string | number,
  status: OrderStatus,
): Promise<WorkshopOrder> {
  return request<WorkshopOrder>(`/api/workshop/orders/${orderId}/status`, {
    method: "POST",
    body: { status },
  });
}
