import { useCallback, useEffect, useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { useParams } from "react-router-dom";

import Button from "../../components/Button";
import Card from "../../components/Card";
import FormField from "../../components/FormField";
import StatusBadge, { type OrderStatus } from "../../components/StatusBadge";
import { ApiError } from "../../api/client";
import {
  addItem,
  deleteItem,
  getOrder,
  setOrderStatus,
  updateItem,
  type ItemInput,
  type ItemKind,
  type OrderDetail,
  type OrderItem,
} from "../../api/workshopOrderDetail";

/* -------------------------------------------------------------------------- */
/* Formatting helpers (whole cents, German display format)                     */
/* -------------------------------------------------------------------------- */

/** Formats whole cents as '1.234,56 €' (dot thousands, comma decimals). */
export function formatCents(cents: number): string {
  const rounded = Math.round(cents);
  const sign = rounded < 0 ? "-" : "";
  const abs = Math.abs(rounded);
  const euro = Math.floor(abs / 100)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  const rest = (abs % 100).toString().padStart(2, "0");
  return `${sign}${euro},${rest} €`;
}

/**
 * Parses a euro amount typed by the user ('92', '92,50', '92.50') into whole
 * cents. Returns null when the input is not a non-negative amount.
 */
export function euroToCents(raw: string): number | null {
  const cleaned = raw.trim().replace(/€/g, "").replace(/\s/g, "");
  if (!/^\d+(?:[.,]\d{1,2})?$/.test(cleaned)) {
    return null;
  }
  return Math.round(Number(cleaned.replace(",", ".")) * 100);
}

/** Parses a positive number typed with either ',' or '.' as decimal separator. */
export function parseNumber(raw: string): number | null {
  const cleaned = raw.trim().replace(/\s/g, "");
  if (!/^\d+(?:[.,]\d+)?$/.test(cleaned)) {
    return null;
  }
  const value = Number(cleaned.replace(",", "."));
  return Number.isFinite(value) && value > 0 ? value : null;
}

/** '2,5 Std.' — comma decimals, one decimal place. */
export function formatHours(hours: number): string {
  return `${hours.toFixed(1).replace(".", ",")} Std.`;
}

function formatQuantity(quantity: number): string {
  return quantity.toString().replace(".", ",");
}

function formatMileage(km: number): string {
  return `${Math.round(km)
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ".")} km`;
}

function formatDate(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "–";
  try {
    return new Intl.DateTimeFormat("de-DE", {
      timeZone: "Europe/Berlin",
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
    }).format(date);
  } catch {
    return date.toISOString().slice(0, 10);
  }
}

function formatDateTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "–";
  try {
    return new Intl.DateTimeFormat("de-DE", {
      timeZone: "Europe/Berlin",
      day: "2-digit",
      month: "2-digit",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }).format(date);
  } catch {
    return date.toISOString();
  }
}

/* -------------------------------------------------------------------------- */
/* Status labels / lifecycle                                                    */
/* -------------------------------------------------------------------------- */

const STATUS_LABELS: Record<OrderStatus, string> = {
  requested: "angefragt",
  confirmed: "bestätigt",
  in_progress: "in Arbeit",
  done: "fertig",
  picked_up: "abgeholt",
};

const STATUS_ORDER: OrderStatus[] = ["requested", "confirmed", "in_progress", "done", "picked_up"];

/** Action verb on the button that advances to a status. */
const NEXT_ACTION_LABELS: Record<OrderStatus, string> = {
  requested: "Anfragen",
  confirmed: "Bestätigen",
  in_progress: "In Arbeit setzen",
  done: "Fertig setzen",
  picked_up: "Abholen",
};

/* -------------------------------------------------------------------------- */
/* Layout                                                                      */
/* -------------------------------------------------------------------------- */

const pageHead: CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  alignItems: "center",
  justifyContent: "space-between",
  gap: "var(--space-3)",
  marginBottom: "var(--space-4)",
};

const twoCol: CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "var(--space-4)",
  alignItems: "flex-start",
};

