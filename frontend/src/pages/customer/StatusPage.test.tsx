import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import StatusPage from "./StatusPage";

function jsonResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(JSON.stringify(body)),
  } as Response;
}

const successBody = {
  order_number: "A-2025-0139",
  status: "done",
  history: [
    { status: "requested", changed_at: "2025-03-05T09:14:00Z" },
    { status: "confirmed", changed_at: "2025-03-05T11:30:00Z" },
    { status: "in_progress", changed_at: "2025-03-11T08:05:00Z" },
    { status: "done", changed_at: "2025-03-12T15:42:00Z" },
  ],
  vehicle: { id: 1, plate: "F-RS 6207", brand: "Ford", model: "Focus", mileage: 128450 },
  invoice: {
    items: [
      {
        id: 1,
        kind: "labor",
        description: "Arbeitszeit — Bremsenservice",
        hours: 2.5,
        quantity: null,
        unit_price_cents: 9200,
        total_cents: 23000,
      },
      {
        id: 2,
        kind: "part",
        description: "Bremsbeläge vorn",
        hours: null,
        quantity: 1,
        unit_price_cents: 8450,
        total_cents: 8450,
      },
    ],
    net_cents: 31450,
    vat_cents: 5975,
    gross_cents: 37425,
  },
};

function mockFetchOnce(response: Response) {
  const fetchMock = vi.fn((_input: string, _init?: RequestInit) =>
    Promise.resolve(response),
  );
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(jsonResponse(200, successBody))));
});

describe("StatusPage lookup form", () => {
  it("renders the title, description and a neutral form (AC-24)", () => {
    render(<StatusPage />);

    expect(screen.getByRole("heading", { level: 1, name: "Status abrufen" })).toBeInTheDocument();
    expect(screen.getByLabelText("Auftragsnummer")).toBeInTheDocument();
    expect(screen.getByLabelText("Kennzeichen")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Status abrufen" })).toBeInTheDocument();
    expect(screen.queryByText("Bitte Auftragsnummer angeben.")).not.toBeInTheDocument();
    expect(screen.queryByText("Bitte Kennzeichen angeben.")).not.toBeInTheDocument();
  });

  it("shows the validation messages only after a submit attempt and does not call the API", async () => {
    const user = userEvent.setup();
    const fetchMock = mockFetchOnce(jsonResponse(200, successBody));
    render(<StatusPage />);

    await user.click(screen.getByRole("button", { name: "Status abrufen" }));

    expect(screen.getByText("Bitte Auftragsnummer angeben.")).toBeInTheDocument();
    expect(screen.getByText("Bitte Kennzeichen angeben.")).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("auto-uppercases the plate while typing", async () => {
    const user = userEvent.setup();
    render(<StatusPage />);

    const plate = screen.getByLabelText("Kennzeichen");
    await user.type(plate, "f-rs 6207");

    expect(plate).toHaveValue("F-RS 6207");
  });
});

describe("StatusPage successful lookup", () => {
  it("requests the declared endpoint and shows status, history and invoice (AC-12)", async () => {
    const user = userEvent.setup();
    const fetchMock = mockFetchOnce(jsonResponse(200, successBody));
    render(<StatusPage />);

    await user.type(screen.getByLabelText("Auftragsnummer"), "A-2025-0139");
    await user.type(screen.getByLabelText("Kennzeichen"), "f-rs 6207");
    await user.click(screen.getByRole("button", { name: "Status abrufen" }));

    expect((await screen.findAllByText("fertig")).length).toBeGreaterThan(0);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe(
      "/api/orders/status?order_number=A-2025-0139&plate=F-RS+6207",
    );

    const timeline = screen.getByTestId("status-timeline");
    expect(timeline).toBeInTheDocument();
    expect(screen.getByText("angefragt")).toBeInTheDocument();
    expect(screen.getByText("05.03.2025, 10:14")).toBeInTheDocument();
    expect(screen.getByText("12.03.2025, 16:42")).toBeInTheDocument();

    expect(screen.getByRole("heading", { name: "Rechnung" })).toBeInTheDocument();
    expect(screen.getByText("Arbeitszeit — Bremsenservice")).toBeInTheDocument();
    expect(screen.getByText("2,5 Std.")).toBeInTheDocument();
    expect(screen.getByText("92,00 €")).toBeInTheDocument();
    expect(screen.getByText("230,00 €")).toBeInTheDocument();
    expect(screen.getByText("MwSt. 19 %")).toBeInTheDocument();
    expect(screen.getByText("314,50 €")).toBeInTheDocument();
    expect(screen.getByText("59,75 €")).toBeInTheDocument();
    expect(screen.getByText("374,25 €")).toBeInTheDocument();
  });

  it("shows the quiet empty state while there is no invoice yet", async () => {
    const user = userEvent.setup();
    mockFetchOnce(jsonResponse(200, { ...successBody, status: "in_progress", invoice: null }));
    render(<StatusPage />);

    await user.type(screen.getByLabelText("Auftragsnummer"), "A-2025-0139");
    await user.type(screen.getByLabelText("Kennzeichen"), "F-RS 6207");
    await user.click(screen.getByRole("button", { name: "Status abrufen" }));

    expect(await screen.findByText("Noch keine Rechnung vorhanden")).toBeInTheDocument();
    expect(screen.queryByText("MwSt. 19 %")).not.toBeInTheDocument();
  });
});

describe("StatusPage error handling", () => {
  it("shows the API 404 message inline and keeps the entered values", async () => {
    const user = userEvent.setup();
    mockFetchOnce(
      jsonResponse(404, {
        error: {
          code: "order_not_found",
          message: "Auftrag mit dieser Auftragsnummer und diesem Kennzeichen wurde nicht gefunden.",
        },
      }),
    );
    render(<StatusPage />);

    await user.type(screen.getByLabelText("Auftragsnummer"), "A-0000-0000");
    await user.type(screen.getByLabelText("Kennzeichen"), "F-RS 6207");
    await user.click(screen.getByRole("button", { name: "Status abrufen" }));

    const banner = await screen.findByRole("alert");
    expect(banner).toHaveAttribute("id", "lookup-error-banner");
    expect(
      screen.getByText(
        "Auftrag mit dieser Auftragsnummer und diesem Kennzeichen wurde nicht gefunden.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Code: order_not_found")).toBeInTheDocument();

    expect(screen.getByLabelText("Auftragsnummer")).toHaveValue("A-0000-0000");
    expect(screen.getByLabelText("Kennzeichen")).toHaveValue("F-RS 6207");
  });
});
