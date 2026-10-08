"""Outbox writer for invoice notifications.

The worker never sends a real e-mail; it stores one notification row per
invoice in the ``outbox`` table. The insert is delivered by the
"Implement invoice generation in the worker" ticket; the signature below is
the contract that ticket fills.
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
    raise NotImplementedError("outbox writing is implemented by ticket #15")
