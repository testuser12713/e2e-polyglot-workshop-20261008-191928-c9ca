"""Smoke test: one worker pass against the real PostgreSQL and Valkey.

No mocking: the pass opens a real PostgreSQL connection and pops from the
real Valkey list. The test only clears its own queue key and never touches
tables owned by other tickets.
"""

from __future__ import annotations

import logging

import pytest
import redis
from worker.config import load_config
from worker.main import main


def test_single_pass_connects_and_reports_empty_queue(caplog: pytest.LogCaptureFixture) -> None:
    config = load_config()
    client = redis.from_url(config.valkey_url, decode_responses=True)
    client.delete(config.queue_name)

    with caplog.at_level(logging.INFO):
        exit_code = main(["--once"])

    assert exit_code == 0
    assert config.queue_name in caplog.text
    assert "empty" in caplog.text
    assert client.llen(config.queue_name) == 0
    client.close()


def test_load_config_names_missing_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.delenv("DATABASE_URL", raising=False)

    with pytest.raises(RuntimeError, match="DATABASE_URL"):
        load_config()