const col: CSSProperties = {
  flex: "1 1 340px",
  minWidth: 0,
  display: "flex",
  flexDirection: "column",
};

const num: CSSProperties = {
  textAlign: "right",
  fontVariantNumeric: "tabular-nums",
};

const rowActions: CSSProperties = {
  display: "inline-flex",
  gap: "var(--space-0)",
};

const iconButton: CSSProperties = {
  width: 44,
  height: 44,
  display: "inline-flex",
  alignItems: "center",
  justifyContent: "center",
  border: "1px solid transparent",
  background: "transparent",
  borderRadius: "var(--radius-md)",
  color: "var(--color-fg-muted)",
  cursor: "pointer",
};

const totalsBlock: CSSProperties = {
  marginTop: "var(--space-3)",
  paddingTop: "var(--space-3)",
  borderTop: "1px solid var(--color-border)",
  display: "flex",
  flexDirection: "column",
  gap: "var(--space-1)",
};

const totalRow: CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  gap: "var(--space-3)",
  fontSize: "var(--size-sm)",
  color: "var(--color-fg-muted)",
};

const totalGrand: CSSProperties = {
  ...totalRow,
  marginTop: "var(--space-2)",
  color: "var(--color-fg)",
  fontSize: "var(--size-lg)",
  fontWeight: 600,
};

const editorBlock: CSSProperties = {
  marginTop: "var(--space-4)",
  paddingTop: "var(--space-3)",
  borderTop: "1px solid var(--color-border)",
};

const formRow: CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "var(--space-3)",
};

const formCol: CSSProperties = {
  flex: "1 1 200px",
  minWidth: 0,
};

const overlay: CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "var(--color-overlay)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  padding: "var(--space-3)",
  zIndex: 50,
};

const dialog: CSSProperties = {
  background: "var(--color-surface)",
  borderRadius: "var(--radius-lg)",
  padding: "var(--space-4)",
  maxWidth: 420,
  width: "100%",
};

const dialogActions: CSSProperties = {
  display: "flex",
  justifyContent: "flex-end",
  gap: "var(--space-2)",
  marginTop: "var(--space-4)",
};

const timeline: CSSProperties = {
  listStyle: "none",
  margin: 0,
  padding: 0,
  display: "flex",
  flexDirection: "column",
};

const timelineItem: CSSProperties = {
  display: "flex",
  gap: "var(--space-2)",
  alignItems: "flex-start",
  minHeight: 32,
};

const timelineContent: CSSProperties = {
  display: "flex",
  justifyContent: "space-between",
  gap: "var(--space-2)",
  flex: 1,
  fontSize: "var(--size-sm)",
  color: "var(--color-fg)",
};

const detailRow: CSSProperties = {
  display: "flex",
  gap: "var(--space-1)",
  fontSize: "var(--size-sm)",
  marginBottom: "var(--space-1)",
};

const detailLabel: CSSProperties = {
  color: "var(--color-fg-muted)",
  minWidth: 116,
};

const plainInput: CSSProperties = {
  minHeight: 44,
  padding: "10px 12px",
  borderRadius: "var(--radius-md)",
  border: "1px solid var(--color-border)",
  background: "var(--color-surface)",
  color: "var(--color-fg)",
  fontFamily: "inherit",
  fontSize: "var(--size-sm)",
  width: "100%",
};

/* -------------------------------------------------------------------------- */
/* Small local building blocks                                                 */
/* -------------------------------------------------------------------------- */

function Banner({ message, code }: { message: string; code?: string }) {
  return (
    <div
      role="alert"
      style={{
        border: "1px solid var(--color-danger)",
        background: "var(--color-danger-soft)",
        color: "var(--color-danger)",
        borderRadius: "var(--radius-md)",
        padding: "var(--space-2) var(--space-3)",
        fontSize: "var(--size-sm)",
        marginTop: "var(--space-2)",
      }}
    >
      <span>{message}</span>
      {code && (
        <span style={{ color: "var(--color-fg-muted)", fontSize: "var(--size-xs)", marginLeft: 8 }}>
          Code: {code}
        </span>
      )}
    </div>
  );
}

