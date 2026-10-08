import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import App from "./App";
import { AuthProvider } from "./auth/AuthContext";

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthProvider>
        <App />
      </AuthProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  // The workshop routes sit behind RequireAuth now, so the route-table tests
  // seed a session the same way a real login would; the unauthenticated
  // redirect is covered in LoginPage.test.tsx.
  window.localStorage.clear();
  window.localStorage.setItem("werkstatt.token", "test-token");
  vi.stubGlobal(
    "fetch",
    vi.fn(() =>
      Promise.resolve({
        ok: true,
        status: 200,
        text: () => Promise.resolve(JSON.stringify({ status: "ok" })),
      }),
    ),
  );
});

describe("App route table", () => {
  it("renders the appointment request page at /", async () => {
    renderAt("/");
    expect(
      await screen.findByRole("heading", { level: 1, name: "Termin anfragen" }),
    ).toBeInTheDocument();
  });

  it("renders the status page at /status", async () => {
    renderAt("/status");
    expect(
      await screen.findByRole("heading", { level: 1, name: "Status und Rechnung" }),
    ).toBeInTheDocument();
  });

  it("renders the login page at /werkstatt/login", async () => {
    renderAt("/werkstatt/login");
    expect(
      await screen.findByRole("heading", { level: 1, name: "Werkstatt-Anmeldung" }),
    ).toBeInTheDocument();
  });

  it("wraps /werkstatt in the RequireAuth guard", async () => {
    renderAt("/werkstatt");
    expect(
      await screen.findByRole("heading", { level: 1, name: "Dashboard" }),
    ).toBeInTheDocument();
  });

  it("renders the order list at /werkstatt/auftraege", async () => {
    renderAt("/werkstatt/auftraege");
    expect(await screen.findByRole("heading", { level: 1, name: "Aufträge" })).toBeInTheDocument();
  });

  it("renders the order detail with its route parameter at /werkstatt/auftraege/:id", async () => {
    renderAt("/werkstatt/auftraege/42");
    expect(
      await screen.findByRole("heading", { level: 1, name: "Auftrag 42" }),
    ).toBeInTheDocument();
  });

  it("redirects an unknown route to the appointment page", async () => {
    renderAt("/gibt-es-nicht");
    expect(
      await screen.findByRole("heading", { level: 1, name: "Termin anfragen" }),
    ).toBeInTheDocument();
  });

  it("shows the API-reachability indicator fed by /api/health", async () => {
    renderAt("/");
    expect(await screen.findByText("API erreichbar")).toBeInTheDocument();
  });
});
