"""Entry point of the invoice worker.

``python -m worker --once`` runs a single pass and exits; without ``--once``
the worker polls the queue in a loop with a short sleep and logs every pass.
"""

from __future__ import annotations

import argparse
import logging
import time

from .calc import compute_amounts
from .config import Config, load_config
from .outbox import write_notification
from .queue import MessageQueue
from .repository import Repository

logger = logging.getLogger("worker")

POLL_INTERVAL_SECONDS = 5.0


def process_one(config: Config, repository: Repository, queue: MessageQueue) -> bool:
    """Process at most one queue message.

    Returns:
        ``True`` when a message was taken from the queue (also when its order
        could not be found), ``False`` when the queue was empty.
    """
    message = queue.pop()
    if message is None:
        logger.info("queue %s is empty", config.queue_name)
        return False

    logger.info("processing order %s (%s)", message.order_number, message.order_id)
    order = repository.load_order(message.order_id)
    if order is None:
        logger.warning("order %s (%s) not found; skipping", message.order_id, message.order_number)
        return True

    amounts = compute_amounts(order.positions, config.hourly_rate_cents)
    repository.store_invoice(order, amounts)
    write_notification(
        repository.connection,
        recipient=order.customer.email,
        subject=f"Rechnung zu Auftrag {order.order_number}",
        body=(
            f"Ihre Rechnung zu Auftrag {order.order_number} liegt vor. "
            f"Gesamtbetrag: {amounts.gross_cents} Cent."
        ),
    )
    logger.info("invoice stored for order %s", order.order_number)
    return True


def main(argv: list[str] | None = None) -> int:
    """Run the worker. Returns the process exit code."""
    parser = argparse.ArgumentParser(
        prog="worker",
        description="Consume completed workshop orders and write invoices.",
    )
    parser.add_argument(
        "--once",
        action="store_true",
        help="run a single pass and exit instead of polling forever",
    )
    args = parser.parse_args(argv)

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s: %(message)s",
    )

    config = load_config()
    repository = Repository(config.database_url)
    queue = MessageQueue(config.valkey_url, config.queue_name)
    repository.connect()
    logger.info("connected to PostgreSQL and Valkey (queue %s)", config.queue_name)

    try:
        if args.once:
            process_one(config, repository, queue)
            return 0

        logger.info("worker started; polling every %.1fs", POLL_INTERVAL_SECONDS)
        while True:
            process_one(config, repository, queue)
            time.sleep(POLL_INTERVAL_SECONDS)
    except KeyboardInterrupt:
        logger.info("worker stopped")
        return 0
    finally:
        queue.close()
        repository.close()
