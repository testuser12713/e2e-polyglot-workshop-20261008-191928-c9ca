import type { ReactNode } from "react";

export interface CardProps {
  title?: ReactNode;
  meta?: ReactNode;
  children: ReactNode;
  className?: string;
}

/** Design.md Card: surface, radius lg, 1px border, no shadow. */
export default function Card({ title, meta, children, className }: CardProps) {
  const classes = ["card", className].filter(Boolean).join(" ");

  return (
    <section className={classes}>
      {(title || meta) && (
        <header className="card__header">
          {title && <h2 className="card__title">{title}</h2>}
          {meta && <span className="card__meta">{meta}</span>}
        </header>
      )}
      {children}
    </section>
  );
}
