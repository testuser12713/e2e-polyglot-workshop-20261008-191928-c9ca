import { useEffect, useState, type FormEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { ApiError } from "../../api/client";
import { useAuth } from "../../auth/AuthContext";
import Button from "../../components/Button";
import Card from "../../components/Card";
import FormField from "../../components/FormField";

/**
 * Login-specific styles. Global tokens and component classes live in
 * styles/global.css; the few classes only this screen uses are scoped here.
 */
const LOGIN_STYLES = `
  .login-page { max-width: 440px; margin: 0 auto; }
  .login-banner {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
    border-radius: var(--radius-md);
    padding: 12px 16px;
    font-size: var(--size-sm);
    line-height: 1.5;
    margin-bottom: var(--space-3);
    background: var(--color-danger-soft);
    border: 1px solid rgba(179, 38, 30, 0.35);
    color: var(--color-fg);
  }
  .login-banner__icon { width: 18px; height: 18px; flex: 0 0 18px; margin-top: 1px; }
  .login-banner__body { flex: 1; }
  .login-banner__code {
    display: block;
    font-size: var(--size-xs);
    color: var(--color-fg-muted);
    margin-top: 2px;
  }
  .login-submit { width: 100%; }
  html[data-login-route="true"] .app-nav,
  html[data-login-route="true"] .app-header__toggle { display: none; }
`;

interface LocationState {
  from?: { pathname?: string; search?: string; hash?: string };
}

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [errorCode, setErrorCode] = useState<string | null>(null);

  // The app-bar shows the wordmark only on this screen: the attribute hides
  // the navigation (and its mobile toggle) for as long as the page is mounted.
  useEffect(() => {
    document.title = "Anmelden — Werkstatt-Portal";
    document.documentElement.dataset.loginRoute = "true";
    return () => {
      delete document.documentElement.dataset.loginRoute;
      document.title = "Werkstatt-Portal";
    };
  }, []);

  const emailError = email.trim() ? undefined : "Bitte E-Mail-Adresse angeben.";
  const passwordError = password ? undefined : "Bitte Passwort angeben.";

  async function handleSubmit(event: FormEvent<HTMLFormElement>): Promise<void> {
    event.preventDefault();
    setSubmitted(true);
    setError(null);
    setErrorCode(null);

    if (emailError || passwordError) {
      return;
    }

    setSubmitting(true);
    try {
      await login(email.trim(), password);
      const state = location.state as LocationState | null;
      const from = state?.from;
      const target = from?.pathname
        ? `${from.pathname}${from.search ?? ""}${from.hash ?? ""}`
        : "/werkstatt";
      navigate(target, { replace: true });
    } catch (caught) {
      setError(
        caught instanceof ApiError ? caught.message : "Die Anmeldung ist fehlgeschlagen.",
      );
      setErrorCode(caught instanceof ApiError ? caught.code : null);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <section className="login-page">
      <style>{LOGIN_STYLES}</style>

      <div className="page-head">
        <h1 className="page__title">Werkstatt-Anmeldung</h1>
        <p className="page__lead">
          Dieser Bereich ist für Mitarbeiterinnen und Mitarbeiter der Werkstatt.
        </p>
      </div>

      <Card>
        <form noValidate onSubmit={handleSubmit}>
          {error && (
            <div
              id="login-error-banner"
              data-testid="login-error-banner"
              className="login-banner"
              role="alert"
            >
              <svg
                className="login-banner__icon"
                viewBox="0 0 20 20"
                fill="none"
                stroke="currentColor"
                strokeWidth={1.5}
                aria-hidden="true"
              >
                <path d="M10 6v5M10 14v.5" strokeLinecap="round" />
                <circle cx="10" cy="10" r="8" />
              </svg>
              <div className="login-banner__body">
                {error}
                {errorCode && <span className="login-banner__code">Code: {errorCode}</span>}
              </div>
            </div>
          )}

          <FormField label="E-Mail" htmlFor="email" error={emailError} submitted={submitted}>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="username"
              placeholder="mitarbeiter@werkstatt.de"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </FormField>

          <FormField label="Passwort" htmlFor="password" error={passwordError} submitted={submitted}>
            <input
              id="password"
              name="password"
              type="password"
              autoComplete="current-password"
              placeholder="••••••••"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </FormField>

          <Button
            type="submit"
            className="login-submit"
            loading={submitting}
            disabled={submitting}
          >
            Anmelden
          </Button>
        </form>
      </Card>
    </section>
  );
}
