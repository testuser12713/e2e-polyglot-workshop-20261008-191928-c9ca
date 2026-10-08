import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import OrderDetailPage, { euroToCents, formatCents } from "./OrderDetailPage";
import type { OrderDetail, OrderItem, WorkshopOrder } from "../../api/workshopOrderDetail";

/* -------------------------------------------------------------------------- */
/* Fixtures                                                                    */
/* -------------------------------------------------------------------------- */

function makeOrder(overrides: Partial<WorkshopOrder> = {}): WorkshopOrder {
  return {
    id: 5,
    order_number: "A-2025-0140",
    status: "in_progress",
    next_status: "done",
    preferred_date: "2025-03-14T00:00:00Z",
    description: "Zahnriemenwechsel inkl. Wasserpumpe",
    created_at: "2025-03-10T08:22:00Z",
    vehicle: {
      id: 1,
      plate: "M-AB 9910",
      brand: "Audi",
      model: "A4 Avant",
      mileage: 154980,
    },
    customer: {
      id: 1,
      name: "Familie Hofmann",
      email: "hofmann@example.com",
      phone: "0151 7788990",
    },
    labor_cents: 36800,
    parts_cents: 26350,
    net_cents: 63150,
    vat_cents: 11999,
    gross_cents: 75149,
    ...overrides,
  };
}

function makeItems(): OrderItem[] {
  return [
    {
      id: 1,
      kind: "labor",
      description: "Arbeitszeit — Zahnriemenwechsel",
      hours: 4,
      quantity: null,
      unit_price_cents: 9200,
      total_cents: 36800,
    },
    {
      id: 2,
      kind: "part",
      description: "Zahnriemensatz",
      hours: null,
      quantity: 1,
      unit_price_cents: 18900,
      total_cents: 18900,
    },
    {
      id: 3,
      kind: "part",
      description: "Wasserpumpe",
      hours: null,
      quantity: 1,
      unit_price_cents: 7450,
      total_cents: 7450,
    },
  ];
}

function makeDetail(overrides: Partial<WorkshopOrder> = {}): OrderDetail {
  return {
    order: makeOrder(overrides),
    items: makeItems(),
    history: [
      { status: "requested", changed_at: "2025-03-10T08:22:00Z" },
      { status: "confirmed", changed_at: "2025-03-10T14:05:00Z" },
      { status: "in_progress", changed_at: "2025-03-13T07:58:00Z" },
    ],
  };
}

function jsonResponse(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(JSON.stringify(body)),
  } as unknown as Response;
}

type FetchMock = ReturnType<typeof vi.fn>;

