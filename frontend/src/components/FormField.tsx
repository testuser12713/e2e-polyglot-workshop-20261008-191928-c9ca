import { useState, type FocusEvent, type ReactNode } from "react";

export interface FormFieldProps {
  label: string;
  /** id of the control so the label can point at it. */
  htmlFor?: string;
  /** Validation message; shown only once the field was touched or submitted (AC-24). */
  error?: string;
  /** Set by a parent form after a submit attempt. */
  submitted?: boolean;
  /** Quiet helper text shown while there is no error. */
  hint?: string;
  className?: string;
  children: ReactNode;
}

/**
 * Field primitive (Design.md Field) with AC-24 validation timing:
 * an untouched field is neutral — no red border, no message. The first error
 * appears after blur or a submit attempt.
 */
export default function FormField({
  label,
  htmlFor,
  error,
  submitted = false,
  hint,
  className,
  children,
}: FormFieldProps) {
  const [touched, setTouched] = useState(false);
  const showError = Boolean(error) && (touched || submitted);
  const classes = ["field", showError ? "field--error" : null, className].filter(Boolean).join(" ");

  const handleBlur = (event: FocusEvent<HTMLDivElement>) => {
    if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
      setTouched(true);
    }
  };

  return (
    <div className={classes} onBlur={handleBlur}>
      <label className="field__label" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
      {showError ? (
        <span className="field__error" role="alert">
          {error}
        </span>
      ) : (
        hint && <span className="field__hint">{hint}</span>
      )}
    </div>
  );
}
