"""Terminal colors, paths, CJK detection, readline setup, shared mutable state."""

import atexit
import os
import re

# --- Terminal color support ---
_COLORS = {
    "source": "\033[36m",   # cyan
    "word": "\033[33m",     # yellow
    "label": "\033[32m",    # green
    "pron": "\033[35m",     # purple
    "dim": "\033[2m",       # dim
    "warn": "\033[31m",     # red
    "reset": "\033[0m",
}

_USE_COLOR = True


def _c(name, text):
    """Wrap text in ANSI color if colors are enabled."""
    if not _USE_COLOR:
        return text
    code = _COLORS.get(name)
    if code is None:
        return text
    return f"{code}{text}{_COLORS['reset']}"


# config.py lives in ecdlib/, so the package dir is one level up from us,
# and ecd.db is in the project root (parent of the package dir).
_PKG_DIR = os.path.dirname(os.path.abspath(__file__))
DB_PATH = os.path.join(os.path.dirname(_PKG_DIR), "ecd.db")
HISTORY_DB = os.path.expanduser("~/.ecd_lookup.db")
HISTFILE = os.path.expanduser("~/.ecd_history")


# --- Readline support ---
try:
    import readline
except ImportError:
    readline = None


def _save_history():
    if readline is None:
        return
    try:
        readline.write_history_file(HISTFILE)
    except (PermissionError, OSError):
        pass


if readline:
    try:
        readline.read_history_file(HISTFILE)
    except (FileNotFoundError, PermissionError, OSError):
        pass
    atexit.register(_save_history)


# --- CJK detection ---
CJK_RE = re.compile(r"[一-鿿㐀-䶿豈-﫿]")


def is_chinese_query(text):
    return bool(CJK_RE.search(text))


# --- Shared mutable state ---
_last_word = ""
_auto_add = False

# --- Language ---
from .lang import set_lang as _set_lang, get_lang as _get_lang

_LANG = _get_lang()
