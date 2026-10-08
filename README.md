# Kfz-Werkstatt-Kundenportal — API

Diese Go-API ist das Rückgrat des Kfz-Werkstatt-Kundenportals. Sie hält Kunden,
Fahrzeuge, Werkstattaufträge samt Statusablauf, Rechnungen und Postausgang in
PostgreSQL vor, veröffentlicht bei Statuswechsel auf `done` eine Nachricht in
einer Valkey-Liste (die ein Python-Worker zur Rechnungserstellung abholt) und
stellt den Kundenbereich (Termin anfragen, Status je Auftragsnummer und
Kennzeichen abrufen) sowie den nach Bearer-Token geschützten Werkstattbereich
bereit. Alle Antworten sind JSON, Beträge sind ganze Cent, Zeitangaben sind
ISO-8601 in UTC, und jede Fehlerantwort hat denselben Körper
`{"error":{"code","message"}}`.

## Tech-Stack

- **api**: Go (net/http aus der Standardbibliothek, pgx/v5) — Port 8080
- **database**: PostgreSQL 18
- **queue**: Valkey 9.1 (Liste `workshop-invoices`)
- **worker**: Python (Rechnungserstellung) — siehe `worker/README.md`
- **web**: Vite + React + TypeScript (Kunden- und Werkstattbereich) — siehe `frontend/README.md`
- **tests**: Go `httptest` gegen eine echte PostgreSQL-Datenbank

## Voraussetzungen

- Go 1.23 oder neuer
- PostgreSQL 18 und Valkey 9.1 (am einfachsten über die mitgelieferte
  `compose.yaml`)

## Datenbanken starten

```bash
docker compose up -d
```

Damit laufen PostgreSQL auf `localhost:5432` (Benutzer und Datenbank `app`, die
Zugangsdaten stehen in `compose.yaml`) und Valkey auf `localhost:6379`.

## Konfiguration

Die API liest ihre Konfiguration aus der Umgebung. Fehlt ein Pflichtwert,
startet sie nicht und nennt die fehlende Variable. In `RUN.json` sind alle
Variablen einer Klasse zugeordnet (`dev`, `generate` oder `external`).

| Variable | Pflicht | Standard | Bedeutung |
| --- | --- | --- | --- |
| `PORT` | nein | `8080` | Port des HTTP-Servers |
| `DATABASE_URL` | ja | — | PostgreSQL-Verbindung, z. B. `postgresql://<user>:<password>@localhost:5432/app` |
| `VALKEY_URL` | ja | — | Valkey-Verbindung, z. B. `redis://localhost:6379/0` |
| `QUEUE_NAME` | nein | `workshop-invoices` | Valkey-Liste für Rechnungsaufträge |
| `HOURLY_RATE_CENTS` | nein | `8900` | Stundensatz in Cent für die Rechnung |
| `AUTH_SECRET` | ja | — | Signatur-Schlüssel der Bearer-Token (wird je Lauf erzeugt) |
| `EMPLOYEE_EMAIL` | ja | — | E-Mail des beim Start angelegten Mitarbeiters |
| `EMPLOYEE_PASSWORD` | ja | — | Klartext-Passwort des Mitarbeiters; gespeichert wird nur der Hash |
| `CORS_ORIGIN` | nein | `http://localhost:5173` | erlaubte Herkunft der Web-App |

## Starten (Entwicklung)

```bash
cd backend
export DATABASE_URL="postgresql://<user>:<password>@localhost:5432/app"
export VALKEY_URL="redis://localhost:6379/0"
export AUTH_SECRET="$(openssl rand -hex 32)"
export EMPLOYEE_EMAIL="meister@example.com"
export EMPLOYEE_PASSWORD="changeme"
go run .
```

Die API legt ihr Schema beim Start automatisch an (Migrationen unter
`backend/migrations/`) und lauscht danach auf `http://localhost:8080`.

Für einen Produktions-Build:

```bash
cd backend
go build -o workshop-api .
```

## Tests

Die Tests laufen mit `httptest` gegen eine echte PostgreSQL-Datenbank
(`DATABASE_URL`), ohne SQLite und ohne Datenbank-Attrappen:

```bash
cd backend
DATABASE_URL="postgresql://<user>:<password>@localhost:5432/app" go test ./...
```

## API-Endpunkte