function stubFetch(handler: (url: string, method: string, body: unknown) => Response): FetchMock {
  const mock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const method = (init?.method ?? "GET").toUpperCase();
    const body = init?.body ? JSON.parse(init.body as string) : undefined;
    return handler(url, method, body);
  });
  vi.stubGlobal("fetch", mock);
  return mock as unknown as FetchMock;
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/werkstatt/auftraege/5"]}>
      <Routes>
        <Route path="/werkstatt/auftraege/:id" element={<OrderDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

/* -------------------------------------------------------------------------- */
/* Pure formatting helpers                                                     */
/* -------------------------------------------------------------------------- */

describe("money helpers", () => {
  it("formats whole cents in German notation", () => {
    expect(formatCents(75149)).toBe("751,49 €");
    expect(formatCents(123456)).toBe("1.234,56 €");
  });

  it("parses euro input with comma or dot into whole cents", () => {
    expect(euroToCents("92")).toBe(9200);
    expect(euroToCents("92,50")).toBe(9250);
    expect(euroToCents("45.00")).toBe(4500);
    expect(euroToCents("")).toBeNull();
    expect(euroToCents("abc")).toBeNull();
  });
});

/* -------------------------------------------------------------------------- */
/* Screen                                                                      */
/* -------------------------------------------------------------------------- */

describe("OrderDetailPage", () => {
  it("loads the order and renders its number, badge, positions and totals", async () => {
    stubFetch((url) => {
      if (url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(200, makeDetail());
      }
      throw new Error(`unexpected GET ${url}`);
    });

    renderPage();

    expect(await screen.findByRole("heading", { level: 1, name: "A-2025-0140" })).toBeInTheDocument();
    expect(screen.getAllByText("in Arbeit").length).toBeGreaterThan(0);
    expect(screen.getByText("Zahnriemensatz")).toBeInTheDocument();
    expect(screen.getAllByText("631,50 €").length).toBeGreaterThan(0);
    expect(screen.getByText("119,99 €")).toBeInTheDocument();
    expect(screen.getByText("751,49 €")).toBeInTheDocument();
    expect(screen.getByText("Fahrzeug & Kunde")).toBeInTheDocument();
    expect(screen.getAllByText("M-AB 9910").length).toBeGreaterThan(0);
  });

  it("adds a part position, converting the euro price to cents and showing new totals", async () => {
    const user = userEvent.setup();
    const updatedOrder = makeOrder({
      net_cents: 67650,
      vat_cents: 12854,
      gross_cents: 80504,
    });
    const fetchMock = stubFetch((url, method, body) => {
      if (method === "GET" && url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(200, makeDetail());
      }
      if (method === "POST" && url.endsWith("/api/workshop/orders/5/items")) {
        void body;
        return jsonResponse(201, {
          item: {
            id: 9,
            kind: "part",
            description: "Ölfilter",
            hours: null,
            quantity: 1,
            unit_price_cents: 4500,
            total_cents: 4500,
          },
          order: updatedOrder,
        });
      }
      throw new Error(`unexpected ${method} ${url}`);
    });

    renderPage();
    await screen.findByRole("heading", { level: 1, name: "A-2025-0140" });

    await user.click(screen.getByRole("button", { name: "Position hinzufügen" }));
    await user.type(screen.getByLabelText("Bezeichnung"), "Ölfilter");
    await user.type(screen.getByLabelText("Menge"), "1");
    await user.type(screen.getByLabelText("Einzelpreis (€)"), "45,00");
    await user.click(screen.getByRole("button", { name: "Speichern" }));

    expect(await screen.findByText("Ölfilter")).toBeInTheDocument();
    expect(await screen.findByText("805,04 €")).toBeInTheDocument();

    const postCall = fetchMock.mock.calls.find(
      (call) => (call[1] as RequestInit | undefined)?.method === "POST",
    );
    expect(postCall).toBeDefined();
    const sentBody = JSON.parse((postCall?.[1] as RequestInit).body as string);
    expect(sentBody).toEqual({
      kind: "part",
      description: "Ölfilter",
      hours: null,
      quantity: 1,
      unit_price_cents: 4500,
    });
  });

  it("edits an existing position through PUT", async () => {
    const user = userEvent.setup();
    const updatedOrder = makeOrder({ net_cents: 66350, vat_cents: 12607, gross_cents: 78957 });
    const fetchMock = stubFetch((url, method) => {
      if (method === "GET" && url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(200, makeDetail());
      }
      if (method === "PUT" && url.endsWith("/api/workshop/orders/5/items/1")) {
        return jsonResponse(200, {
          item: {
            id: 1,
            kind: "labor",
            description: "Arbeitszeit — Zahnriemenwechsel",
            hours: 4,
            quantity: null,
            unit_price_cents: 10000,
            total_cents: 40000,
          },
          order: updatedOrder,
        });
      }
      throw new Error(`unexpected ${method} ${url}`);
    });

    renderPage();
    await screen.findByRole("heading", { level: 1, name: "A-2025-0140" });

    await user.click(screen.getAllByRole("button", { name: "Position bearbeiten" })[0]);
    const priceInput = screen.getByLabelText("Stundensatz (€)");
    await user.clear(priceInput);
    await user.type(priceInput, "100,00");
    await user.click(screen.getByRole("button", { name: "Speichern" }));

    const putCall = fetchMock.mock.calls.find(
      (call) => (call[1] as RequestInit | undefined)?.method === "PUT",
    );
    expect(putCall).toBeDefined();
    const sentBody = JSON.parse((putCall?.[1] as RequestInit).body as string);
    expect(sentBody.unit_price_cents).toBe(10000);
    expect(sentBody.hours).toBe(4);
    expect(await screen.findByText("400,00 €")).toBeInTheDocument();
  });

  it("asks for confirmation before deleting a position", async () => {
    const user = userEvent.setup();
    const updatedOrder = makeOrder({ net_cents: 55700, vat_cents: 10583, gross_cents: 66283 });
    const fetchMock = stubFetch((url, method) => {
      if (method === "GET" && url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(200, makeDetail());
      }
      if (method === "DELETE" && url.endsWith("/api/workshop/orders/5/items/3")) {
        return jsonResponse(200, { order: updatedOrder });
      }
      throw new Error(`unexpected ${method} ${url}`);
    });

    renderPage();
    await screen.findByRole("heading", { level: 1, name: "A-2025-0140" });

    await user.click(screen.getAllByRole("button", { name: "Position löschen" })[2]);
    expect(screen.getByRole("dialog", { name: "Position löschen" })).toBeInTheDocument();
    expect(screen.getAllByText(/Wasserpumpe/).length).toBeGreaterThan(0);

    await user.click(screen.getByRole("button", { name: "Löschen" }));

    await waitFor(() => {
      expect(screen.queryByText("Wasserpumpe")).not.toBeInTheDocument();
    });
    expect(screen.getByText("662,83 €")).toBeInTheDocument();
    expect(
      fetchMock.mock.calls.some(
        (call) =>
          (call[1] as RequestInit | undefined)?.method === "DELETE" &&
          String(call[0]).endsWith("/api/workshop/orders/5/items/3"),
      ),
    ).toBe(true);
  });

  it("sets the next status reported by the API and disables the control at the end", async () => {
    const user = userEvent.setup();
    stubFetch((url, method, body) => {
      if (method === "GET" && url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(200, makeDetail());
      }
      if (method === "POST" && url.endsWith("/api/workshop/orders/5/status")) {
        expect(body).toEqual({ status: "done" });
        return jsonResponse(200, makeOrder({ status: "done", next_status: "picked_up" }));
      }
      throw new Error(`unexpected ${method} ${url}`);
    });

    renderPage();
    await screen.findByRole("heading", { level: 1, name: "A-2025-0140" });

    expect(screen.getByText(/Nächster erlaubter Schritt:/)).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Fertig setzen" }));
    expect(screen.getByRole("dialog", { name: "Status ändern" })).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Bestätigen" }));

    expect(await screen.findByRole("button", { name: "Abholen" })).toBeInTheDocument();
    expect(screen.getAllByText("fertig").length).toBeGreaterThan(0);
  });

  it("shows a disabled status control with an explanation at the end of the lifecycle", async () => {
    stubFetch((url) => {
      if (url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(
          200,
          makeDetail({ status: "picked_up", next_status: null }),
        );
      }
      throw new Error(`unexpected GET ${url}`);
    });

    renderPage();
    await screen.findByRole("heading", { level: 1, name: "A-2025-0140" });

    const terminal = screen.getByRole("button", { name: "Abgeschlossen" });
    expect(terminal).toBeDisabled();
    expect(screen.getByText(/abgeschlossen und abgeholt/)).toBeInTheDocument();
  });

  it("shows the API message when a status transition is rejected with 409", async () => {
    const user = userEvent.setup();
    stubFetch((url, method) => {
      if (method === "GET" && url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(200, makeDetail());
      }
      if (method === "POST" && url.endsWith("/api/workshop/orders/5/status")) {
        return jsonResponse(409, {
          error: { code: "invalid_transition", message: "Ungültiger Statuswechsel." },
        });
      }
      throw new Error(`unexpected ${method} ${url}`);
    });

    renderPage();
    await screen.findByRole("heading", { level: 1, name: "A-2025-0140" });

    await user.click(screen.getByRole("button", { name: "Fertig setzen" }));
    await user.click(screen.getByRole("button", { name: "Bestätigen" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Ungültiger Statuswechsel.");
    expect(screen.getByText("Code: invalid_transition")).toBeInTheDocument();
  });

  it("keeps validation quiet until the form is submitted (AC-24)", async () => {
    const user = userEvent.setup();
    stubFetch((url) => {
      if (url.endsWith("/api/workshop/orders/5")) {
        return jsonResponse(200, makeDetail());
      }
      throw new Error(`unexpected GET ${url}`);
    });

    renderPage();
    await screen.findByRole("heading", { level: 1, name: "A-2025-0140" });

    await user.click(screen.getByRole("button", { name: "Position hinzufügen" }));
    expect(screen.queryByText("Bezeichnung ist erforderlich.")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Speichern" }));
    expect(screen.getByText("Bezeichnung ist erforderlich.")).toBeInTheDocument();
    expect(screen.getByText("Menge ist erforderlich.")).toBeInTheDocument();
    expect(screen.getByText("Einzelpreis ist erforderlich.")).toBeInTheDocument();
  });

  it("renders a readable error state when the order cannot be loaded", async () => {
    stubFetch(() =>
      jsonResponse(404, { error: { code: "not_found", message: "Auftrag nicht gefunden." } }),
    );

    renderPage();
    expect(await screen.findByRole("alert")).toHaveTextContent("Auftrag nicht gefunden.");
  });
});
