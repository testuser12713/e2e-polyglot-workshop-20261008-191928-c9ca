import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";

import { ApiError } from "../../api/client";
import {
  confirmWorkshopOrder,
  listWorkshopOrders,
  type WorkshopOrder,
} from "../../api/workshopOrders";
import Button from "../../components/Button";
import StatusBadge, { type OrderStatus } from "../../components/StatusBadge";
import "./OrderListPage.css";

type StatusFilter = OrderStatus | "all";

interface StatusFilterOption {
  value: StatusFilter;
  label: string;
}

/** Design.md: 'Alle' first, then the five statuses with their fixed labels. */
const STATUS_FILTERS: StatusFilterOption[] = [
  { value: "all", label: "Alle" },
  { value: "requested", label: "angefragt" },
  { value: "confirmed", label: "bestätigt" },
  { value: "in_progress", label: "in Arbeit" },
  { value: "done", label: "fertig" },
  { value: "picked_up", label: "abgeholt" },
];

const SEARCH_DEBOUNCE_MS = 250;
const DATE_PATTERN = /^(\d{4})-(\d{2})-(\d{2})/;

interface ErrorInfo {
  message: string;
  code?: string;
}

function describeError(error: unknown): ErrorInfo {
  if (error instanceof ApiError) {
    return { message: error.message, code: error.code };
  }
  if (error instanceof Error) {
    return { message: error.message };
  }
  return { message: "Es ist ein Fehler aufgetreten." };
}

/** Design.md: dates are displayed as 'DD.MM.YYYY', never a raw ISO string. */
function formatDate(value: string | null): string {
  if (!value) {
    return "–";
  }
  const match = DATE_PATTERN.exec(value);
  if (!match) {
    return value;
  }
  const [, year, month, day] = match;
  return `${day}.${month}.${year}`;
}

function statusLabel(status: StatusFilter): string {
  return STATUS_FILTERS.find((option) => option.value === status)?.label ?? status;
}

/** A disabled confirm action still explains why it cannot run (AC-23). */
function confirmExplanation(status: OrderStatus): string | undefined {
  if (status === "requested") {
    return undefined;
  }
  if (status === "confirmed") {
    return "Auftrag ist bereits bestätigt.";
  }
  return "Nur angefragte Aufträge können bestätigt werden.";
}

