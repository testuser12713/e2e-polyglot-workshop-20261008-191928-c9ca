# Design — Project Identity

> This document is project-long-lived. Tokens are not changed without
> the Architect's approval. Developers MUST use these tokens
> instead of improvising their own colors/spacings.

## Style Direction

Calm, light industrial-professional portal (Linear/Stripe restraint): warm neutral surfaces, one petrol-blue accent, and color used only to encode the five order statuses — dense but quiet, fully legible on phone width.

## Colors

- `--color-bg`: **#F6F7F9**
- `--color-surface`: **#FFFFFF**
- `--color-surface-sunken`: **#EEF1F4**
- `--color-fg`: **#16202B**
- `--color-fg-muted`: **#5A6675**
- `--color-fg-subtle`: **#8A95A3**
- `--color-border`: **#DFE4EA**
- `--color-border-strong`: **#C3CBD5**
- `--color-accent`: **#0F5A8A**
- `--color-accent-hover`: **#0C4A73**
- `--color-accent-active`: **#093A5A**
- `--color-accent-soft`: **#E7F0F7**
- `--color-accent-fg`: **#FFFFFF**
- `--color-focus-ring`: **rgba(15,90,138,0.35)**
- `--color-danger`: **#B3261E**
- `--color-danger-soft`: **#FBECEA**
- `--color-success`: **#2F855A**
- `--color-warning`: **#B7791F**
- `--color-status-requested`: **#B7791F**
- `--color-status-requested-soft`: **#FDF6E7**
- `--color-status-confirmed`: **#1D6FA5**
- `--color-status-confirmed-soft`: **#E8F2F9**
- `--color-status-in-progress`: **#6B46C1**
- `--color-status-in-progress-soft`: **#F0EBFB**
- `--color-status-done`: **#2F855A**
- `--color-status-done-soft`: **#E7F5EC**
- `--color-status-picked-up`: **#4A5568**
- `--color-status-picked-up-soft`: **#EDF0F3**
- `--color-overlay`: **rgba(22,32,43,0.45)**

## Typography

- `font_family`: 'Inter', 'Segoe UI', system-ui, -apple-system, 'Helvetica Neue', Arial, sans-serif
- `font_mono`: 'JetBrains Mono', 'SFMono-Regular', Consolas, 'Liberation Mono', monospace
- `heading_weight`: 600
- `body_weight`: 400
- `weight_medium`: 500
- `size_xs`: 12px
- `size_sm`: 14px
- `size_base`: 16px
- `size_lg`: 18px
- `size_xl`: 24px
- `size_2xl`: 32px
- `line_height_tight`: 1.25
- `line_height_body`: 1.55
- `numeric_features`: font-variant-numeric: tabular-nums for every amount, kilometre reading, hour count and KPI; mono font for order number (Auftragsnummer) and licence plate (Kennzeichen)

## Spacing Scale

- `--space-0`: 4px
- `--space-1`: 8px
- `--space-2`: 12px
- `--space-3`: 16px
- `--space-4`: 24px
- `--space-5`: 32px
- `--space-6`: 48px

## Border-Radii

- `--radius-sm`: 4px
- `--radius-md`: 8px
- `--radius-lg`: 12px
- `--radius-pill`: 999px

## Components

### Button

min-height 44px (mobile tap target), padding 12px 20px, radius md, font 14px/500, gap 8px to icon, transition 120ms ease-out. Variants: primary (bg=accent, fg=accent-fg; hover bg=accent-hover; active bg=accent-active + translateY(1px); disabled bg=surface-sunken fg=fg-subtle border=border, cursor not-allowed). secondary (bg=surface, fg=fg, 1px border=border-strong; hover bg=surface-sunken; active bg=border; disabled as above). ghost (transparent bg, fg=accent; hover bg=accent-soft; no border). danger (bg=surface fg=danger 1px border=danger; hover bg=danger-soft). Focus: 2px accent outline + 3px focus-ring, never removed. Loading state: label stays, spinner replaces icon, button disabled -> never silently does nothing (AC-23). Disabled buttons that need an explanation carry a title/help text, e.g. 'Erst bestätigen, dann in Arbeit setzen'. German labels: 'Termin anfragen', 'Speichern', 'Bestätigen', 'Abmelden'.

### IconButton

44x44px hit area, radius md, 20px icon, fg=fg-muted; hover bg=surface-sunken fg=fg; active bg=border; disabled fg=fg-subtle. Always with accessible label (aria-label), e.g. row actions 'Position bearbeiten', 'Position löschen'.

### Field (Input/Textarea/Select)

