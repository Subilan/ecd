"""ecd_extract — Build ecd.db from Apple Dictionary bundles."""

from .build import build_db
from .cli import main
from .parse import parse_tabfile

__all__ = ["build_db", "main", "parse_tabfile"]
