import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";

import { ApiError } from "../../api/client";
import {
  getDashboard,
  getRecentOrders,
  type DashboardOrder,
  type DashboardSummary,
} from "../../api/dashboard";
import Button from "../../components/Button";
import Card from "../../components/Card";
import StatusBadge from "../../components/StatusBadge";
import Table, { type TableColumn } from "../../components/Table";
import "./DashboardPage.css";

const euroFormatter = new Intl.NumberFormat("de-DE", {
  style: "currency",
  currency: "EUR",
});

const dateFormatter = new Intl.DateTimeFormat("de-DE", {
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  timeZone: "Europe/Berlin",
});

const monthFormatter = new Intl.DateTimeFormat("de-DE", {
  month: "long",
  year: "numeric",
  timeZone: "Europe/Berlin",
});

/** Amounts arrive as whole cents and are shown as Euro with German formatting. */
function formatEuro(cents: number): string {
  return euroFormatter.format((Number.isFinite(cents) ? cents : 0) / 100);
}

function formatDate(value: string | null | undefined): string {
  if (!value) return "–";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "–";
  return dateFormatter.format(parsed);
}

type SectionState<T> =
  | { status: "loading" }
  | { status: "ready"; data: T }
  | { status: "error"; message: string; code: string };

function toError(error: unknown): { message: string; code: string } {
  if (error instanceof ApiError) {
    return { message: error.message, code: error.code };
  }
  return { message: "Es ist ein Fehler aufgetreten.", code: "" };
}

function KpiSkeleton() {
  return (
    <div className="dashboard-kpi-grid" aria-hidden="true">
      {[0, 1, 2].map((tile) => (
        <div key={tile} className="card dashboard-kpi">
          <span className="skeleton dashboard-skeleton--label" />
          <span className="skeleton dashboard-skeleton--value" />
        </div>
      ))}
    </div>
  );
}

function OrdersSkeleton() {
  return (
    <div className="dashboard-skeleton-rows" aria-hidden="true">
      {[0, 1, 2].map((row) => (
        <span key={row} className="skeleton dashboard-skeleton--row" />
      ))}
    </div>
  );
}

interface ErrorBannerProps {
  message: string;
  code: string;
  onRetry: () => void;
}

function ErrorBanner({ message, code, onRetry }: ErrorBannerProps) {
  return (
    <div className="dashboard-banner dashboard-banner--danger" role="alert">
      <div className="dashboard-banner__body">
        <p className="dashboard-banner__message">{message}</p>
        {code && <span className="dashboard-banner__code">Code: {code}</span>}
      </div>
      <Button variant="secondary" onClick={onRetry}>
        Erneut versuchen
      </Button>
    </div>
  );
}

export default function DashboardPage() {
  const [summary, setSummary] = useState<SectionState<DashboardSummary>>({ status: "loading" });
  const [orders, setOrders] = useState<SectionState<DashboardOrder[]>>({ status: "loading" });

  const load = useCallback(() => {
    setSummary({ status: "loading" });
    setOrders({ status: "loading" });

    getDashboard()
      .then((data) => setSummary({ status: "ready", data }))
      .catch((error: unknown) => setSummary({ status: "error", ...toError(error) }));

    getRecentOrders()
      .then((data) => setOrders({ status: "ready", data }))
      .catch((error: unknown) => setOrders({ status: "error", ...toError(error) }));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const columns: TableColumn<DashboardOrder>[] = [
    {
      key: "order_number",
      header: "Auftragsnummer",
      render: (order) => (
        <Link className="mono nums" to={`/werkstatt/auftraege/${order.id}`}>
          {order.order_number}
        </Link>
      ),
    },
    {
      key: "plate",
      header: "Kennzeichen",
      render: (order) => <span className="mono">{order.vehicle.plate}</span>,
    },
    { key: "customer", header: "Kunde", render: (order) => order.customer.name },
    {
      key: "status",
      header: "Status",
      render: (order) => <StatusBadge status={order.status} />,
    },
    {
      key: "preferred_date",
      header: "Wunschtermin",
      align: "right",
      render: (order) => <span className="nums">{formatDate(order.preferred_date)}</span>,
    },
  ];

  return (
    <section className="dashboard">
      <h1 className="page__title">Dashboard</h1>
      <p className="page__lead">Überblick über die laufenden Aufträge und den Umsatz.</p>

      {summary.status === "loading" && <KpiSkeleton />}

      {summary.status === "error" && (
        <ErrorBanner message={summary.message} code={summary.code} onRetry={load} />
      )}

      {summary.status === "ready" && (
        <div className="dashboard-kpi-grid">
          <Card className="dashboard-kpi">
            <span className="dashboard-kpi__label">Offene Aufträge</span>
            <span className="dashboard-kpi__value">{summary.data.open_orders}</span>
            <span className="dashboard-kpi__delta">angefragt, bestätigt und in Arbeit</span>
          </Card>
          <Card className="dashboard-kpi">
            <span className="dashboard-kpi__label">Heute fertig</span>
            <span className="dashboard-kpi__value">{summary.data.done_today}</span>
            <span className="dashboard-kpi__delta">Stand {formatDate(new Date().toISOString())}</span>
          </Card>
          <Card className="dashboard-kpi">
            <span className="dashboard-kpi__label">Umsatz laufender Monat</span>
            <span className="dashboard-kpi__value">
              {formatEuro(summary.data.revenue_month_cents)}
            </span>
            <span className="dashboard-kpi__delta">{monthFormatter.format(new Date())}</span>
          </Card>
        </div>
      )}

      <Card
        title="Neueste Aufträge"
        meta={<Link to="/werkstatt/auftraege">Alle Aufträge</Link>}
        className="dashboard-recent"
      >
        {orders.status === "loading" && <OrdersSkeleton />}

        {orders.status === "error" && (
          <ErrorBanner message={orders.message} code={orders.code} onRetry={load} />
        )}

        {orders.status === "ready" &&
          (orders.data.length === 0 ? (
            <p className="dashboard-empty">Keine Aufträge vorhanden.</p>
          ) : (
            <>
              <div className="dashboard-orders-table">
                <Table
                  columns={columns}
                  rows={orders.data}
                  rowKey={(order) => String(order.id)}
                  caption="Neueste Aufträge"
                />
              </div>

              <ul className="dashboard-order-cards">
                {orders.data.map((order) => (
                  <li key={order.id} className="dashboard-order-card">
                    <div className="dashboard-order-card__line">
                      <Link
                        className="dashboard-order-card__value mono nums"
                        to={`/werkstatt/auftraege/${order.id}`}
                      >
                        {order.order_number}
                      </Link>
                      <StatusBadge status={order.status} />
                    </div>
                    <div className="dashboard-order-card__line">
                      <span className="dashboard-order-card__label">Kennzeichen</span>
                      <span className="dashboard-order-card__value mono">{order.vehicle.plate}</span>
                    </div>
                    <div className="dashboard-order-card__line">
                      <span className="dashboard-order-card__label">Kunde</span>
                      <span className="dashboard-order-card__value">{order.customer.name}</span>
                    </div>
                    <div className="dashboard-order-card__line">
                      <span className="dashboard-order-card__label">Wunschtermin</span>
                      <span className="dashboard-order-card__value nums">
                        {formatDate(order.preferred_date)}
                      </span>
                    </div>
                  </li>
                ))}
              </ul>
            </>
          ))}
      </Card>
    </section>
  );
}
