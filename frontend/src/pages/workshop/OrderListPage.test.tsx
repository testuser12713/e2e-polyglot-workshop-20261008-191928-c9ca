import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { WorkshopOrder } from "../../api/workshopOrders";
import OrderListPage from "./OrderListPage";

const REQUESTED: WorkshopOrder = {
  id: 1,
  order_number: "A-2025-0142",
  status: "requested",
  next_status: "confirmed",
  preferred_date: "2025-03-18",
  description: "Bremsen quietschen",
  created_at: "2025-03-10T09:00:00Z",
  vehicle: { id: 11, plate: "M-KF 4711", brand: "VW", model: "Golf", mileage: 128450 },
  customer: { id: 21, name: "Michaela Berger", email: "michaela@example.de", phone: "0170" },
  labor_cents: 0,
  parts_cents: 0,
  net_cents: 0,
  vat_cents: 0,
  gross_cents: 0,
};

const CONFIRMED: WorkshopOrder = {
  ...REQUESTED,
  id: 2,
  order_number: "A-2025-0141",
  status: "confirmed",
  next_status: "in_progress",
  preferred_date: "2025-03-17",
  vehicle: { id: 12, plate: "B-KW 2384", brand: "Audi", model: "A3", mileage: 88200 },
  customer: { id: 22, name: "Thomas Kraus", email: "thomas@example.de", phone: "0171" },
};

function jsonResponse(body: unknown, ok = true, status = 200) {
  return Promise.resolve({
    ok,
    status,
    text: () => Promise.resolve(JSON.stringify(body)),
  });
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/werkstatt/auftraege"]}>
      <Routes>
        <Route path="/werkstatt/auftraege" element={<OrderListPage />} />
        <Route path="/werkstatt/auftraege/:id" element={<div>Detailseite</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

/** A stateful GET/POST stub that filters like the real endpoint. */
function installOrdersApi(orders: WorkshopOrder[] = [REQUESTED, CONFIRMED]) {
  const fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), "http://localhost");
    const method = (init?.method ?? "GET").toUpperCase();

    if (url.pathname === "/api/workshop/orders" && method === "GET") {
      let list = orders;
      const status = url.searchParams.get("status");
      const plate = url.searchParams.get("plate");
      if (status) {
        list = list.filter((order) => order.status === status);
      }
      if (plate) {
        const needle = plate.toLowerCase();
        list = list.filter((order) => order.vehicle.plate.toLowerCase().includes(needle));
      }
      return jsonResponse({ orders: list });
    }

    const confirm = /^\/api\/workshop\/orders\/(\d+)\/confirm$/.exec(url.pathname);
    if (confirm && method === "POST") {
      const id = Number(confirm[1]);
      const base = orders.find((order) => order.id === id) ?? orders[0];
      return jsonResponse({ ...base, status: "confirmed", next_status: "in_progress" });
    }

    return jsonResponse({ error: { code: "not_found", message: "Nicht gefunden" } }, false, 404);
  });

  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn(() => jsonResponse({ status: "ok" })),
  );
});

