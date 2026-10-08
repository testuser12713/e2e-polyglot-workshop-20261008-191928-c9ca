"""Access to the Valkey work queue.

The API pushes one JSON message per completed order:
``{"order_id": int, "order_number": str}``. The worker pops the oldest
message; an empty queue is not an error.
"""

from __future__ import annotations

import json
from dataclasses import dataclass

import redis


@dataclass(frozen=True)
class QueueMessage:
    """One job taken from the Valkey list."""

    order_id: int
    order_number: str


class MessageQueue:
    """Reads order messages from the configured Valkey list."""

    def __init__(self, url: str, queue_name: str) -> None:
        self._queue_name = queue_name
        self._client = redis.from_url(url, decode_responses=True)

    @property
    def queue_name(self) -> str:
        """Name of the Valkey list this queue reads from."""
        return self._queue_name

    def ping(self) -> bool:
        """Return whether Valkey is reachable."""
        return bool(self._client.ping())

    def pop(self) -> QueueMessage | None:
        """Pop the oldest message, or ``None`` when the queue is empty."""
        raw = self._client.lpop(self._queue_name)
        if raw is None:
            return None
        data = json.loads(raw)
        return QueueMessage(
            order_id=int(data["order_id"]),
            order_number=str(data["order_number"]),
        )

    def close(self) -> None:
        """Release the Valkey connection."""
        self._client.close()
