"""History database connection and operations."""

import sqlite3

from .config import HISTORY_DB


def _ensure_history_db():
    conn = sqlite3.connect(HISTORY_DB)
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS lookup_history (
            word TEXT NOT NULL PRIMARY KEY,
            count INTEGER NOT NULL DEFAULT 1,
            last_query TEXT NOT NULL DEFAULT (datetime('now'))
        )
        """
    )
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS flashcards (
            word TEXT NOT NULL PRIMARY KEY,
            created TEXT NOT NULL DEFAULT (datetime('now')),
            ease_factor REAL NOT NULL DEFAULT 2.5,
            interval_days INTEGER NOT NULL DEFAULT 0,
            repetitions INTEGER NOT NULL DEFAULT 0,
            next_review TEXT NOT NULL DEFAULT (datetime('now', '+10 minutes')),
            last_review TEXT,
            total_reviews INTEGER NOT NULL DEFAULT 0,
            total_correct INTEGER NOT NULL DEFAULT 0
        )
        """
    )
    conn.commit()
    return conn


def record_lookup(word):
    conn = _ensure_history_db()
    conn.execute(
        """
        INSERT INTO lookup_history (word, count, last_query)
        VALUES (?, 1, datetime('now'))
        ON CONFLICT(word) DO UPDATE SET
            count = count + 1,
            last_query = datetime('now')
        """,
        (word,),
    )
    conn.commit()
    conn.close()


def add_flashcard(word):
    """Add a word to the flashcard deck. Returns True if new, False if already exists."""
    conn = _ensure_history_db()
    try:
        conn.execute("INSERT INTO flashcards (word) VALUES (?)", (word,))
        conn.commit()
        return True
    except sqlite3.IntegrityError:
        return False
    finally:
        conn.close()


def del_flashcard(word):
    """Delete a word from the flashcard deck. Returns True if deleted, False if not found."""
    conn = _ensure_history_db()
    try:
        cur = conn.execute("DELETE FROM flashcards WHERE word = ?", (word,))
        conn.commit()
        return cur.rowcount > 0
    finally:
        conn.close()
