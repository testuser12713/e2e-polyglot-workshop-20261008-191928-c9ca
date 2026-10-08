import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import type { DashboardOrder } from "../../api/dashboard";
import DashboardPage from "./DashboardPage";

const SUMMARY = { open_orders: 4, done_today: 2, revenue_month_cents: 842000 };

const SAMPLE_ORDER: DashboardOrder = {
  id: 142,
  order_number: "A-2025-0142",
  status: "requested",
  next_status: "confirmed",
  preferred_date: "2025-03-18",
  description: "Inspektion",
  created_at: "2025-03-10T09:00:00Z",
  vehicle: { id: 1, plate: "M-KF 4711", brand: "VW", model: "Golf", mileage: 90000 },
  customer: { id: 1, name: "Michaela Berger", email: "m@example.com", phone: "0170 1" },
  labor_cents: 0,
  parts_cents: 0,
  net_cents: 0,
  vat_cents: 0,
  gross_cents: 0,
};

function response(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(JSON.stringify(body)),
  } as unknown as Response;
}

function installFetch(
  overrides: {
    dashboard?: () => Response;
    orders?: () => Response;
  } = {},
): void {
  vi.stubGlobal(
    "fetch",
    vi.fn((input: RequestInfo | URL) => {
      const url = String(input);
      if (url.includes("/api/workshop/dashboard")) {
        return Promise.resolve(overrides.dashboard ? overrides.dashboard() : response(SUMMARY));
      }
      if (url.includes("/api/workshop/orders")) {
        return Promise.resolve(
          overrides.orders ? overrides.orders() : response({ orders: [SAMPLE_ORDER] }),
        );
      }
      return Promise.resolve(response({ status: "ok" }));
    }),
  );
}

function renderPage() {
  return render(
    <MemoryRouter>
      <DashboardPage />
    </MemoryRouter>,
  );
}

describe("DashboardPage", () => {
  it("shows the three KPI figures from the API", async () => {
    installFetch();
    renderPage();

    expect(await screen.findByText("Offene Aufträge")).toBeInTheDocument();
    expect(screen.getByText("Heute fertig")).toBeInTheDocument();
    expect(screen.getByText("Umsatz laufender Monat")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("2")).toBeInTheDocument();
  });

  it("formats this month's revenue as Euro with German formatting", async () => {
    installFetch();
    renderPage();

    const value = await screen.findByText(/8\.420,00/);
    expect(value.textContent).toContain("€");
  });

  it("shows the API error message and a retry action when the request fails", async () => {
    let dashboardCalls = 0;
    installFetch({
      dashboard: () => {
        dashboardCalls += 1;
        if (dashboardCalls === 1) {
          return response(
            { error: { code: "db_down", message: "Datenbank nicht erreichbar" } },
            500,
          );
        }
        return response(SUMMARY);
      },
    });
    renderPage();

    expect(await screen.findByText("Datenbank nicht erreichbar")).toBeInTheDocument();
    expect(screen.queryByText("Offene Aufträge")).not.toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Erneut versuchen" }));

    expect(await screen.findByText("Offene Aufträge")).toBeInTheDocument();
  });

  it("lists the most recent orders with a link into the order detail", async () => {
    installFetch();
    renderPage();

    const links = await screen.findAllByRole("link", { name: "A-2025-0142" });
    expect(links.length).toBeGreaterThan(0);
    for (const link of links) {
      expect(link).toHaveAttribute("href", "/werkstatt/auftraege/142");
    }
  });

  it("shows the API error message instead of an empty order list when orders fail", async () => {
    installFetch({
      orders: () => response({ error: { code: "db_down", message: "Aufträge nicht verfügbar" } }, 500),
    });
    renderPage();

    expect(await screen.findByText("Aufträge nicht verfügbar")).toBeInTheDocument();
    expect(screen.queryByText("Keine Aufträge vorhanden.")).not.toBeInTheDocument();
  });
});
