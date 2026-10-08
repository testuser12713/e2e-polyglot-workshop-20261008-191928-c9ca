# Werkstatt-Portal – Web-App

Vite + React + TypeScript SPA des Kfz-Werkstatt-Kundenportals. Sie liefert den
Kundenbereich (Termin anfragen, Status und Rechnung abrufen) und den nach
Anmeldung geschützten Werkstattbereich (Dashboard, Auftragsliste, Auftragsdetail).
Die Web-App spricht die Go-API über HTTP/JSON an; die Basis-URL kommt aus
`VITE_API_BASE_URL`.

## Technik

- Vite + React 19 + TypeScript
- react-router-dom (Routen siehe unten)
- Vitest + Testing Library (jsdom)

## Installation

```bash
cd frontend
npm ci
```

## Entwicklung starten

```bash
cd frontend
npm run dev
```

Der Vite-Dev-Server läuft standardmäßig auf `http://localhost:5173`.

## Produktions-Build

```bash
cd frontend
npm run build      # tsc --noEmit && vite build  -> dist/
npm run preview    # gebauten Stand lokal ausliefern
```

## Konfiguration

| Variable | Bedeutung |
| --- | --- |
| `VITE_API_BASE_URL` | Basis-URL der API, z. B. `http://localhost:8000`. Ist sie nicht gesetzt (oder noch nicht aufgelöst), nutzt der Client einen relativen Same-Origin-Pfad als Arbeitsdefault. |

In `RUN.json` ist sie als `VITE_API_BASE_URL=${service:api.origin}` deklariert,
damit sie in jeder Umgebung auf die tatsächlich laufende API zeigt.

## Tests

```bash
cd frontend
npm test        # vitest run
npm run build   # tsc --noEmit && vite build
```

## Verwendung

Die App-Shell (Kopfzeile mit Wortmarke, Navigation und Erreichbarkeitsanzeige
der API über `GET /api/health`) umschließt jede Seite.

| Pfad | Seite |
| --- | --- |
| `/` | Termin anfragen (Kundenbereich) |
| `/status` | Status- und Rechnungsabruf (Kundenbereich) |
| `/werkstatt/login` | Anmeldung für Mitarbeiter |
| `/werkstatt` | Dashboard (geschützt) |
| `/werkstatt/auftraege` | Auftragsliste (geschützt) |
| `/werkstatt/auftraege/:id` | Auftragsdetail (geschützt) |

Alle `/werkstatt`-Routen laufen durch den `RequireAuth`-Guard.

## API-Anbindung

`src/api/client.ts` ist der gemeinsame, typisierte Request-Helper: er liest die
Basis-URL aus `VITE_API_BASE_URL`, hängt den Bearer-Token an und wandelt den
einheitlichen Fehlerkörper `{"error":{"code","message"}}` in eine `ApiError` um.
`getHealth()` ruft `GET /api/health` für die Erreichbarkeitsanzeige auf.
