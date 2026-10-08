"""Outbox writer for invoice notifications.

The worker never sends a real e-mail; it stores one notification row per
invoice in the ``outbox`` table: recipient e-mail, subject and body.
"""

from __future__ import annotations

from psycopg import Connection


def write_notification(
    connection: Connection,
    *,
    recipient: str,
    subject: str,
    body: str,
) -> None:
    """Write one pending notification to the outbox.

    Args:
        connection: Open PostgreSQL connection to insert with.
        recipient: Recipient e-mail address.
        subject: Notification subject line.
        body: Notification body text.
    """
    with connection.cursor() as cursor:
        cursor.execute(
            "INSERT INTO outbox (recipient, subject, body) VALUES (%s, %s, %s)",
            (recipient, subject, body),
        )
    connection.commit()
