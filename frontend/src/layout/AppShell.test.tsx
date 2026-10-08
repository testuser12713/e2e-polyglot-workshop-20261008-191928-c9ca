import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuthProvider } from "../auth/AuthContext";
import AppShell from "./AppShell";

function mockHealth(ok: boolean) {
  vi.stubGlobal(
    "fetch",
    vi.fn(() =>
      ok
        ? Promise.resolve({
            ok: true,
            status: 200,
            text: () => Promise.resolve(JSON.stringify({ status: "ok" })),
          })
        : Promise.reject(new Error("network down")),
    ),
  );
}

function renderShell() {
  return render(
    <MemoryRouter>
      <AuthProvider>
        <AppShell />
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe("AppShell", () => {
  beforeEach(() => {
    mockHealth(true);
  });

  it("renders the wordmark and the navigation links", async () => {
    renderShell();

    expect(screen.getByText("Werkstatt-Portal")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Termin anfragen" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Statusabruf" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Hauptnavigation" })).toBeInTheDocument();
    await screen.findByText("API erreichbar");
  });

  it("toggles the mobile menu panel", async () => {
    const user = userEvent.setup();
    renderShell();

    const toggle = screen.getByRole("button", { name: "Menü öffnen" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    await user.click(toggle);

    const close = screen.getByRole("button", { name: "Menü schließen" });
    expect(close).toHaveAttribute("aria-expanded", "true");
    await screen.findByText("API erreichbar");
  });

  it("reports the API as reachable when GET /api/health answers", async () => {
    renderShell();

    expect(await screen.findByText("API erreichbar")).toBeInTheDocument();
    const calledUrl = String((globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]);
    expect(calledUrl).toContain("/api/health");
  });

  it("reports the API as unreachable when the health request fails", async () => {
    mockHealth(false);
    renderShell();

    expect(await screen.findByText("API nicht erreichbar")).toBeInTheDocument();
  });
});
