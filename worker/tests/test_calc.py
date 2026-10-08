"""Unit tests for the invoice amount calculation (AC-08)."""

from __future__ import annotations

from worker.calc import Position, compute_amounts, position_total_cents

HOURLY_RATE = 8900


def _labor(hours: float, total_cents: int = 0) -> Position:
    return Position(
        kind="labor",
        description="Arbeitszeit",
        hours=hours,
        quantity=None,
        unit_price_cents=HOURLY_RATE,
        total_cents=total_cents,
    )


def _part(quantity: int, unit_price_cents: int, total_cents: int) -> Position:
    return Position(
        kind="part",
        description="Teil",
        hours=None,
        quantity=quantity,
        unit_price_cents=unit_price_cents,
        total_cents=total_cents,
    )


def test_labor_plus_parts_with_19_percent_vat() -> None:
    positions = [_labor(2.5), _part(1, 12345, 12345)]

    amounts = compute_amounts(positions, HOURLY_RATE)

    assert amounts.net_cents == 22250 + 12345
    assert amounts.vat_cents == 6573
    assert amounts.gross_cents == amounts.net_cents + amounts.vat_cents


def test_labor_uses_the_configured_hourly_rate_not_the_stored_total() -> None:
    position = _labor(1.0, total_cents=999999)

    assert position_total_cents(position, HOURLY_RATE) == 8900


def test_amounts_are_rounded_half_up_to_whole_cents() -> None:
    positions = [_labor(0.1)]

    amounts = compute_amounts(positions, 12345)

    assert amounts.net_cents == 1235
    assert amounts.vat_cents == 235
    assert amounts.gross_cents == 1470


def test_empty_positions_yield_zero_amounts() -> None:
    amounts = compute_amounts([], HOURLY_RATE)

    assert amounts.net_cents == 0
    assert amounts.vat_cents == 0
    assert amounts.gross_cents == 0


def test_multiple_labor_positions_are_summed() -> None:
    positions = [_labor(1.0), _labor(0.5)]

    amounts = compute_amounts(positions, HOURLY_RATE)

    assert amounts.net_cents == 8900 + 4450
    assert amounts.gross_cents == amounts.net_cents + amounts.vat_cents
