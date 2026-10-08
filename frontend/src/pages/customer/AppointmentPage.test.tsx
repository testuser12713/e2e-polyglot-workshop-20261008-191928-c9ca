import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import AppointmentPage from "./AppointmentPage";

function renderPage() {
  return render(
    <MemoryRouter>
      <AppointmentPage />
    </MemoryRouter>,
  );
}

/** Fills every required field of the form with valid values. */
async function fillValidForm(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText("Kennzeichen"), "M-KF 4711");
  await user.type(screen.getByLabelText("Marke"), "BMW");
  await user.type(screen.getByLabelText("Modell"), "320i");
  await user.type(screen.getByLabelText("Kilometerstand"), "128450");
  await user.type(screen.getByLabelText("Name"), "Erika Muster");
  await user.type(screen.getByLabelText("E-Mail"), "erika@example.com");
  await user.type(screen.getByLabelText("Telefon"), "0171 2345678");
  await user.type(screen.getByLabelText("Problembeschreibung"), "Inspektion fällig.");
}

function stubResponse(status: number, payload: unknown) {
  vi.stubGlobal(
    "fetch",
    vi.fn(() =>
      Promise.resolve({
        ok: status >= 200 && status < 300,
        status,
        text: () => Promise.resolve(JSON.stringify(payload)),
      }),
    ),
  );
}

beforeEach(() => {
  stubResponse(200, { status: "ok" });
});

describe("AppointmentPage", () => {
  it("starts neutral — no validation message on an untouched form (AC-24)", () => {
    renderPage();

    expect(screen.queryByText("Bitte geben Sie das Kennzeichen an.")).not.toBeInTheDocument();
    expect(screen.queryByText("Bitte geben Sie den Kilometerstand an.")).not.toBeInTheDocument();
    expect(screen.queryByText("Bitte geben Sie Ihren Namen an.")).not.toBeInTheDocument();
    expect(screen.queryByText("Bitte beschreiben Sie kurz Ihr Anliegen.")).not.toBeInTheDocument();
  });

  it("shows the plate message only after the field was touched and left empty", async () => {
    const user = userEvent.setup();
    renderPage();

    expect(screen.queryByText("Bitte geben Sie das Kennzeichen an.")).not.toBeInTheDocument();

    await user.click(screen.getByLabelText("Kennzeichen"));
    await user.tab();

    expect(
      await screen.findByText("Bitte geben Sie das Kennzeichen an."),
    ).toBeInTheDocument();
  });

  it("validates the mileage with its own message after touch", async () => {
    const user = userEvent.setup();
    renderPage();

    const mileage = screen.getByLabelText("Kilometerstand");
    await user.type(mileage, "12a");
    await user.tab();

    expect(
      await screen.findByText("Bitte geben Sie den Kilometerstand als ganze Zahl an."),
    ).toBeInTheDocument();
  });

  it("shows all messages after a submit attempt on an empty form", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.click(screen.getByRole("button", { name: "Termin anfragen" }));

    expect(await screen.findByText("Bitte geben Sie das Kennzeichen an.")).toBeInTheDocument();
    expect(screen.getByText("Bitte geben Sie Ihren Namen an.")).toBeInTheDocument();
    expect(screen.getByText("Bitte geben Sie eine gültige E-Mail-Adresse an.")).toBeInTheDocument();
    expect(screen.getByText("Bitte beschreiben Sie kurz Ihr Anliegen.")).toBeInTheDocument();
  });

  it("posts the values and shows the returned order number, pointing at /status", async () => {
    const user = userEvent.setup();
    stubResponse(201, { order_id: 7, order_number: "A-2026-0007" });
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Termin anfragen" }));

    const banner = await screen.findByTestId("success-banner");
    expect(banner).toHaveTextContent("A-2026-0007");
    expect(screen.getByRole("link", { name: "Status abrufen" })).toHaveAttribute(
      "href",
      "/status",
    );

    const fetchMock = fetch as unknown as ReturnType<typeof vi.fn>;
    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/appointments");
    expect(init.method).toBe("POST");
    expect(JSON.parse(init.body as string)).toMatchObject({
      plate: "M-KF 4711",
      mileage: 128450,
      name: "Erika Muster",
      email: "erika@example.com",
      description: "Inspektion fällig.",
    });
  });

  it("shows the API error message when the request is rejected", async () => {
    const user = userEvent.setup();
    stubResponse(409, {
      error: { code: "plate_taken", message: "Das Kennzeichen ist bereits vergeben." },
    });
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Termin anfragen" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Das Kennzeichen ist bereits vergeben.");
    expect(screen.queryByTestId("success-banner")).not.toBeInTheDocument();
  });

  it("disables the submit button while the request is in flight", async () => {
    const user = userEvent.setup();
    let resolveFetch: (value: {
      ok: boolean;
      status: number;
      text: () => Promise<string>;
    }) => void = () => undefined;
    vi.stubGlobal(
      "fetch",
      vi.fn(
        () =>
          new Promise((resolve) => {
            resolveFetch = resolve;
          }),
      ),
    );
    renderPage();

    await fillValidForm(user);
    await user.click(screen.getByRole("button", { name: "Termin anfragen" }));

    const button = screen.getByRole("button", { name: "Termin anfragen" });
    expect(button).toBeDisabled();
    expect(button).toHaveAttribute("aria-busy", "true");

    resolveFetch({
      ok: true,
      status: 201,
      text: () => Promise.resolve(JSON.stringify({ order_id: 1, order_number: "A-2026-0001" })),
    });

    expect(await screen.findByTestId("success-banner")).toHaveTextContent("A-2026-0001");
  });
});
