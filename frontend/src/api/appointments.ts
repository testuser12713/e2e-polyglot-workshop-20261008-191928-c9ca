/**
 * Appointment intake API.
 *
 * Posts the customer's request to the shared endpoint
 * `POST /api/appointments` declared in the sprint interface and returns the
 * created order's id and human-readable order number.
 */

import { request } from "./client";

export interface AppointmentRequest {
  name: string;
  email: string;
  phone: string;
  plate: string;
  brand: string;
  model: string;
  /** Whole kilometres. */
  mileage: number;
  /** ISO date (YYYY-MM-DD) or an empty string when none was chosen. */
  preferred_date: string;
  description: string;
}

export interface AppointmentResponse {
  order_id: number;
  order_number: string;
}

/** Creates a new appointment request and returns the created order reference. */
export async function createAppointment(
  payload: AppointmentRequest,
): Promise<AppointmentResponse> {
  return request<AppointmentResponse>("/api/appointments", {
    method: "POST",
    body: payload,
  });
}
