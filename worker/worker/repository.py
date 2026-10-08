"""PostgreSQL access for the invoice worker.

``Repository`` owns the database connection, loads an order together with its
customer, vehicle and positions, and stores the generated invoice. Storing an
invoice is idempotent: an order that already has one keeps it untouched.
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
        with self.connection.cursor() as cursor:
            cursor.execute(
                """
                SELECT o.id, o.order_number,
                       c.id, c.name, c.email, c.phone,
                       v.id, v.plate, v.brand, v.model, v.mileage
                FROM orders o
                JOIN customers c ON c.id = o.customer_id
                JOIN vehicles v ON v.id = o.vehicle_id
                WHERE o.id = %s
                """,
                (order_id,),
            )
            row = cursor.fetchone()
            if row is None:
                return None

            cursor.execute(
                """
                SELECT kind, description, hours, quantity, unit_price_cents, total_cents
                FROM order_items
                WHERE order_id = %s
                ORDER BY id
                """,
                (order_id,),
            )
            positions = [
                Position(
                    kind=item[0],
                    description=item[1],
                    hours=float(item[2]) if item[2] is not None else None,
                    quantity=int(item[3]) if item[3] is not None else None,
                    unit_price_cents=int(item[4]),
                    total_cents=int(item[5]),
                )
                for item in cursor.fetchall()
            ]

        return Order(
            id=row[0],
            order_number=row[1],
            customer=Customer(id=row[2], name=row[3], email=row[4], phone=row[5]),
            vehicle=Vehicle(id=row[6], plate=row[7], brand=row[8], model=row[9], mileage=row[10]),
            positions=positions,
        )

    def has_invoice(self, order_id: int) -> bool:
        """Return whether the order already has an invoice stored."""
        with self.connection.cursor() as cursor:
            cursor.execute("SELECT 1 FROM invoices WHERE order_id = %s", (order_id,))
            return cursor.fetchone() is not None

    def store_invoice(self, order: Order, amounts: InvoiceAmounts) -> None:
        """Store exactly one invoice with its positions for an order.

        Idempotent: an invoice is only created when the order does not have one
        yet; an existing invoice is left untouched.

        Args:
            order: The order the invoice belongs to.
            amounts: The computed net, VAT and gross amounts.
        """
        with self.connection.cursor() as cursor:
            cursor.execute("SELECT id FROM invoices WHERE order_id = %s", (order.id,))
            existing = cursor.fetchone()
            if existing is None:
                cursor.execute(
                    """
                    INSERT INTO invoices (order_id, net_cents, vat_cents, gross_cents)
                    VALUES (%s, %s, %s, %s)
                    RETURNING id
                    """,
                    (order.id, amounts.net_cents, amounts.vat_cents, amounts.gross_cents),
                )
                invoice_id = cursor.fetchone()[0]
                for position in order.positions:
                    cursor.execute(
                        """
                        INSERT INTO invoice_items
                            (invoice_id, kind, description, hours, quantity,
                             unit_price_cents, total_cents)
                        VALUES (%s, %s, %s, %s, %s, %s, %s)
                        """,
                        (
                            invoice_id,
                            position.kind,
                            position.description,
                            position.hours,
                            position.quantity,
                            position.unit_price_cents,
                            position.total_cents,
                        ),
                    )
        self.connection.commit()
