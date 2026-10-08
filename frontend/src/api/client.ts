/// <reference types="vite/client" />

/**
 * The typed request helper every later api module builds on.
 *
 * - Base URL comes from `import.meta.env.VITE_API_BASE_URL` (declared in
 *   RUN.json as `${service:api.origin}`). No hard-coded localhost: until the
 *   api service is declared the variable is unset (or still holds an
 *   unresolved `${...}` placeholder) and we fall back to a same-origin
 *   relative base, which is a working default.
 * - Attaches the Bearer token when one is known.
 * - Parses the uniform error body `{"error":{"code","message"}}` into ApiError.
 */

export interface ApiErrorBody {
  error: { code: string; message: string };
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

function resolveBaseUrl(raw: string | undefined): string {
  if (!raw || raw.includes("${")) {
    return "";
  }
  return raw.replace(/\/+$/, "");
}

export const API_BASE_URL: string = resolveBaseUrl(import.meta.env.VITE_API_BASE_URL);

let authToken: string | null = null;

/** The bearer middleware shares one token across every api module. */
export function setAuthToken(token: string | null): void {
  authToken = token;
}

export function getAuthToken(): string | null {
  return authToken;
}

export interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
  /** Overrides the module token for this single call. */
  token?: string | null;
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, token, headers, ...rest } = options;
  const headerRecord: Record<string, string> = {
    Accept: "application/json",
    ...((headers as Record<string, string> | undefined) ?? {}),
  };

  const init: RequestInit = { ...rest, headers: headerRecord };

  if (body !== undefined) {
    headerRecord["Content-Type"] = "application/json";
    init.body = JSON.stringify(body);
  }

  const effectiveToken = token === undefined ? authToken : token;
  if (effectiveToken) {
    headerRecord.Authorization = `Bearer ${effectiveToken}`;
  }

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}${path}`, init);
  } catch {
    throw new ApiError(0, "network_error", "Die API ist nicht erreichbar.");
  }

  const text = await response.text();
  let payload: unknown = null;
  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = null;
    }
  }

  if (!response.ok) {
    const errorBody = payload as ApiErrorBody | null;
    throw new ApiError(
      response.status,
      errorBody?.error?.code ?? "http_error",
      errorBody?.error?.message ?? "Es ist ein Fehler aufgetreten.",
    );
  }

  return payload as T;
}

export interface HealthResponse {
  status: string;
}

/** Reachability probe behind the shell's API indicator. */
export async function getHealth(): Promise<HealthResponse> {
  const payload = await request<HealthResponse>("/api/health");
  // A dev server without the API in front of it answers the SPA fallback with
  // 200 text/html; only a real {"status":"ok"} body proves the API is up.
  if (!payload || typeof payload.status !== "string") {
    throw new ApiError(0, "invalid_health_response", "Die API ist nicht erreichbar.");
  }
  return payload;
}
