"""Configuration for the invoice worker.

Every value is read from the environment lazily, so importing this module
never fails. ``load_config`` validates the whole configuration once and names
the first missing variable instead of letting the process die with a bare
traceback. The variables and their dev values are declared in ``RUN.json``.
"""

from __future__ import annotations

import os
from dataclasses import dataclass

DEFAULT_QUEUE_NAME = "workshop-invoices"
DEFAULT_HOURLY_RATE_CENTS = 8900


@dataclass(frozen=True)
class Config:
    """Worker configuration resolved from the environment."""

    database_url: str
    valkey_url: str
    queue_name: str
    hourly_rate_cents: int


def _required(name: str) -> str:
    value = os.environ.get(name)
    if not value:
        raise RuntimeError(
            f"{name} is not set. Declare it in RUN.json (see worker/README.md) "
            "and export it before starting the worker."
        )
    return value


def load_config() -> Config:
    """Read and validate the worker configuration.

    Raises:
        RuntimeError: when a required variable (``DATABASE_URL``,
            ``VALKEY_URL``) is missing or ``HOURLY_RATE_CENTS`` is not a
            whole number of cents.
    """
    try:
        hourly_rate_cents = int(os.environ.get("HOURLY_RATE_CENTS", str(DEFAULT_HOURLY_RATE_CENTS)))
    except ValueError as exc:
        raise RuntimeError(
            "HOURLY_RATE_CENTS must be an integer number of cents, "
            f"got {os.environ.get('HOURLY_RATE_CENTS')!r}."
        ) from exc

    return Config(
        database_url=_required("DATABASE_URL"),
        valkey_url=_required("VALKEY_URL"),
        queue_name=os.environ.get("QUEUE_NAME", DEFAULT_QUEUE_NAME),
        hourly_rate_cents=hourly_rate_cents,
    )