Label above, 13px/500 fg-muted, 6px gap. Control 44px min-height (textarea min 96px), padding 10px 12px, radius md, bg=surface, 1px border=border; hover border=border-strong; focus border=accent + 3px focus-ring; disabled bg=surface-sunken fg=fg-subtle. Placeholder fg-subtle. Helper text 12px fg-muted under control. Error state: border=danger, message 12px danger below, icon warning. Validation timing (AC-24): untouched fields are neutral — no message, no red; the first error appears after blur or after submit attempt, then live while typing. Inputs for money accept Euro with comma (e.g. '45,00'), kilometres integer, and are stored as integer cents/kilometres.

### Card

bg=surface, radius lg, 1px border=border, no shadow (optional shadow 0 1px 2px rgba(22,32,43,0.04) for floating layers only), padding 16px mobile / 24px desktop. Card header: title 16px/600 fg + optional meta 13px fg-muted right-aligned; 16px gap to body. Section spacing between cards 16px mobile, 24px desktop.

### StatusBadge

pill, padding 4px 10px, 12px/500, 1px border in status color at 25% alpha, bg=status-*-soft, fg=status-*. German labels fixed: 'angefragt' (requested), 'bestätigt' (confirmed), 'in Arbeit' (in_progress), 'fertig' (done), 'abgeholt' (picked_up). Never color-only: label text always present (also the status-filter chips and the status timeline). Optional 6px dot left of the label.

### AppShell / Navigation

Sticky top bar 56px mobile / 64px desktop, bg=surface, 1px bottom border=border, content max-width 1120px centered with 16px/24px padding. Left: wordmark 'Werkstatt-Portal' 16px/600. Right: customer area shows only quiet links; workshop area shows the logged-in email (13px fg-muted) and a 'Abmelden' ghost button. Mobile: nav collapses into a menu icon (44px) opening a full-width panel; no horizontal scroll.

### OrderListRow / Table

Desktop: table with header 12px/600 uppercase fg-muted, rows 56-64px, 1px divider=border, hover bg=surface-sunken. Columns: Auftragsnummer (mono), Kennzeichen (mono uppercase), Kunde, Status (StatusBadge), Wunschtermin, Aktionen. Phone (<640px): each row becomes a stacked card — line 1 Auftragsnummer + StatusBadge, line 2 Kennzeichen + Kunde, line 3 Wunschtermin, actions full-width below. Numbers right-aligned with tabular-nums. Empty state: centered fg-muted text 'Keine Aufträge für diesen Filter' plus primary action 'Filter zurücksetzen'.

### FilterBar

Status filter as pill chips (min-height 44px, 12px/500, border=border-strong, active chip bg=accent-soft border=accent fg=accent) with 'Alle' first, plus a licence-plate search input with magnifier icon. Debounce 250ms, search is case-insensitive substring. Active filters shown as removable chips, 'Filter zurücksetzen' as ghost button. Sticky at the top of the list on mobile.

### StatusTimeline

Vertical list for the order history: 20px dot in status color + 2px connector border, label 14px/500 fg (German status label), timestamp 12px fg-muted right, format 'DD.MM.YYYY, HH:MM'. Reached steps filled, future steps empty/dimmed. Shows UTC instants converted to the display time zone.

### LineItemEditor / LineItemTable

Rows for position type 'Arbeitszeit' (Stunden × Stundensatz) and 'Teil' (Bezeichnung, Menge, Einzelpreis). Columns: Bezeichnung, Menge, Einzelpreis, Summe, actions (edit/delete icon buttons 44px). Inline edit: fields in the row, 'Speichern'/'Abbrechen'. Row sum right-aligned tabular-nums. Totals block below: Netto, 19 % Mehrwertsteuer, Brutto — Brutto 18px/600 with 24px top gap. Delete asks for confirmation. Touch targets >=44px, no hover-only actions.

### InvoiceView

Read-only customer invoice: header with 'Rechnung' 24px/600, Auftragsnummer (mono) and Kennzeichen, then line items (Bezeichnung | Menge | Einzelpreis | Summe), then right-aligned totals. Amounts always '1.234,56 €' (two decimals, comma, thousands dot, tabular-nums). Tax line labelled 'MwSt. 19 %'. 'Noch keine Rechnung vorhanden' as quiet empty state while the order is not done. Print-friendly: 1px borders, no background colors carrying meaning.

### KPITile

Dashboard tile: bg=surface, radius lg, border=border, padding 16/24, label 13px fg-muted above, value 32px/600 fg with tabular-nums, optional delta 12px fg-muted. Values: offene Aufträge (count), heute fertig (count), Umsatz laufender Monat (as '12.340,00 €'). Responsive: 1 column phone, 3 columns from 900px.

