import { useState, type CSSProperties, type ChangeEvent, type FormEvent } from "react";
import { Link } from "react-router-dom";

import { createAppointment, type AppointmentRequest } from "../../api/appointments";
import { ApiError } from "../../api/client";
import Button from "../../components/Button";
import FormField from "../../components/FormField";

interface FormValues {
  plate: string;
  brand: string;
  model: string;
  mileage: string;
  name: string;
  email: string;
  phone: string;
  preferred_date: string;
  description: string;
}

const EMPTY_FORM: FormValues = {
  plate: "",
  brand: "",
  model: "",
  mileage: "",
  name: "",
  email: "",
  phone: "",
  preferred_date: "",
  description: "",
};

const EMAIL_PATTERN = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;

/** Validation messages keyed by field; an empty map means the form is valid. */
function validate(values: FormValues): Partial<Record<keyof FormValues, string>> {
  const errors: Partial<Record<keyof FormValues, string>> = {};

  if (!values.plate.trim()) {
    errors.plate = "Bitte geben Sie das Kennzeichen an.";
  }
  if (!values.mileage.trim()) {
    errors.mileage = "Bitte geben Sie den Kilometerstand an.";
  } else if (!/^\d+$/.test(values.mileage.trim())) {
    errors.mileage = "Bitte geben Sie den Kilometerstand als ganze Zahl an.";
  }
  if (!values.name.trim()) {
    errors.name = "Bitte geben Sie Ihren Namen an.";
  }
  if (!EMAIL_PATTERN.test(values.email.trim())) {
    errors.email = "Bitte geben Sie eine gültige E-Mail-Adresse an.";
  }
  if (!values.description.trim()) {
    errors.description = "Bitte beschreiben Sie kurz Ihr Anliegen.";
  }

  return errors;
}

const rowStyle: CSSProperties = {
  display: "flex",
  flexWrap: "wrap",
  gap: "var(--space-3)",
};

const rowItemStyle: CSSProperties = { flex: "1 1 220px", minWidth: 0 };

const sectionTitleStyle: CSSProperties = { margin: "var(--space-4) 0 var(--space-0)" };

const alertStyle: CSSProperties = {
  border: "1px solid var(--color-danger)",
  background: "var(--color-danger-soft)",
  color: "var(--color-fg)",
  borderRadius: "var(--radius-md)",
  padding: "var(--space-2) var(--space-3)",
  fontSize: "var(--size-sm)",
  marginBottom: "var(--space-3)",
};

const successStyle: CSSProperties = {
  display: "flex",
  gap: "var(--space-2)",
  alignItems: "flex-start",
  border: "1px solid var(--color-success)",
  background: "var(--color-status-done-soft)",
  color: "var(--color-fg)",
  borderRadius: "var(--radius-md)",
  padding: "var(--space-2) var(--space-3)",
  fontSize: "var(--size-sm)",
};

/**
 * Customer appointment request form (Design.md Field / Button, mockup
 * termin-anfragen.html). AC-24: validation feedback appears only after a field
 * was touched or the form was submitted, so an untouched form starts neutral.
 */
