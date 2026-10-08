import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, type InitialEntry } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuthProvider, useAuth } from "../../auth/AuthContext";
import RequireAuth from "../../auth/RequireAuth";
import LoginPage from "./LoginPage";

function LogoutProbe() {
  const { logout } = useAuth();
  return (
    <button type="button" onClick={logout}>
      Abmelden
    </button>
  );
}

interface FakeResponse {
  ok: boolean;
  status: number;
  text: () => Promise<string>;
}

function response(status: number, body: unknown): Promise<FakeResponse> {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    text: () => Promise.resolve(JSON.stringify(body)),
  });
}

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

function renderLogin(initialEntry: InitialEntry = "/werkstatt/login") {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <AuthProvider>
        <Routes>
          <Route path="/werkstatt/login" element={<LoginPage />} />
          <Route path="/werkstatt" element={<div>Dashboard-Ziel</div>} />
          <Route path="/werkstatt/auftraege" element={<div>Auftragsliste-Ziel</div>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe("LoginPage", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("renders the heading, description, both fields and the submit button", () => {
    renderLogin();

    expect(
      screen.getByRole("heading", { level: 1, name: "Werkstatt-Anmeldung" }),
    ).toBeInTheDocument();
    expect(screen.getByText(/Mitarbeiterinnen und Mitarbeiter/)).toBeInTheDocument();
    expect(screen.getByLabelText("E-Mail")).toBeInTheDocument();
    expect(screen.getByLabelText("Passwort")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Anmelden" })).toBeInTheDocument();
  });

  it("starts neutral: no validation message on an untouched form (AC-24)", () => {
    renderLogin();

    expect(screen.queryByText("Bitte E-Mail-Adresse angeben.")).not.toBeInTheDocument();
    expect(screen.queryByText("Bitte Passwort angeben.")).not.toBeInTheDocument();
  });

  it("shows the required-field messages once the form is submitted", async () => {
    const user = userEvent.setup();
    renderLogin();

    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(screen.getByText("Bitte E-Mail-Adresse angeben.")).toBeInTheDocument();
    expect(screen.getByText("Bitte Passwort angeben.")).toBeInTheDocument();
  });

  it("disables the submit button while the request runs (AC-23)", async () => {
    const user = userEvent.setup();
    const deferred = createDeferred<FakeResponse>();
    vi.stubGlobal("fetch", vi.fn(() => deferred.promise));
    renderLogin();

    await user.type(screen.getByLabelText("E-Mail"), "meister@werkstatt.de");
    await user.type(screen.getByLabelText("Passwort"), "geheim123");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(screen.getByRole("button", { name: "Anmelden" })).toBeDisabled();

    deferred.resolve({
      ok: true,
      status: 200,
      text: () =>
        Promise.resolve(
          JSON.stringify({
            token: "token-123",
            employee: { id: 1, name: "Meister", email: "meister@werkstatt.de" },
          }),
        ),
    });

    await screen.findByText("Dashboard-Ziel");
  });

  it("posts to /api/auth/login and returns to the protected target route", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(() =>
      response(200, {
        token: "token-123",
        employee: { id: 1, name: "Meister", email: "meister@werkstatt.de" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    renderLogin({
      pathname: "/werkstatt/login",
      state: { from: { pathname: "/werkstatt/auftraege" } },
    });

    await user.type(screen.getByLabelText("E-Mail"), "meister@werkstatt.de");
    await user.type(screen.getByLabelText("Passwort"), "geheim123");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    expect(await screen.findByText("Auftragsliste-Ziel")).toBeInTheDocument();
    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toContain("/api/auth/login");
    expect(init.method).toBe("POST");
    // The session survives a reload: the token is persisted, not only in state.
    expect(window.localStorage.getItem("werkstatt.token")).toBe("token-123");
  });

  it("shows the API message in the error banner and keeps the entered values", async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(() =>
        response(401, {
          error: { code: "invalid_credentials", message: "E-Mail oder Passwort ist falsch." },
        }),
      ),
    );
    renderLogin();

    const email = screen.getByLabelText("E-Mail");
    const password = screen.getByLabelText("Passwort");
    await user.type(email, "meister@werkstatt.de");
    await user.type(password, "falsch");
    await user.click(screen.getByRole("button", { name: "Anmelden" }));

    const banner = await screen.findByTestId("login-error-banner");
    expect(banner).toHaveTextContent("E-Mail oder Passwort ist falsch.");
    expect(banner).toHaveTextContent("Code: invalid_credentials");
    expect(banner).toHaveAttribute("role", "alert");
    expect(email).toHaveValue("meister@werkstatt.de");
    expect(password).toHaveValue("falsch");
  });

  it("clears the persisted token on logout", async () => {
    window.localStorage.setItem("werkstatt.token", "token-123");
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <AuthProvider>
          <LogoutProbe />
        </AuthProvider>
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Abmelden" }));

    expect(window.localStorage.getItem("werkstatt.token")).toBeNull();
  });
});

describe("RequireAuth", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("redirects an unauthenticated visitor to the login page", async () => {
    render(
      <MemoryRouter initialEntries={["/werkstatt/auftraege"]}>
        <AuthProvider>
          <Routes>
            <Route path="/werkstatt/login" element={<div>Login-Ziel</div>} />
            <Route
              path="/werkstatt/auftraege"
              element={
                <RequireAuth>
                  <div>Geschützt</div>
                </RequireAuth>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(screen.queryByText("Geschützt")).not.toBeInTheDocument();
    expect(await screen.findByText("Login-Ziel")).toBeInTheDocument();
  });

  it("renders the protected content when a token is present", async () => {
    window.localStorage.setItem("werkstatt.token", "token-123");

    render(
      <MemoryRouter initialEntries={["/werkstatt/auftraege"]}>
        <AuthProvider>
          <Routes>
            <Route path="/werkstatt/login" element={<div>Login-Ziel</div>} />
            <Route
              path="/werkstatt/auftraege"
              element={
                <RequireAuth>
                  <div>Geschützt</div>
                </RequireAuth>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Geschützt")).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText("Login-Ziel")).not.toBeInTheDocument());
  });
});
