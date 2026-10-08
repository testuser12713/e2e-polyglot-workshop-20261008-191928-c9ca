"""End-to-end pipeline test: queue -> repository -> calc -> repository -> outbox.

Runs against the real PostgreSQL and Valkey, no doubles. The test creates the
schema it uses and only ever touches rows it inserted itself.
"""

from __future__ import annotations

import json
import os
import uuid

import psycopg
import pytest
import redis
from worker.calc import compute_amounts
from worker.config import Config
from worker.main import process_one
from worker.queue import MessageQueue
from worker.repository import Repository

QUEUE_NAME = "workshop-invoices-pipeline-test"
HOURLY_RATE_CENTS = 8900

SCHEMA_SQL = """
CREATE TABLE IF NOT EXISTS customers (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name text NOT NULL,
    email text NOT NULL,
    phone text NOT NULL
);
CREATE TABLE IF NOT EXISTS vehicles (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    plate text NOT NULL UNIQUE,
    brand text NOT NULL,
    model text NOT NULL,
    mileage integer NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS orders (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_number text NOT NULL UNIQUE,
    status text NOT NULL DEFAULT 'requested',
    customer_id integer NOT NULL REFERENCES customers(id),
    vehicle_id integer NOT NULL REFERENCES vehicles(id),
    preferred_date date,
    description text,
    created_at timestamptz NOT NULL DEFAULT now(),
    labor_cents integer NOT NULL DEFAULT 0,
    parts_cents integer NOT NULL DEFAULT 0,
    net_cents integer NOT NULL DEFAULT 0,
    vat_cents integer NOT NULL DEFAULT 0,
    gross_cents integer NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS order_items (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id integer NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    kind text NOT NULL,
    description text NOT NULL DEFAULT '',
    hours numeric(8, 2),
    quantity integer,
    unit_price_cents integer NOT NULL DEFAULT 0,
    total_cents integer NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS invoices (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id integer NOT NULL UNIQUE REFERENCES orders(id),
    net_cents integer NOT NULL,
    vat_cents integer NOT NULL,
    gross_cents integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS invoice_items (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invoice_id integer NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    kind text NOT NULL,
    description text NOT NULL DEFAULT '',
    hours numeric(8, 2),
    quantity integer,
    unit_price_cents integer NOT NULL DEFAULT 0,
    total_cents integer NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS outbox (
    id integer GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    recipient text NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
"""


@pytest.fixture(scope="module", autouse=True)
def _schema() -> None:
    """Create the tables this pipeline uses before any test runs."""
    connection = psycopg.connect(os.environ["DATABASE_URL"])
    try:
        with connection.cursor() as cursor:
            cursor.execute(SCHEMA_SQL)
        connection.commit()
    finally:
        connection.close()


@pytest.fixture
def repository() -> Repository:
    repo = Repository(os.environ["DATABASE_URL"])
    repo.connect()
    yield repo
    repo.close()


@pytest.fixture
def valkey() -> redis.Redis:
    client = redis.from_url(os.environ["VALKEY_URL"], decode_responses=True)
    client.delete(QUEUE_NAME)
    yield client
    client.delete(QUEUE_NAME)
    client.close()


def _config() -> Config:
    return Config(
        database_url=os.environ["DATABASE_URL"],
        valkey_url=os.environ["VALKEY_URL"],
        queue_name=QUEUE_NAME,
        hourly_rate_cents=HOURLY_RATE_CENTS,
    )


def _insert_order(repository: Repository, *, email: str, plate: str, order_number: str) -> dict:
    with repository.connection.cursor() as cursor:
        cursor.execute(
            "INSERT INTO customers (name, email, phone) VALUES (%s, %s, %s) RETURNING id",
            ("Testkunde", email, "0123-456789"),
        )
        customer_id = cursor.fetchone()[0]
        cursor.execute(
            "INSERT INTO vehicles (plate, brand, model, mileage) "
            "VALUES (%s, %s, %s, %s) RETURNING id",
            (plate, "VW", "Golf", 120000),
        )
        vehicle_id = cursor.fetchone()[0]
        cursor.execute(
            "INSERT INTO orders (order_number, status, customer_id, vehicle_id) "
            "VALUES (%s, %s, %s, %s) RETURNING id",
            (order_number, "done", customer_id, vehicle_id),
        )
        order_id = cursor.fetchone()[0]
        entries = [
            ("labor", "Arbeitszeit", 2.5, None, HOURLY_RATE_CENTS, 22250),
            ("part", "Bremsbelag", None, 1, 12345, 12345),
        ]
        for kind, description, hours, quantity, unit_price, total in entries:
            cursor.execute(
                "INSERT INTO order_items "
                "(order_id, kind, description, hours, quantity, unit_price_cents, total_cents) "
                "VALUES (%s, %s, %s, %s, %s, %s, %s)",
                (order_id, kind, description, hours, quantity, unit_price, total),
            )
    repository.connection.commit()
    return {
        "order_id": order_id,
        "customer_id": customer_id,
        "vehicle_id": vehicle_id,
        "order_number": order_number,
    }


