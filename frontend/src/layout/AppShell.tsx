import { useEffect, useState } from "react";
import { Link, NavLink, Outlet } from "react-router-dom";

import { getHealth } from "../api/client";
import { useAuth } from "../auth/AuthContext";

type HealthState = "checking" | "ok" | "down";

const HEALTH_LABELS: Record<HealthState, string> = {
  checking: "API wird geprüft …",
  ok: "API erreichbar",
  down: "API nicht erreichbar",
};

function ApiStatusIndicator() {
  const [state, setState] = useState<HealthState>("checking");

  useEffect(() => {
    let active = true;
    getHealth()
      .then(() => {
        if (active) setState("ok");
      })
      .catch(() => {
        if (active) setState("down");
      });
    return () => {
      active = false;
    };
  }, []);

  return (
    <span
      className={`api-status api-status--${state}`}
      role="status"
      aria-live="polite"
      data-testid="api-status"
    >
      <span className="api-status__dot" aria-hidden="true" />
      {HEALTH_LABELS[state]}
    </span>
  );
}

/**
 * Shell chrome every page renders in (Design.md AppShell / Navigation):
 * sticky top bar, wordmark, quiet customer links, workshop account area and
 * the API-reachability indicator. On phone the nav collapses into a menu
 * button that opens a full-width panel; there is no horizontal scroll.
 */
export default function AppShell() {
  const { employee, logout } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <>
      <header className="app-header">
        <div className="app-header__inner">
          <Link to="/" className="wordmark">
            Werkstatt-Portal
          </Link>

          <button
            type="button"
            className="app-header__toggle"
            aria-label={menuOpen ? "Menü schließen" : "Menü öffnen"}
            aria-expanded={menuOpen}
            aria-controls="app-nav"
            onClick={() => setMenuOpen((open) => !open)}
          >
            ☰
          </button>

          <nav
            id="app-nav"
            className={menuOpen ? "app-nav app-nav--open" : "app-nav"}
            aria-label="Hauptnavigation"
          >
            <NavLink to="/" end className="app-nav__link">
              Termin anfragen
            </NavLink>
            <NavLink to="/status" className="app-nav__link">
              Statusabruf
            </NavLink>
            <NavLink to="/werkstatt" className="app-nav__link">
              Werkstatt
            </NavLink>

            {employee ? (
              <div className="app-nav__account">
                <span className="app-nav__email">{employee.email}</span>
                <button type="button" className="btn btn--ghost" onClick={logout}>
                  Abmelden
                </button>
              </div>
            ) : (
              <NavLink to="/werkstatt/login" className="app-nav__link">
                Werkstatt-Login
              </NavLink>
            )}

            <ApiStatusIndicator />
          </nav>
        </div>
      </header>

      <main className="app-main">
        <div className="container">
          <Outlet />
        </div>
      </main>
    </>
  );
}