describe("OrderListPage", () => {
  it("loads the orders and renders the screen's columns", async () => {
    installOrdersApi();
    renderPage();

    const table = await screen.findByTestId("order-list-table");

    expect(within(table).getByRole("columnheader", { name: "Auftragsnummer" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Kennzeichen" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Kunde" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Status" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Wunschtermin" })).toBeInTheDocument();
    expect(within(table).getByRole("columnheader", { name: "Aktionen" })).toBeInTheDocument();

    expect(within(table).getByText("A-2025-0142")).toBeInTheDocument();
    expect(within(table).getByText("M-KF 4711")).toBeInTheDocument();
    expect(within(table).getByText("Michaela Berger")).toBeInTheDocument();
    expect(within(table).getByText("angefragt")).toBeInTheDocument();
    expect(within(table).getByText("18.03.2025")).toBeInTheDocument();
  });

  it("shows the empty state with its primary reset action", async () => {
    installOrdersApi([]);
    renderPage();

    expect(await screen.findByText("Keine Aufträge für diesen Filter")).toBeInTheDocument();
    const empty = screen.getByTestId("order-empty");
    expect(within(empty).getByRole("button", { name: "Filter zurücksetzen" })).toBeInTheDocument();
  });

  it("narrows the list by status and shows the active filter chip", async () => {
    const user = userEvent.setup();
    const fetchMock = installOrdersApi();
    renderPage();

    await screen.findByTestId("order-list-table");
    await user.click(screen.getByRole("button", { name: "angefragt" }));

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("status=requested"),
        expect.anything(),
      ),
    );

    const table = screen.getByTestId("order-list-table");
    await waitFor(() => expect(within(table).queryByText("A-2025-0141")).not.toBeInTheDocument());
    expect(within(table).getByText("A-2025-0142")).toBeInTheDocument();
    expect(screen.getByText("Status: angefragt")).toBeInTheDocument();
  });

  it("debounces the plate search and narrows by plate", async () => {
    const user = userEvent.setup({ delay: null });
    const fetchMock = installOrdersApi();
    renderPage();

    await screen.findByTestId("order-list-table");
    expect(fetchMock).toHaveBeenCalledTimes(1);

    await user.type(screen.getByLabelText("Kennzeichen suchen"), "M-KF");
    // One request for the whole typing burst — not one per keystroke.
    expect(fetchMock).toHaveBeenCalledTimes(1);

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("plate=M-KF"),
        expect.anything(),
      ),
    );

    const table = screen.getByTestId("order-list-table");
    await waitFor(() => expect(within(table).queryByText("A-2025-0141")).not.toBeInTheDocument());
    expect(within(table).getByText("A-2025-0142")).toBeInTheDocument();
  });

  it("removes an active filter chip and resets all filters", async () => {
    const user = userEvent.setup();
    const fetchMock = installOrdersApi();
    renderPage();

    await screen.findByTestId("order-list-table");
    await user.click(screen.getByRole("button", { name: "angefragt" }));
    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("status=requested"),
        expect.anything(),
      ),
    );

    await user.click(screen.getByRole("button", { name: "Statusfilter angefragt entfernen" }));
    await waitFor(() =>
      expect(fetchMock).toHaveBeenLastCalledWith(
        expect.not.stringContaining("status="),
        expect.anything(),
      ),
    );

    await user.click(screen.getAllByRole("button", { name: "Filter zurücksetzen" })[0]);
    await waitFor(() => expect(screen.getByRole("button", { name: "Alle" })).toHaveAttribute("aria-pressed", "true"));
  });

  it("confirms a requested order and refreshes its row", async () => {
    const user = userEvent.setup();
    const fetchMock = installOrdersApi();
    renderPage();

    const table = await screen.findByTestId("order-list-table");
    await user.click(within(table).getByTestId("confirm-1"));

    await waitFor(() =>
      expect(fetchMock).toHaveBeenCalledWith(
        expect.stringContaining("/api/workshop/orders/1/confirm"),
        expect.objectContaining({ method: "POST" }),
      ),
    );

    await waitFor(() => expect(within(table).getByTestId("confirm-1")).toBeDisabled());
    expect(within(table).getAllByText("bestätigt")).toHaveLength(2);
  });

  it("shows an already-confirmed order's confirm action disabled with an explanation", async () => {
    installOrdersApi();
    renderPage();

    const table = await screen.findByTestId("order-list-table");
    const button = within(table).getByTestId("confirm-2");

    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("title", "Auftrag ist bereits bestätigt.");
  });

  it("navigates to the order detail route when a row is clicked", async () => {
    const user = userEvent.setup();
    installOrdersApi();
    renderPage();

    const table = await screen.findByTestId("order-list-table");
    await user.click(within(table).getByText("A-2025-0142"));

    expect(await screen.findByText("Detailseite")).toBeInTheDocument();
  });

  it("shows the API error message with a retry action", async () => {
    const user = userEvent.setup();
    let failing = true;
    const fetchMock = vi.fn(() => {
      if (failing) {
        return jsonResponse(
          { error: { code: "internal", message: "Interner Fehler" } },
          false,
          500,
        );
      }
      return jsonResponse({ orders: [REQUESTED] });
    });
    vi.stubGlobal("fetch", fetchMock);
    renderPage();

    expect(await screen.findByText("Interner Fehler")).toBeInTheDocument();
    expect(screen.getByText("Code: internal")).toBeInTheDocument();

    failing = false;
    await user.click(screen.getByRole("button", { name: "Erneut versuchen" }));

    expect(await screen.findByTestId("order-list-table")).toBeInTheDocument();
  });
});
