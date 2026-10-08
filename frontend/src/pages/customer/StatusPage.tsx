import { useState, type FormEvent } from "react";

import { ApiError } from "../../api/client";
import {
  formatDateTime,
  formatEuro,
  formatMenge,
  getOrderStatus,
  normalizePlate,
  type Invoice,
  type OrderStatusResponse,
  type StatusHistoryEntry,
} from "../../api/orderStatus";
import Button from "../../components/Button";
import Card from "../../components/Card";
import FormField from "../../components/FormField";
import StatusBadge, { STATUS_LABELS } from "../../components/StatusBadge";

/**
 * Customer status lookup (GET /api/orders/status). The search runs only on
 * submit; validation messages appear only once a field was touched or the form
 * was submitted (AC-24). A wrong combination keeps the entered values and shows
 * the API's 404 message in the inline banner under the form.
 */
export default function StatusPage() {
  const [orderNumber, setOrderNumber] = useState("");
  const [plate, setPlate] = useState("");
  const [submitted, setSubmitted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<OrderStatusResponse | null>(null);
  const [apiError, setApiError] = useState<{ message: string; code: string } | null>(null);

  const orderError = orderNumber.trim() ? undefined : "Bitte Auftragsnummer angeben.";
  const plateError = plate.trim() ? undefined : "Bitte Kennzeichen angeben.";

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitted(true);

    if (orderError || plateError) {
      setResult(null);
      setApiError(null);
      return;
    }

    setLoading(true);
    setResult(null);
    setApiError(null);
    try {
      const data = await getOrderStatus({ orderNumber, plate });
      setResult(data);
    } catch (error) {
      if (error instanceof ApiError) {
        setApiError({ message: error.message, code: error.code });
      } else {
        setApiError({ message: "Die API ist nicht erreichbar.", code: "network_error" });
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <section>
      <h1 className="page__title">Status abrufen</h1>
      <p className="page__lead">
        Geben Sie Ihre Auftragsnummer und das Kennzeichen ein, um den Bearbeitungsstand und – sobald
        fertig – Ihre Rechnung zu sehen.
      </p>

      <div style={{ maxWidth: 720 }}>
        <Card>
          <form onSubmit={handleSubmit} noValidate>
            <div className="form-row">
              <FormField
                label="Auftragsnummer"
                htmlFor="order-no"
                error={orderError}
                submitted={submitted}
              >
                <input
                  id="order-no"
                  name="orderNo"
                  className="mono"
                  type="text"
                  autoComplete="off"
                  placeholder="A-2025-0139"
                  value={orderNumber}
                  onChange={(event) => setOrderNumber(event.target.value)}
                />
              </FormField>

              <FormField
                label="Kennzeichen"
                htmlFor="plate"
                error={plateError}
                submitted={submitted}
              >
                <input
                  id="plate"
                  name="plate"
                  className="mono"
                  type="text"
                  maxLength={10}
                  autoComplete="off"
                  placeholder="F-RS 6207"
                  style={{ textTransform: "uppercase" }}
                  value={plate}
                  onChange={(event) => setPlate(event.target.value.toUpperCase())}
                />
              </FormField>
            </div>

            {apiError && (
              <div className="lookup-error" id="lookup-error-banner" role="alert">
                <div className="banner banner--danger">
                  <svg
                    viewBox="0 0 20 20"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    aria-hidden="true"
                  >
                    <path d="M10 6v5M10 14v.5" strokeLinecap="round" />
                    <circle cx="10" cy="10" r="8" />
                  </svg>
                  <div className="banner__body">
                    {apiError.message}
                    <span className="banner__code">Code: {apiError.code}</span>
                  </div>
                </div>
              </div>
            )}

            <div className="page-actions">
              <Button type="submit" loading={loading} className="btn--block-mobile">
                Status abrufen
              </Button>
            </div>
          </form>
        </Card>
      </div>

      {result && (
        <div className="lookup-result">
          <div className="two-col">
            <div className="stack">
              <Card
                title="Status"
                meta={
                  <>
                    <span className="mono">{result.order_number}</span> ·{" "}
                    <span className="mono">{result.vehicle.plate}</span>
                  </>
                }
              >
                <StatusBadge status={result.status} />
              </Card>

              <Card title="Verlauf">
                <StatusTimeline entries={result.history} />
              </Card>
            </div>

            {result.invoice ? (
              <Card>
                <InvoiceView
                  invoice={result.invoice}
                  orderNumber={result.order_number}
                  plate={result.vehicle.plate}
                />
              </Card>
            ) : (
              <Card title="Rechnung">
                <p className="invoice-empty">Noch keine Rechnung vorhanden</p>
              </Card>
            )}
          </div>
        </div>
      )}
    </section>
  );
}

/** DESIGN.md StatusTimeline: a reached dot + status label + timestamp per entry. */
function StatusTimeline({ entries }: { entries: StatusHistoryEntry[] }) {
  if (entries.length === 0) {
    return <p className="invoice-empty">Noch kein Verlauf vorhanden</p>;
  }

  return (
    <ol className="timeline" data-testid="status-timeline">
      {entries.map((entry) => {
        const modifier = entry.status.replace(/_/g, "-");
        return (
          <li className="timeline__item" key={`${entry.status}-${entry.changed_at}`}>
            <span className="timeline__rail">
              <span className={`timeline__dot timeline__dot--${modifier}`} aria-hidden="true" />
              <span className="timeline__connector" aria-hidden="true" />
            </span>
            <span className="timeline__content">
              <span className="timeline__label">{STATUS_LABELS[entry.status]}</span>
              <span className="timeline__time">{formatDateTime(entry.changed_at)}</span>
            </span>
          </li>
        );
      })}
    </ol>
  );
}

export interface InvoiceViewProps {
  invoice: Invoice;
  orderNumber: string;
  plate: string;
}

/**
 * DESIGN.md InvoiceView: read-only — line items (Bezeichnung | Menge |
 * Einzelpreis | Summe), then right-aligned Netto, 'MwSt. 19 %' and Brutto.
 */
function InvoiceView({ invoice, orderNumber, plate }: InvoiceViewProps) {
  return (
    <div>
      <div className="invoice__head">
        <h2 className="invoice__title">Rechnung</h2>
        <div className="invoice__meta">
          <div className="mono">{orderNumber}</div>
          <div className="mono">{normalizePlate(plate)}</div>
        </div>
      </div>

      <div className="table__wrap">
        <table className="line-items">
          <thead>
            <tr>
              <th>Bezeichnung</th>
              <th className="num">Menge</th>
              <th className="num">Einzelpreis</th>
              <th className="num">Summe</th>
            </tr>
          </thead>
          <tbody>
            {invoice.items.map((item) => (
              <tr key={item.id}>
                <td>{item.description}</td>
                <td className="num nums">{formatMenge(item)}</td>
                <td className="num nums">{formatEuro(item.unit_price_cents)}</td>
                <td className="num nums">{formatEuro(item.total_cents)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="totals">
        <div className="totals__row">
          <span>Netto</span>
          <span className="num nums">{formatEuro(invoice.net_cents)}</span>
        </div>
        <div className="totals__row">
          <span>MwSt. 19 %</span>
          <span className="num nums">{formatEuro(invoice.vat_cents)}</span>
        </div>
        <div className="totals__row totals__row--grand">
          <span>Brutto</span>
          <span className="num nums">{formatEuro(invoice.gross_cents)}</span>
        </div>
      </div>
    </div>
  );
}