def _cleanup(repository: Repository, ids: dict, email: str) -> None:
    with repository.connection.cursor() as cursor:
        cursor.execute("DELETE FROM outbox WHERE recipient = %s", (email,))
        cursor.execute("DELETE FROM invoices WHERE order_id = %s", (ids["order_id"],))
        cursor.execute("DELETE FROM order_items WHERE order_id = %s", (ids["order_id"],))
        cursor.execute("DELETE FROM orders WHERE id = %s", (ids["order_id"],))
        cursor.execute("DELETE FROM vehicles WHERE id = %s", (ids["vehicle_id"],))
        cursor.execute("DELETE FROM customers WHERE id = %s", (ids["customer_id"],))
    repository.connection.commit()


def _count(repository: Repository, table: str, where: str, params: tuple) -> int:
    with repository.connection.cursor() as cursor:
        cursor.execute(f"SELECT count(*) FROM {table} WHERE {where}", params)
        return cursor.fetchone()[0]


def test_one_pass_stores_invoice_and_one_notification(repository: Repository, valkey) -> None:
    email = f"{uuid.uuid4().hex}@example.com"
    plate = f"T{uuid.uuid4().hex[:6].upper()}"
    order_number = f"AU-{uuid.uuid4().hex[:8].upper()}"
    ids = _insert_order(repository, email=email, plate=plate, order_number=order_number)
    order = repository.load_order(ids["order_id"])
    expected = compute_amounts(order.positions, HOURLY_RATE_CENTS)

    valkey.rpush(
        QUEUE_NAME,
        json.dumps({"order_id": ids["order_id"], "order_number": order_number}),
    )
    queue = MessageQueue(os.environ["VALKEY_URL"], QUEUE_NAME)
    try:
        assert process_one(_config(), repository, queue) is True

        with repository.connection.cursor() as cursor:
            cursor.execute(
                "SELECT net_cents, vat_cents, gross_cents FROM invoices WHERE order_id = %s",
                (ids["order_id"],),
            )
            invoice = cursor.fetchone()
        assert invoice == (expected.net_cents, expected.vat_cents, expected.gross_cents)
        assert invoice[0] == 34595
        assert invoice[1] == 6573
        assert invoice[2] == 41168

        assert (
            _count(
                repository,
                "invoice_items",
                "invoice_id = (SELECT id FROM invoices WHERE order_id = %s)",
                (ids["order_id"],),
            )
            == 2
        )
        assert _count(repository, "outbox", "recipient = %s", (email,)) == 1
        with repository.connection.cursor() as cursor:
            cursor.execute("SELECT subject, body FROM outbox WHERE recipient = %s", (email,))
            subject, body = cursor.fetchone()
        assert order_number in subject
        assert body
    finally:
        queue.close()
        _cleanup(repository, ids, email)


def test_redelivered_message_creates_neither_second_invoice_nor_notification(
    repository: Repository, valkey
) -> None:
    email = f"{uuid.uuid4().hex}@example.com"
    plate = f"T{uuid.uuid4().hex[:6].upper()}"
    order_number = f"AU-{uuid.uuid4().hex[:8].upper()}"
    ids = _insert_order(repository, email=email, plate=plate, order_number=order_number)
    message = json.dumps({"order_id": ids["order_id"], "order_number": order_number})

    valkey.rpush(QUEUE_NAME, message, message)
    queue = MessageQueue(os.environ["VALKEY_URL"], QUEUE_NAME)
    try:
        assert process_one(_config(), repository, queue) is True
        assert process_one(_config(), repository, queue) is True

        assert _count(repository, "invoices", "order_id = %s", (ids["order_id"],)) == 1
        assert _count(repository, "outbox", "recipient = %s", (email,)) == 1
    finally:
        queue.close()
        _cleanup(repository, ids, email)


def test_unknown_order_is_skipped_without_error(repository: Repository, valkey) -> None:
    valkey.rpush(
        QUEUE_NAME,
        json.dumps({"order_id": 2_000_000_000, "order_number": "AU-MISSING"}),
    )
    queue = MessageQueue(os.environ["VALKEY_URL"], QUEUE_NAME)
    try:
        assert process_one(_config(), repository, queue) is True
    finally:
        queue.close()