export default function AppointmentPage() {
  const [values, setValues] = useState<FormValues>(EMPTY_FORM);
  const [submitted, setSubmitted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [orderNumber, setOrderNumber] = useState<string | null>(null);

  const errors = validate(values);

  const update =
    (field: keyof FormValues) =>
    (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>): void => {
      const { value } = event.target;
      setValues((current) => ({ ...current, [field]: value }));
    };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>): Promise<void> => {
    event.preventDefault();
    setSubmitted(true);
    setErrorMessage(null);

    if (Object.keys(errors).length > 0) {
      return;
    }

    const payload: AppointmentRequest = {
      plate: values.plate.trim().toUpperCase(),
      brand: values.brand.trim(),
      model: values.model.trim(),
      mileage: Number(values.mileage.trim()),
      name: values.name.trim(),
      email: values.email.trim(),
      phone: values.phone.trim(),
      preferred_date: values.preferred_date,
      description: values.description.trim(),
    };

    setSubmitting(true);
    try {
      const response = await createAppointment(payload);
      setOrderNumber(response.order_number);
    } catch (error) {
      setErrorMessage(
        error instanceof ApiError ? error.message : "Es ist ein Fehler aufgetreten.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section>
      <h1 className="page__title">Termin anfragen</h1>
      <p className="page__lead">
        Sagen Sie uns, was an Ihrem Fahrzeug zu tun ist — wir melden uns mit einem Terminvorschlag.
      </p>

      <div className="card" style={{ maxWidth: "720px" }}>
        {orderNumber ? (
          <div data-testid="success-banner" role="status" aria-live="polite">
            <div style={successStyle}>
              <div>
                <p style={{ margin: 0 }}>
                  Ihre Terminanfrage ist eingegangen. Wir prüfen sie und melden uns mit einem
                  Terminvorschlag.
                </p>
                <p style={{ margin: "var(--space-1) 0 0" }}>
                  Ihre Auftragsnummer:{" "}
                  <span className="mono" style={{ fontWeight: "var(--weight-heading)" }}>
                    {orderNumber}
                  </span>
                </p>
              </div>
            </div>
            <p className="field__hint" style={{ display: "block", marginTop: "var(--space-3)" }}>
              Notieren Sie sich die Auftragsnummer zusammen mit dem Kennzeichen — damit können Sie
              unter <Link to="/status">Status abrufen</Link> jederzeit den Bearbeitungsstand
              einsehen.
            </p>
          </div>
        ) : (
          <form onSubmit={handleSubmit} noValidate data-testid="appointment-form">
            {errorMessage && (
              <div role="alert" style={alertStyle}>
                {errorMessage}
              </div>
            )}

            <h2 className="card__title" style={sectionTitleStyle}>
              Fahrzeug
            </h2>

            <FormField
              label="Kennzeichen"
              htmlFor="plate"
              error={errors.plate}
              submitted={submitted}
            >
              <input
                id="plate"
                name="plate"
                className="field__control mono"
                type="text"
                maxLength={10}
                autoComplete="off"
                placeholder="M-KF 4711"
                style={{ textTransform: "uppercase" }}
                value={values.plate}
                onChange={update("plate")}
              />
            </FormField>

            <div style={rowStyle}>
              <div style={rowItemStyle}>
                <FormField label="Marke" htmlFor="make">
                  <input
                    id="make"
                    name="brand"
                    className="field__control"
                    type="text"
                    placeholder="z. B. BMW"
                    value={values.brand}
                    onChange={update("brand")}
                  />
                </FormField>
              </div>
              <div style={rowItemStyle}>
                <FormField label="Modell" htmlFor="model">
                  <input
                    id="model"
                    name="model"
                    className="field__control"
                    type="text"
                    placeholder="z. B. 320i"
                    value={values.model}
                    onChange={update("model")}
                  />
                </FormField>
              </div>
            </div>

            <FormField
              label="Kilometerstand"
              htmlFor="mileage"
              error={errors.mileage}
              submitted={submitted}
              hint="Ganze Kilometer, z. B. 128450."
            >
              <input
                id="mileage"
                name="mileage"
                className="field__control nums"
                type="text"
                inputMode="numeric"
                placeholder="128450"
                value={values.mileage}
                onChange={update("mileage")}
              />
            </FormField>

            <h2 className="card__title" style={sectionTitleStyle}>
              Ihr Anliegen
            </h2>

            <div style={rowStyle}>
              <div style={rowItemStyle}>
                <FormField
                  label="Name"
                  htmlFor="name"
                  error={errors.name}
                  submitted={submitted}
                >
                  <input
                    id="name"
                    name="name"
                    className="field__control"
                    type="text"
                    autoComplete="name"
                    placeholder="Vor- und Nachname"
                    value={values.name}
                    onChange={update("name")}
                  />
                </FormField>
              </div>
              <div style={rowItemStyle}>
                <FormField
                  label="E-Mail"
                  htmlFor="email"
                  error={errors.email}
                  submitted={submitted}
                >
                  <input
                    id="email"
                    name="email"
                    className="field__control"
                    type="email"
                    autoComplete="email"
                    placeholder="name@beispiel.de"
                    value={values.email}
                    onChange={update("email")}
                  />
                </FormField>
              </div>
            </div>

            <FormField
              label="Telefon"
              htmlFor="phone"
              hint="Für die Terminabstimmung."
            >
              <input
                id="phone"
                name="phone"
                className="field__control"
                type="tel"
                autoComplete="tel"
                placeholder="z. B. 0171 2345678"
                value={values.phone}
                onChange={update("phone")}
              />
            </FormField>

            <FormField label="Wunschtermin" htmlFor="preferred_date">
              <input
                id="preferred_date"
                name="preferred_date"
                className="field__control nums"
                type="date"
                value={values.preferred_date}
                onChange={update("preferred_date")}
              />
            </FormField>

            <FormField
              label="Problembeschreibung"
              htmlFor="description"
              error={errors.description}
              submitted={submitted}
            >
              <textarea
                id="description"
                name="description"
                className="field__control"
                placeholder="Beschreiben Sie kurz, was zu tun ist — z. B. 'Inspektion fällig, Bremsen quietschen beim Bremsen'"
                value={values.description}
                onChange={update("description")}
              ></textarea>
            </FormField>

            <div style={{ marginTop: "var(--space-1)" }}>
              <Button type="submit" loading={submitting}>
                Termin anfragen
              </Button>
            </div>
          </form>
        )}
      </div>
    </section>
  );
}
