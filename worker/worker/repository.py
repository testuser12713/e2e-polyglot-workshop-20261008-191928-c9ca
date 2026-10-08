"""PostgreSQL access for the invoice worker.

``Repository`` owns the database connection, loads an order together with its
customer, vehicle and positions, and stores the generated invoice. The query
bodies are delivered by the "Implement invoice generation in the worker"
ticket; the signatures below are the contract that ticket fills.
"""

from __future__ import annotations

from dataclasses import dataclass, field

import psycopg
from psycopg import Connection

from .calc import InvoiceAmounts, Position


@dataclass(frozen=True)
class Customer:
    """The customer an order belongs to."""

    id: int
    name: str
    email: str
    phone: str


@dataclass(frozen=True)
class Vehicle:
    """The vehicle an order belongs to."""

    id: int
    plate: str
    brand: str
    model: str
    mileage: int


@dataclass(frozen=True)
class Order:
    """An order ready to be invoiced."""

    id: int
    order_number: str
    customer: Customer
    vehicle: Vehicle
    positions: list[Position] = field(default_factory=list)


class Repository:
    """Database gateway for order loading and invoice storage."""

    def __init__(self, database_url: str) -> None:
        self._database_url = database_url
        self._connection: Connection | None = None

    @property
    def connection(self) -> Connection:
        """The open database connection.

        Raises:
            RuntimeError: when ``connect`` has not run yet.
        """
        if self._connection is None:
            raise RuntimeError("repository is not connected; call connect() first")
        return self._connection

    def connect(self) -> None:
        """Open the database connection and verify it answers."""
        self._connection = psycopg.connect(self._database_url)
        with self._connection.cursor() as cursor:
            cursor.execute("SELECT 1")

    def close(self) -> None:
        """Close the database connection if it is open."""
        if self._connection is not None:
            self._connection.close()
            self._connection = None

    def load_order(self, order_id: int) -> Order | None:
        """Load an order with its customer, vehicle and positions.

        Args:
            order_id: Primary key of the order.

        Returns:
            The order, or ``None`` when no such order exists.
        """
        raise NotImplementedError("order loading is implemented by ticket #15")

    def store_invoice(self, order: Order, amounts: InvoiceAmounts) -> None:
        """Store exactly one invoice with its positions for an order.

        Must be idempotent: storing an invoice for an order that already has
        one leaves the existing invoice untouched.

        Args:
            order: The order the invoice belongs to.
            amounts: The computed net, VAT and gross amounts.
        """
        raise NotImplementedError("invoice storage is implemented by ticket #15")