### Alert / InlineBanner

variant info/success/warning/danger, radius md, padding 12/16, 14px, colored 1px border + soft bg. Used for API errors mapped from {error:{code,message}} — the API message is shown verbatim, plus a 12px fg-muted reference to the code (e.g. 'Code: plate_taken'). Never the only feedback: destructive/blocked actions also disable the control or explain it. Auto-dismiss only for success, with dismiss icon button (44px).

### Toast

Bottom-center on phone / bottom-right on desktop, max-width 400px, bg=fg fg=#FFFFFF radius md padding 12/16, 14px, shadow 0 4px 12px rgba(22,32,43,0.18), auto-dismiss after 4s, dismiss button 44px. Used after 'Position hinzugefügt', 'Status auf fertig gesetzt', never for errors (those are inline).

### ConfirmationDialog

Centered modal, overlay rgba(22,32,43,0.45), panel bg=surface radius lg max-width 420px padding 24, title 18px/600, body 14px fg-muted, actions right-aligned ('Abbrechen' secondary, destructive action primary/danger), min 44px buttons. Full-screen sheet on phone. Esc and overlay click close; focus trapped, focus returns to the trigger.

### OTP / LookupForm (Statusabruf)

Two-field form: 'Auftragsnummer' and 'Kennzeichen' (auto-uppercase, mono), primary 'Status abrufen' full-width on phone. On 404 shows the unified error message inline under the form and keeps the entered values. Result: StatusBadge, StatusTimeline and, if present, InvoiceView; each result block is a Card with 16px gap.

### Skeleton / LoadingState

Surface-sunken blocks radius sm (rows 16px high, card placeholders) with a subtle 1.2s pulse; same layout as the loaded content so nothing shifts. Buttons keep their 44px height while loading. Used for order list, dashboard and invoice.

## Layout Principles

- One content container, max-width 1120px, centered, with 16px side padding on phone (<640px) and 24px from 640px up; use 32px and above only for very wide screens.
- Breakpoints: 640px (phone -> tablet: tables become cards, single to two columns), 900px (two-column layouts, 3-up KPI grid), 1200px (max content width reached, no further stretch). Mobile-first, never a horizontal scrollbar.
- 8px base grid: every margin, padding and gap comes from the spacing scale (4/8/12/16/24/32/48). Gaps inside a group 8-12px, between groups 16px, between sections 32-48px.
- Typography scale is fixed: 12px meta/labels, 14px body and controls, 16px card titles and base, 18px subtitles, 24px page title, 32px KPI value. Headings 600, body 400. No font sizes outside this scale.
- Number and unit formatting is identical everywhere: amounts as Euro with two decimals, comma as decimal separator and dot as thousands separator ('1.234,56 €', negatives '-1.234,56 €'); internal values stay integer cents. Whole-cent rounding only.
- Dates and times, one format only: date 'DD.MM.YYYY' (14.03.2025), time 'HH:MM', combined 'DD.MM.YYYY, HH:MM'; stored UTC, displayed in one fixed display time zone (Europe/Berlin), never a raw ISO string in the UI.
- Other value formats: kilometres '128.450 km' (thousands dot, no decimals), labour time '2,5 Std.' (comma, one decimal), tax rate '19 %' with a narrow space, order number and licence plate in mono uppercase exactly as the API returns them, empty/missing values as an en dash '-'.
- Numbers that are compared or summed (amounts, hours, kilometres, KPIs) are right-aligned and always use tabular-nums so columns line up.
- Status color is the only saturated color in the UI: the five workflow states use the fixed status-* pairs (dot/badge, soft background) with text labels; accent is reserved for primary actions, links and focus, never for decoration.
- Every interactive element has a visible state for default/hover/focus/active/disabled, a minimum 44x44px touch target, and a visible focus ring. An action that is not available is visibly disabled with a reason, never silently inert.
- Form validation messages appear only after blur or a submit attempt, and stay hidden on an untouched form; error text sits directly under its control and never replaces the label.
- Content order per screen: page title (with a one-line description), primary action, filters, content. On phone the primary action and filters stick to the top of the scroll area while the list scrolls.
- Accessibility baseline: minimum contrast 4.5:1 for body text and 3:1 for large text and borders against their background; status is never conveyed by color alone; every icon button has a label.
- Motion is minimal: 120ms ease-out on hover/press, 180ms for overlays; respect prefers-reduced-motion by dropping transitions.