function Dialog({
  title,
  body,
  confirmLabel,
  confirmVariant = "primary",
  onCancel,
  onConfirm,
  busy,
}: {
  title: string;
  body: ReactNode;
  confirmLabel: string;
  confirmVariant?: "primary" | "danger";
  onCancel: () => void;
  onConfirm: () => void;
  busy: boolean;
}) {
  return (
    <div
      style={overlay}
      onClick={(event) => {
        if (event.target === event.currentTarget) onCancel();
      }}
    >
      <div role="dialog" aria-modal="true" aria-label={title} style={dialog}>
        <h2 style={{ fontSize: "var(--size-lg)", fontWeight: 600, marginBottom: "var(--space-2)" }}>
          {title}
        </h2>
        <p style={{ margin: 0, color: "var(--color-fg-muted)", fontSize: "var(--size-sm)" }}>{body}</p>
        <div style={dialogActions}>
          <Button variant="secondary" onClick={onCancel} disabled={busy}>
            Abbrechen
          </Button>
          <Button variant={confirmVariant} onClick={onConfirm} loading={busy}>
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/* Page                                                                        */
/* -------------------------------------------------------------------------- */

interface EditorState {
  mode: "add" | "edit";
  itemId?: number;
}

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const orderId = id ?? "";

  const [detail, setDetail] = useState<OrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [editor, setEditor] = useState<EditorState | null>(null);
  const [itemError, setItemError] = useState<{ message: string; code?: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const [deleteTarget, setDeleteTarget] = useState<OrderItem | null>(null);

  const [statusConfirm, setStatusConfirm] = useState(false);
  const [statusError, setStatusError] = useState<{ message: string; code?: string } | null>(null);

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    setLoadError(null);
    try {
      const data = await getOrder(orderId);
      setDetail(data);
    } catch (error) {
      setLoadError(error instanceof ApiError ? error.message : "Der Auftrag konnte nicht geladen werden.");
    } finally {
      setLoading(false);
    }
  }, [orderId]);

  useEffect(() => {
    void load();
  }, [load]);

  const order = detail?.order ?? null;

  const heading = order ? order.order_number : `Auftrag ${orderId}`;

  const applyOrder = useCallback((updated: OrderDetail["order"]) => {
    setDetail((current) => (current ? { ...current, order: updated } : current));
  }, []);

  /* ---- item mutations -------------------------------------------------- */

  const handleSaveItem = useCallback(
    async (input: ItemInput, editingItemId?: number) => {
      setBusy(true);
      setItemError(null);
      try {
        const result =
          editingItemId === undefined
            ? await addItem(orderId, input)
            : await updateItem(orderId, editingItemId, input);
        setDetail((current) => {
          if (!current) return current;
          const items =
            editingItemId === undefined
              ? [...current.items, result.item]
              : current.items.map((item) => (item.id === editingItemId ? result.item : item));
          return { ...current, items, order: result.order };
        });
        setEditor(null);
      } catch (error) {
        setItemError(
          error instanceof ApiError
            ? { message: error.message, code: error.code }
            : { message: "Die Position konnte nicht gespeichert werden." },
        );
      } finally {
        setBusy(false);
      }
    },
    [orderId],
  );

  const handleDeleteItem = useCallback(async () => {
    if (!deleteTarget) return;
    setBusy(true);
    setItemError(null);
    try {
      const result = await deleteItem(orderId, deleteTarget.id);
      setDetail((current) =>
        current
          ? {
              ...current,
              items: current.items.filter((item) => item.id !== deleteTarget.id),
              order: result.order,
            }
          : current,
      );
      setDeleteTarget(null);
    } catch (error) {
      setItemError(
        error instanceof ApiError
          ? { message: error.message, code: error.code }
          : { message: "Die Position konnte nicht gelöscht werden." },
      );
    } finally {
      setBusy(false);
    }
  }, [deleteTarget, orderId]);

  /* ---- status mutation ------------------------------------------------- */

  const handleSetStatus = useCallback(async () => {
    if (!order?.next_status) return;
    const next = order.next_status;
    setBusy(true);
    setStatusError(null);
    try {
      const updated = await setOrderStatus(orderId, next);
      applyOrder(updated);
      setStatusConfirm(false);
    } catch (error) {
      setStatusError(
        error instanceof ApiError
          ? { message: error.message, code: error.code }
          : { message: "Der Status konnte nicht geändert werden." },
      );
      setStatusConfirm(false);
    } finally {
      setBusy(false);
    }
  }, [applyOrder, order, orderId]);

  /* ---- rendering ------------------------------------------------------- */

  if (loading) {
    return (
      <section data-testid="order-detail" aria-busy="true">
        <h1 className="page__title mono">{heading}</h1>
        <p className="page__lead">Auftrag wird geladen …</p>
      </section>
    );
  }

  if (loadError || !detail || !order) {
    return (
      <section data-testid="order-detail">
        <h1 className="page__title mono">{heading}</h1>
        <Banner message={loadError ?? "Der Auftrag konnte nicht geladen werden."} />
        <div style={{ marginTop: "var(--space-3)" }}>
          <Button variant="secondary" onClick={() => void load()}>
            Erneut versuchen
          </Button>
        </div>
      </section>
    );
  }

  const reached = new Set(detail.history.map((entry) => entry.status));
  const historyByStatus = new Map(detail.history.map((entry) => [entry.status, entry.changed_at]));

  const editingItem =
    editor?.mode === "edit" && editor.itemId !== undefined
      ? (detail.items.find((item) => item.id === editor.itemId) ?? null)
      : null;

  return (
    <section data-testid="order-detail">
      <div style={pageHead}>
        <div>
          <h1 className="page__title mono" style={{ fontFamily: "var(--font-mono)" }}>
            {order.order_number}
          </h1>
          <p className="page__lead" style={{ marginBottom: 0 }}>
            <span className="mono">{order.vehicle.plate}</span> · {order.description} ·{" "}
            {order.customer.name}
          </p>
        </div>
        <StatusBadge status={order.status} />
      </div>

      <div style={twoCol}>
        <div style={col}>
          <Card
            title="Positionen"
            className="card--positions"
            meta={
              <Button
                variant="secondary"
                onClick={() => {
                  setItemError(null);
                  setEditor({ mode: "add" });
                }}
              >
                Position hinzufügen
              </Button>
            }
          >
            {detail.items.length === 0 ? (
              <p style={{ color: "var(--color-fg-muted)", textAlign: "center", padding: "var(--space-5)" }}>
                Noch keine Positionen erfasst.
              </p>
            ) : (
              <div style={{ overflowX: "auto" }}>
                <table className="table" style={{ width: "100%" }}>
                  <thead>
                    <tr>
                      <th>Bezeichnung</th>
                      <th className="num">Menge</th>
                      <th className="num">Einzelpreis</th>
                      <th className="num">Summe</th>
                      <th aria-label="Aktionen" />
                    </tr>
                  </thead>
                  <tbody>
                    {detail.items.map((item) => (
                      <tr key={item.id}>
                        <td>{item.description}</td>
                        <td className="num" style={num}>
                          {item.kind === "labor"
                            ? item.hours !== null
                              ? formatHours(item.hours)
                              : "–"
                            : item.quantity !== null
                              ? formatQuantity(item.quantity)
                              : "–"}
                        </td>
                        <td className="num" style={num}>
                          {formatCents(item.unit_price_cents)}
                        </td>
                        <td className="num" style={num}>
                          {formatCents(item.total_cents)}
                        </td>
                        <td>
                          <span style={rowActions}>
                            <button
                              type="button"
                              style={iconButton}
                              aria-label="Position bearbeiten"
                              onClick={() => {
                                setItemError(null);
                                setEditor({ mode: "edit", itemId: item.id });
                              }}
                            >
                              <PencilIcon />
                            </button>
                            <button
                              type="button"
                              style={iconButton}
                              aria-label="Position löschen"
                              onClick={() => setDeleteTarget(item)}
                            >
                              <TrashIcon />
                            </button>
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            <div style={totalsBlock}>
              <div style={totalRow}>
                <span>Netto</span>
                <span style={num}>{formatCents(order.net_cents)}</span>
              </div>
              <div style={totalRow}>
                <span>MwSt. 19 %</span>
                <span style={num}>{formatCents(order.vat_cents)}</span>
              </div>
              <div style={totalGrand}>
                <span>Brutto</span>
                <span style={num}>{formatCents(order.gross_cents)}</span>
              </div>
            </div>

            {editor && (
              <ItemEditor
                key={editingItem ? `edit-${editingItem.id}` : `add-${editor.mode}`}
                item={editingItem}
                busy={busy}
                error={itemError}
                onCancel={() => {
                  setEditor(null);
                  setItemError(null);
                }}
                onSave={handleSaveItem}
              />
            )}

            {!editor && itemError && (
              <Banner message={itemError.message} code={itemError.code} />
            )}
          </Card>
        </div>

        <div style={col}>
          <Card title="Status">
            <div style={{ marginBottom: "var(--space-3)" }}>
              <StatusBadge status={order.status} />
            </div>

            {order.next_status ? (
              <>
                <p style={{ fontSize: "var(--size-sm)", color: "var(--color-fg-muted)", marginTop: 0 }}>
                  Nächster erlaubter Schritt:{" "}
                  <strong style={{ color: "var(--color-fg)" }}>{STATUS_LABELS[order.next_status]}</strong>.
                </p>
                <Button onClick={() => setStatusConfirm(true)}>
                  {NEXT_ACTION_LABELS[order.next_status]}
                </Button>
              </>
            ) : (
              <>
                <p style={{ fontSize: "var(--size-sm)", color: "var(--color-fg-muted)", marginTop: 0 }}>
                  {order.status === "picked_up"
                    ? "Der Auftrag ist abgeschlossen und abgeholt."
                    : "Der Auftrag ist abgeschlossen."}
                </p>
                <Button
                  disabled
                  title="Der Auftrag ist abgeschlossen – kein weiterer Statusschritt möglich."
                >
                  Abgeschlossen
                </Button>
              </>
            )}

            {statusError && <Banner message={statusError.message} code={statusError.code} />}
          </Card>

          <Card title="Verlauf">
            <ol style={timeline}>
              {STATUS_ORDER.map((status, index) => {
                const isReached = reached.has(status);
                const changedAt = historyByStatus.get(status);
                const isLast = index === STATUS_ORDER.length - 1;
                return (
                  <li key={status} style={timelineItem}>
                    <span
                      aria-hidden="true"
                      style={{
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                        alignSelf: "stretch",
                        paddingTop: 6,
                      }}
                    >
                      <span
                        style={{
                          width: 14,
                          height: 14,
                          borderRadius: "var(--radius-pill)",
                          background: isReached ? "var(--color-accent)" : "var(--color-border-strong)",
                          flex: "none",
                        }}
                      />
                      {!isLast && (
                        <span
                          style={{
                            width: 2,
                            flex: 1,
                            minHeight: 18,
                            background: "var(--color-border)",
                          }}
                        />
                      )}
                    </span>
                    <span
                      style={{
                        ...timelineContent,
                        color: isReached ? "var(--color-fg)" : "var(--color-fg-subtle)",
                      }}
                    >
                      <span style={{ fontWeight: isReached ? 500 : 400 }}>{STATUS_LABELS[status]}</span>
                      <span style={{ color: "var(--color-fg-muted)", fontSize: "var(--size-xs)" }}>
                        {changedAt ? formatDateTime(changedAt) : "–"}
                      </span>
                    </span>
                  </li>
                );
              })}
            </ol>
          </Card>

          <Card title="Fahrzeug & Kunde">
            <div style={{ display: "flex", flexDirection: "column", gap: "var(--space-1)" }}>
              <DetailRow label="Kunde">{order.customer.name}</DetailRow>
              <DetailRow label="Telefon">{order.customer.phone || "–"}</DetailRow>
              <DetailRow label="E-Mail">{order.customer.email || "–"}</DetailRow>
              <DetailRow label="Fahrzeug">
                {order.vehicle.brand} {order.vehicle.model}
              </DetailRow>
              <DetailRow label="Kennzeichen">
                <span className="mono">{order.vehicle.plate}</span>
              </DetailRow>
              <DetailRow label="Kilometerstand">
                <span style={num}>{formatMileage(order.vehicle.mileage)}</span>
              </DetailRow>
              <DetailRow label="Wunschtermin">
                <span style={num}>{formatDate(order.preferred_date)}</span>
              </DetailRow>
              <DetailRow label="Anliegen">{order.description || "–"}</DetailRow>
            </div>
          </Card>
        </div>
      </div>

      {deleteTarget && (
        <Dialog
          title="Position löschen"
          body={
            <>
              „{deleteTarget.description}“ wirklich löschen? Die Summen werden danach neu berechnet.
            </>
          }
          confirmLabel="Löschen"
          confirmVariant="danger"
          busy={busy}
          onCancel={() => setDeleteTarget(null)}
          onConfirm={() => void handleDeleteItem()}
        />
      )}

      {statusConfirm && order.next_status && (
        <Dialog
          title="Status ändern"
          body={
            <>
              Status auf „{STATUS_LABELS[order.next_status]}“ setzen?
            </>
          }
          confirmLabel="Bestätigen"
          busy={busy}
          onCancel={() => setStatusConfirm(false)}
          onConfirm={() => void handleSetStatus()}
        />
      )}
    </section>
  );
}

function DetailRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div style={detailRow}>
      <span style={detailLabel}>{label}:</span>
      <span style={{ color: "var(--color-fg)" }}>{children}</span>
    </div>
  );
}

function PencilIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 20 20"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      aria-hidden="true"
    >
      <path d="M4 13.5l-.5 3 3-.5L15 7.5 12.5 5 4 13.5z" strokeLinejoin="round" />
    </svg>
  );
}

function TrashIcon() {
  return (
    <svg
      width="20"
      height="20"
      viewBox="0 0 20 20"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      aria-hidden="true"
    >
      <path d="M4 6h12M8 6V4h4v2M6 6l1 11h6l1-11" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

/* -------------------------------------------------------------------------- */
/* Inline line-item editor                                                     */
/* -------------------------------------------------------------------------- */

interface ItemEditorProps {
  item: OrderItem | null;
  busy: boolean;
  error: { message: string; code?: string } | null;
  onCancel: () => void;
  onSave: (input: ItemInput, editingItemId?: number) => void;
}

interface FieldErrors {
  description?: string;
  quantity?: string;
  price?: string;
}

function ItemEditor({ item, busy, error, onCancel, onSave }: ItemEditorProps) {
  const [kind, setKind] = useState<ItemKind>(item?.kind ?? "part");
  const [description, setDescription] = useState(item?.description ?? "");
  const [hours, setHours] = useState(
    item?.kind === "labor" && item.hours !== null ? formatQuantity(item.hours) : "",
  );
  const [quantity, setQuantity] = useState(
    item?.kind === "part" && item.quantity !== null ? formatQuantity(item.quantity) : "",
  );
  const [price, setPrice] = useState(
    item
      ? `${Math.floor(item.unit_price_cents / 100)},${(item.unit_price_cents % 100)
          .toString()
          .padStart(2, "0")}`
      : "",
  );
  const [submitted, setSubmitted] = useState(false);

  const errors = useMemo<FieldErrors>(() => {
    const next: FieldErrors = {};
    if (!description.trim()) {
      next.description = "Bezeichnung ist erforderlich.";
    }
    if (kind === "labor") {
      if (parseNumber(hours) === null) {
        next.quantity = "Stunden sind erforderlich.";
      }
    } else if (parseNumber(quantity) === null) {
      next.quantity = "Menge ist erforderlich.";
    }
    if (euroToCents(price) === null) {
      next.price = kind === "labor" ? "Stundensatz ist erforderlich." : "Einzelpreis ist erforderlich.";
    }
    return next;
  }, [description, hours, kind, price, quantity]);

  const hasErrors = Object.keys(errors).length > 0;

  const submit = () => {
    setSubmitted(true);
    if (hasErrors) return;

    const unitPriceCents = euroToCents(price);
    if (kind === "labor") {
      const parsedHours = parseNumber(hours);
      if (parsedHours === null || unitPriceCents === null) return;
      onSave(
        {
          kind: "labor",
          description: description.trim(),
          hours: parsedHours,
          quantity: null,
          unit_price_cents: unitPriceCents,
        },
        item?.id,
      );
    } else {
      const parsedQuantity = parseNumber(quantity);
      if (parsedQuantity === null || unitPriceCents === null) return;
      onSave(
        {
          kind: "part",
          description: description.trim(),
          hours: null,
          quantity: parsedQuantity,
          unit_price_cents: unitPriceCents,
        },
        item?.id,
      );
    }
  };

  const quantityLabel = kind === "labor" ? "Stunden" : "Menge";
  const priceLabel = kind === "labor" ? "Stundensatz (€)" : "Einzelpreis (€)";

  return (
    <div style={editorBlock}>
      <div style={formRow}>
        <div style={formCol}>
          <FormField label="Art" htmlFor="pos-kind">
            <select
              id="pos-kind"
              className="field__control"
              style={plainInput}
              value={kind}
              onChange={(event) => setKind(event.target.value as ItemKind)}
            >
              <option value="part">Teil</option>
              <option value="labor">Arbeitszeit</option>
            </select>
          </FormField>
        </div>
        <div style={formCol}>
          <FormField
            label="Bezeichnung"
            htmlFor="pos-description"
            error={errors.description}
            submitted={submitted}
          >
            <input
              id="pos-description"
              className="field__control"
              style={plainInput}
              type="text"
              placeholder="z. B. Ölfilter"
              value={description}
              onChange={(event) => setDescription(event.target.value)}
            />
          </FormField>
        </div>
      </div>

      <div style={formRow}>
        <div style={formCol}>
          <FormField
            label={quantityLabel}
            htmlFor="pos-quantity"
            error={errors.quantity}
            submitted={submitted}
          >
            <input
              id="pos-quantity"
              className="field__control"
              style={{ ...plainInput, fontVariantNumeric: "tabular-nums" }}
              type="text"
              inputMode="decimal"
              placeholder={kind === "labor" ? "4,0" : "1"}
              value={kind === "labor" ? hours : quantity}
              onChange={(event) =>
                kind === "labor" ? setHours(event.target.value) : setQuantity(event.target.value)
              }
            />
          </FormField>
        </div>
        <div style={formCol}>
          <FormField label={priceLabel} htmlFor="pos-price" error={errors.price} submitted={submitted}>
            <input
              id="pos-price"
              className="field__control"
              style={{ ...plainInput, fontVariantNumeric: "tabular-nums" }}
              type="text"
              inputMode="decimal"
              placeholder="45,00"
              value={price}
              onChange={(event) => setPrice(event.target.value)}
            />
          </FormField>
        </div>
      </div>

      {error && <Banner message={error.message} code={error.code} />}

      <div style={{ display: "flex", gap: "var(--space-2)", marginTop: "var(--space-2)" }}>
        <Button onClick={submit} loading={busy}>
          Speichern
        </Button>
        <Button variant="secondary" onClick={onCancel} disabled={busy}>
          Abbrechen
        </Button>
      </div>
    </div>
  );
}
