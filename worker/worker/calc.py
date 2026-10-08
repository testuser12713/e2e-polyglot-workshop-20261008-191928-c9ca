"""Invoice amount calculation.

Pure functions over positions and the configured hourly rate. The concrete
computation is delivered by the "Implement invoice generation in the worker"
ticket; the signatures below are the contract that ticket fills.
"""

from __future__ import annotations

from collections.abc import Sequence
from dataclasses import dataclass


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
    raise NotImplementedError("invoice calculation is implemented by ticket #15")
