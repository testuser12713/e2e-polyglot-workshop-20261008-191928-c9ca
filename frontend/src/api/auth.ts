/**
 * Staff authentication against the Go API.
 *
 * `POST /api/auth/login` is the only login endpoint (Shared interface of the
 * sprint): it answers `{token, employee:{id,name,email}}` on success and the
 * uniform `{"error":{"code","message"}}` body with status 401 on bad
 * credentials — the client helper turns that into an ApiError.
 */

import { request } from "./client";

export interface Employee {
  id: number;
  name: string;
  email: string;
}

export interface LoginResponse {
  token: string;
  employee: Employee;
}

/** Signs a staff member in and returns the bearer token plus the employee. */
export function login(email: string, password: string): Promise<LoginResponse> {
  return request<LoginResponse>("/api/auth/login", {
    method: "POST",
    body: { email, password },
  });
}
