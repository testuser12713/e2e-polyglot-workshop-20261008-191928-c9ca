# Rechnungsworker

Python-Worker des Werkstatt-Kundenportals. Er liest abgeschlossene
Werkstattaufträge aus einer Valkey-Liste, berechnet daraus die Rechnung
(Arbeitszeit × Stundensatz plus Teile, zuzüglich 19 % Mehrwertsteuer, alle
Beträge in ganzen Cent), speichert sie in PostgreSQL und legt eine
Benachrichtigung im Postausgang ab.

Diese Vorlage liefert das Gerüst: die Verbindung zu PostgreSQL und Valkey
steht, die Rechnungsberechnung, das Laden der Aufträge und das Schreiben von
Rechnung und Benachrichtigung folgen im Ticket "Implement invoice generation
in the worker".

## Voraussetzungen

- Python 3.11 oder neuer
- laufendes PostgreSQL und laufendes Valkey

## Installation

```bash
cd worker
python -m pip install -e .
```

Für die Tests zusätzlich die Dev-Abhängigkeiten:

```bash
python -m pip install -e ".[dev]"
```

## Umgebungsvariablen

| Variable             | Pflicht | Standard              | Beschreibung                                   |
| -------------------- | ------- | --------------------- | ---------------------------------------------- |
| `DATABASE_URL`       | ja      | –                     | PostgreSQL-Verbindungszeichenfolge             |
| `VALKEY_URL`         | ja      | –                     | Valkey-Verbindungszeichenfolge                 |
| `QUEUE_NAME`         | nein    | `workshop-invoices`   | Name der Valkey-Liste mit den Auftragsnachrichten |
| `HOURLY_RATE_CENTS`  | nein    | `8900`                | Stundensatz in ganzen Cent (89,00 €)           |

Die Dev-Werte sind in `RUN.json` deklariert.

## Starten

Ein einzelner Durchlauf, danach beendet sich der Prozess:

```bash
python -m worker --once
```

Dauerbetrieb (pollt die Warteschlange im kurzen Takt und protokolliert jeden
Durchlauf):

```bash
python -m worker
```

## Tests

Die Tests laufen gegen echte Dienste – kein Mocking von Datenbank oder
Warteschlange. Sie erwarten eine erreichbare PostgreSQL- und
Valkey-Instanz:

```bash
cd worker
PYTHONPATH=. python -m pytest
```

## Nachrichtenformat

Die API legt je abgeschlossenem Auftrag ein JSON-Objekt in die Liste
`QUEUE_NAME`:

```json
{"order_id": 42, "order_number": "AU-2026-0042"}
```