export default function OrderListPage() {
  const navigate = useNavigate();

  const [status, setStatus] = useState<StatusFilter>("all");
  const [plateInput, setPlateInput] = useState("");
  const [plate, setPlate] = useState("");
  const [orders, setOrders] = useState<WorkshopOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<ErrorInfo | null>(null);
  const [reloadKey, setReloadKey] = useState(0);
  const [confirmingId, setConfirmingId] = useState<number | null>(null);
  const [actionError, setActionError] = useState<ErrorInfo | null>(null);

  // Debounce the plate search so typing does not fire a request per keystroke.
  useEffect(() => {
    const handle = window.setTimeout(() => {
      setPlate(plateInput.trim());
    }, SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(handle);
  }, [plateInput]);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setLoadError(null);

    listWorkshopOrders({ status, plate })
      .then((data) => {
        if (active) {
          setOrders(data);
        }
      })
      .catch((error: unknown) => {
        if (active) {
          setLoadError(describeError(error));
          setOrders([]);
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, [status, plate, reloadKey]);

  const resetFilters = (): void => {
    setStatus("all");
    setPlateInput("");
    setPlate("");
  };

  const clearPlate = (): void => {
    setPlateInput("");
    setPlate("");
  };

  const openOrder = (id: number): void => {
    navigate(`/werkstatt/auftraege/${id}`);
  };

  const handleConfirm = async (order: WorkshopOrder): Promise<void> => {
    if (order.status !== "requested" || confirmingId !== null) {
      return;
    }
    setConfirmingId(order.id);
    setActionError(null);
    try {
      const updated = await confirmWorkshopOrder(order.id);
      setOrders((previous) =>
        previous.map((entry) => (entry.id === updated.id ? updated : entry)),
      );
    } catch (error: unknown) {
      setActionError(describeError(error));
    } finally {
      setConfirmingId(null);
    }
  };

  const renderActions = (order: WorkshopOrder) => {
    const explanation = confirmExplanation(order.status);
    const confirmable = order.status === "requested";

    return (
      <div className="order-actions" onClick={(event) => event.stopPropagation()}>
        <Button
          variant="secondary"
          data-testid={`confirm-${order.id}`}
          disabled={!confirmable}
          loading={confirmingId === order.id}
          title={explanation}
          aria-disabled={!confirmable}
          onClick={() => void handleConfirm(order)}
        >
          Bestätigen
        </Button>
        <Button variant="ghost" onClick={() => openOrder(order.id)}>
          Details
        </Button>
      </div>
    );
  };

  const hasStatusFilter = status !== "all";
  const hasPlateFilter = plate.length > 0;

  return (
    <section className="order-list">
      <div className="page-head">
        <h1 className="page__title">Aufträge</h1>
        <p className="page__lead">Alle Werkstattaufträge mit Statusfilter und Kennzeichensuche.</p>
      </div>

      <div className="filterbar">
        <div className="filterbar__chips" role="group" aria-label="Nach Status filtern">
          {STATUS_FILTERS.map((option) => (
            <button
              key={option.value}
              type="button"
              className={status === option.value ? "chip is-active" : "chip"}
              aria-pressed={status === option.value}
              onClick={() => setStatus(option.value)}
            >
              {option.label}
            </button>
          ))}
        </div>

        <div className="filterbar__search">
          <svg
            className="search-icon"
            viewBox="0 0 20 20"
            fill="none"
            stroke="currentColor"
            strokeWidth="1.5"
            aria-hidden="true"
          >
            <circle cx="9" cy="9" r="6" />
            <path d="M14 14l4 4" strokeLinecap="round" />
          </svg>
          <input
            className="field__control plate"
            type="text"
            placeholder="Kennzeichen suchen…"
            aria-label="Kennzeichen suchen"
            autoComplete="off"
            value={plateInput}
            onChange={(event) => setPlateInput(event.target.value)}
          />
        </div>

        <div className="active-filters">
          {hasStatusFilter && (
            <span className="active-filter-chip">
              Status: {statusLabel(status)}
              <button
                type="button"
                className="icon-btn"
                aria-label={`Statusfilter ${statusLabel(status)} entfernen`}
                onClick={() => setStatus("all")}
              >
                ×
              </button>
            </span>
          )}
          {hasPlateFilter && (
            <span className="active-filter-chip">
              Kennzeichen: {plate.toUpperCase()}
              <button
                type="button"
                className="icon-btn"
                aria-label="Kennzeichenfilter entfernen"
                onClick={clearPlate}
              >
                ×
              </button>
            </span>
          )}
          <Button variant="ghost" onClick={resetFilters}>
            Filter zurücksetzen
          </Button>
        </div>
      </div>

      {actionError && (
        <div className="alert alert--danger" role="alert">
          <p>{actionError.message}</p>
          {actionError.code && <p className="alert__code">Code: {actionError.code}</p>}
        </div>
      )}

      {loadError ? (
        <div className="alert alert--danger" role="alert">
          <p>{loadError.message}</p>
          {loadError.code && <p className="alert__code">Code: {loadError.code}</p>}
          <Button variant="secondary" onClick={() => setReloadKey((key) => key + 1)}>
            Erneut versuchen
          </Button>
        </div>
      ) : loading ? (
        <p className="order-list__loading" role="status">
          Aufträge werden geladen …
        </p>
      ) : orders.length === 0 ? (
        <div className="empty-state" data-testid="order-empty">
          <p>Keine Aufträge für diesen Filter</p>
          <Button variant="primary" onClick={resetFilters}>
            Filter zurücksetzen
          </Button>
        </div>
      ) : (
        <>
          <div className="order-list__table" data-testid="order-list-table">
            <table className="table">
              <thead>
                <tr>
                  <th>Auftragsnummer</th>
                  <th>Kennzeichen</th>
                  <th>Kunde</th>
                  <th>Status</th>
                  <th className="num">Wunschtermin</th>
                  <th>Aktionen</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((order) => (
                  <tr
                    key={order.id}
                    className="order-row"
                    onClick={() => openOrder(order.id)}
                  >
                    <td>
                      <span className="mono">{order.order_number}</span>
                    </td>
                    <td>
                      <span className="plate">{order.vehicle.plate}</span>
                    </td>
                    <td>{order.customer.name}</td>
                    <td>
                      <StatusBadge status={order.status} />
                    </td>
                    <td className="num">
                      <span className="tabular">{formatDate(order.preferred_date)}</span>
                    </td>
                    <td>{renderActions(order)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="order-cards" data-testid="order-list-cards">
            {orders.map((order) => (
              <div
                key={order.id}
                className="order-card order-row"
                onClick={() => openOrder(order.id)}
              >
                <div className="order-card__line">
                  <span className="order-card__value mono">{order.order_number}</span>
                  <StatusBadge status={order.status} />
                </div>
                <div className="order-card__line">
                  <span className="order-card__value plate">{order.vehicle.plate}</span>
                  <span className="order-card__value">{order.customer.name}</span>
                </div>
                <div className="order-card__line">
                  <span className="order-card__label">Wunschtermin</span>
                  <span className="order-card__value tabular">{formatDate(order.preferred_date)}</span>
                </div>
                <div className="order-card__actions">{renderActions(order)}</div>
              </div>
            ))}
          </div>
        </>
      )}
    </section>
  );
}
