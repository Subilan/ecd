"""Path configuration and constants for ecd_extract."""

import os
import shutil
import sys
from pathlib import Path

EXTRACT_DIR = Path(__file__).resolve().parent.parent
PROJECT_ROOT = EXTRACT_DIR.parent

DEFAULT_COLLINS = os.path.expanduser(
    "~/Library/Dictionaries/柯林斯高阶英汉双解词典.dictionary"
)
DEFAULT_OXFORD = os.path.expanduser(
    "~/Library/Dictionaries/牛津高阶英汉双解词典（第8版）.dictionary"
)
DEFAULT_DB = str(PROJECT_ROOT / "ecd.db")
SCHEMA_SQL = str(EXTRACT_DIR / "schema.sql")

# Prefer pyglossary from same venv as this python; fall back to PATH
PYGLOSSARY = shutil.which(
    "pyglossary", path=str(Path(sys.executable).parent) + os.pathsep + os.environ.get("PATH", "")
)
if not PYGLOSSARY:
    sys.exit("pyglossary not found. Run: extract/.venv/bin/pip install -r extract/requirements.txt")
