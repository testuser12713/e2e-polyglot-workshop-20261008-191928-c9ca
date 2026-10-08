"""Invoice amount calculation.

Pure functions over positions and the configured hourly rate. Labor positions
are billed as ``hours * hourly_rate_cents``, part positions take their stored
``total_cents``; 19 % VAT is added on the net sum. Every amount is a whole
number of cents, rounded commercially (half up).
"""

from __future__ import annotations

from collections.abc import Sequence
from dataclasses import dataclass
from decimal import ROUND_HALF_UP, Decimal

VAT_RATE_PERCENT = 19

_CENT = Decimal("1")


def _round_cents(amount: Decimal) -> int:
    """Round a Decimal amount to whole cents, commercially (half up)."""
    return int(amount.quantize(_CENT, rounding=ROUND_HALF_UP))


@dataclass(frozen=True)
class Position:
    """One invoice position.

    ``kind`` is either ``"labor"`` (``hours`` is set, ``quantity`` is null)
    or ``"part"`` (``quantity`` is set, ``hours`` is null). All amounts are
    whole cents.
    """

    kind: str
    description: str
    hours: float | None
    quantity: int | None
    unit_price_cents: int
    total_cents: int


@dataclass(frozen=True)
class InvoiceAmounts:
    """Net, VAT and gross amounts of an invoice, all whole cents."""

    net_cents: int
    vat_cents: int
    gross_cents: int


def position_total_cents(position: Position, hourly_rate_cents: int) -> int:
    """Return the billable amount of one position in whole cents.

    A labor position is ``hours * hourly_rate_cents``; a part position is its
    stored ``total_cents``. The result is rounded commercially.
    """
    if position.kind == "labor":
        hours = Decimal(str(position.hours or 0))
        return _round_cents(hours * Decimal(hourly_rate_cents))
    return int(position.total_cents)


def compute_amounts(positions: Sequence[Position], hourly_rate_cents: int) -> InvoiceAmounts:
    """Compute the invoice amounts from positions and the hourly rate.

    Labor positions are billed as ``hours * hourly_rate_cents``, parts take
    their stored ``total_cents``; 19 % VAT is added on the net sum. All
    amounts are whole cents, rounded commercially.

    Args:
        positions: The order positions to bill.
        hourly_rate_cents: The configured hourly rate in cents.

    Returns:
        The net, VAT and gross amounts of the invoice.
    """
    net_cents = sum(position_total_cents(position, hourly_rate_cents) for position in positions)
    vat_cents = _round_cents(Decimal(net_cents) * Decimal(VAT_RATE_PERCENT) / Decimal(100))
    return InvoiceAmounts(
        net_cents=net_cents,
        vat_cents=vat_cents,
        gross_cents=net_cents + vat_cents,
    )
