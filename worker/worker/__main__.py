"""``python -m worker`` support. The ticket's start command runs this."""

from .main import main

if __name__ == "__main__":
    raise SystemExit(main())
