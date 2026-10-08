export type OrderStatus =
  | "requested"
  | "confirmed"
  | "in_progress"
  | "done"
  | "picked_up";

/** German labels are fixed by Design.md. */
const STATUS_LABELS: Record<OrderStatus, string> = {
  requested: "angefragt",
  confirmed: "bestätigt",
  in_progress: "in Arbeit",
  done: "fertig",
  picked_up: "abgeholt",
};

export interface StatusBadgeProps {
  status: OrderStatus;
  /** Optional 6px dot left of the label (Design.md). */
  withDot?: boolean;
}

/** Status is never color-only: the German label text is always present. */
export default function StatusBadge({ status, withDot = true }: StatusBadgeProps) {
  const label = STATUS_LABELS[status];
  const modifier = status.replace(/_/g, "-");

  return (
    <span className={`status-badge status-badge--${modifier}`}>
      {withDot && <span className="status-badge__dot" aria-hidden="true" />}
      {label}
    </span>
  );
}
