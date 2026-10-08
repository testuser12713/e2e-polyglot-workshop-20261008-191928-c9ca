"""Invoice worker for the workshop customer portal.

The worker consumes completed orders from the Valkey list ``QUEUE_NAME``,
computes the invoice amounts from the order positions and the configured
hourly rate, stores the invoice in PostgreSQL and writes one notification
to the outbox.
"""

__all__ = ["__version__"]

__version__ = "0.1.0"