Geschützte Endpunkte erwarten `Authorization: Bearer <token>`.

Öffentlich:

| Methode | Pfad | Anfrage | Antwort |
| --- | --- | --- | --- |
| GET | `/api/health` | — | `200 {"status":"ok"}` |
| POST | `/api/customers` | `{name,email,phone}` | `201 Customer` |
| POST | `/api/vehicles` | `{plate,brand,model,mileage}` | `201 Vehicle` / `409` Kennzeichen vergeben |
| POST | `/api/appointments` | `{name,email,phone,plate,brand,model,mileage,preferred_date,description}` | `201 {"order_id","order_number"}` |
| GET | `/api/orders/status?order_number=&plate=` | — | `200 {"order_number","status","history":[History],"vehicle":Vehicle,"invoice":Invoice\|null}` / `404` |
| POST | `/api/auth/login` | `{email,password}` | `200 {"token","employee":{id,name,email}}` / `401` |

Werkstatt (Bearer-Token erforderlich):

| Methode | Pfad | Anfrage | Antwort |
| --- | --- | --- | --- |
| GET | `/api/workshop/orders?status=&plate=` | — | `200 {"orders":[Order]}` |
| POST | `/api/workshop/orders/{id}/confirm` | — | `200 Order` |
| POST | `/api/workshop/orders/{id}/status` | `{status}` | `200 Order` / `409` unerlaubter Übergang |
| GET | `/api/workshop/orders/{id}` | — | `200 {"order":Order,"items":[Item],"history":[History]}` |
| POST | `/api/workshop/orders/{id}/items` | `{kind,description,hours,quantity,unit_price_cents}` | `201 {"item":Item,"order":Order}` |
| PUT | `/api/workshop/orders/{id}/items/{item_id}` | `{kind,description,hours,quantity,unit_price_cents}` | `200 {"item":Item,"order":Order}` |
| DELETE | `/api/workshop/orders/{id}/items/{item_id}` | — | `200 {"order":Order}` |
| GET | `/api/workshop/dashboard` | — | `200 {"open_orders","done_today","revenue_month_cents"}` |

Datentypen:

- `Customer = {id,name,email,phone}`
- `Vehicle = {id,plate,brand,model,mileage}`
- `Order = {id,order_number,status,next_status,preferred_date,description,created_at,vehicle,customer,labor_cents,parts_cents,net_cents,vat_cents,gross_cents}`
- `Item = {id,kind,description,hours,quantity,unit_price_cents,total_cents}`
- `History = {status,changed_at}`
- Statuswerte: `requested`, `confirmed`, `in_progress`, `done`, `picked_up`
- `Invoice = {items:[Item],net_cents,vat_cents,gross_cents}`

Jede Fehlerantwort hat den Körper
`{"error":{"code":"...","message":"..."}}` mit passendem HTTP-Status. Ein
unbekannter Pfad antwortet `404` mit genau diesem Körper.

## Projektstruktur

- `backend/` — Go-API (dieser Dienst)
  - `main.go` — Konfiguration, Verbindung, Migrationen, HTTP-Server
  - `internal/config` — Konfiguration aus der Umgebung
  - `internal/db` — pgxpool und Migrations-Runner
  - `internal/httpapi` — Router, Fehlerkörper, Bearer-Middleware
  - `internal/order`, `internal/customer`, `internal/vehicle` — Domänenmodelle und Stores
  - `internal/queue` — Valkey-Publisher
  - `migrations/0001_init.sql` — vollständiges Schema
- `worker/` — Python-Worker zur Rechnungserstellung (siehe `worker/README.md`)
- `frontend/` — Vite/React-Web-App (siehe `frontend/README.md`)

## Funktionen

- Kunden und Fahrzeuge anlegen (Fahrzeugkennzeichen eindeutig)
- Terminwunsch als Auftrag im Status `requested`
- Statusablauf `requested → confirmed → in_progress → done → picked_up`
- Verlauf jedes Statuswechsels mit UTC-Zeitpunkt
- Rechnungserstellung durch den Worker (Arbeitszeit × Stundensatz + Teile, 19 % MwSt.)
- Statusabruf für Kunden inklusive Rechnung
- Geschützter Werkstattbereich: Auftragsliste, Bestätigen, Positionen, Status, Dashboard
- Einheitlicher Fehlerkörper für jede Fehlerantwort
