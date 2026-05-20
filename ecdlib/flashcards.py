"""Flashcard deck statistics and review session."""

import json
import sys
import termios
import tty
from datetime import datetime, timedelta, timezone

from .config import _c
from .db import _ensure_history_db, add_flashcard, del_flashcard
from .display import _print_entry_body
from .search import search_english
from .sm2 import _sm2_schedule


def _get_due_cards(conn, limit=20):
    """Fetch cards where next_review <= now."""
    return conn.execute(
        """
        SELECT word, ease_factor, interval_days, repetitions, next_review,
               total_reviews, total_correct
        FROM flashcards
        WHERE next_review <= datetime('now')
        ORDER BY next_review
        LIMIT ?
        """,
        (limit,),
    ).fetchall()


def _get_card_count(conn):
    return conn.execute("SELECT COUNT(*) FROM flashcards").fetchone()[0]


def _get_due_count(conn):
    return conn.execute(
        "SELECT COUNT(*) FROM flashcards WHERE next_review <= datetime('now')"
    ).fetchone()[0]


def _get_new_count(conn):
    return conn.execute(
        "SELECT COUNT(*) FROM flashcards WHERE repetitions = 0"
    ).fetchone()[0]


def _get_mature_count(conn):
    return conn.execute(
        "SELECT COUNT(*) FROM flashcards WHERE interval_days >= 21"
    ).fetchone()[0]


def _get_avg_ease(conn):
    row = conn.execute("SELECT AVG(ease_factor) FROM flashcards").fetchone()[0]
    return round(row, 2) if row else 0.0


def _get_leech_count(conn):
    return conn.execute(
        "SELECT COUNT(*) FROM flashcards WHERE ease_factor <= 1.3"
    ).fetchone()[0]


def _get_next_review_info(conn):
    """Return (delta_str, is_overdue) for the nearest next_review."""
    row = conn.execute(
        "SELECT MIN(next_review) FROM flashcards"
    ).fetchone()[0]
    if not row:
        return None, False
    next_dt = datetime.strptime(row, "%Y-%m-%d %H:%M:%S")
    now = datetime.now(tz=timezone.utc).replace(tzinfo=None)
    delta = next_dt - now
    total_secs = int(delta.total_seconds())
    is_overdue = total_secs <= 0
    abs_secs = abs(total_secs)

    days = abs_secs // 86400
    hours = (abs_secs % 86400) // 3600
    mins = (abs_secs % 3600) // 60

    if days > 0:
        if hours > 0:
            delta_str = f"{days}d {hours}h"
        else:
            delta_str = f"{days}d"
    elif hours > 0:
        if mins > 0:
            delta_str = f"{hours}h {mins}m"
        else:
            delta_str = f"{hours}h"
    else:
        delta_str = f"{mins}m"
    return delta_str, is_overdue


def print_deck_stats():
    """Print flashcard deck statistics."""
    conn = _ensure_history_db()
    total = _get_card_count(conn)
    if total == 0:
        print("Deck is empty. Use .add to add words.")
        conn.close()
        return
    due = _get_due_count(conn)
    newc = _get_new_count(conn)
    mature = _get_mature_count(conn)
    leeches = _get_leech_count(conn)
    avg_ef = _get_avg_ease(conn)
    next_str, overdue = _get_next_review_info(conn)
    conn.close()

    print(f"\n{_c('label', 'Deck Statistics')}")
    print(f"  {_c('label', 'Total:')}        {total}")
    print(f"  {_c('label', 'Due:')}          {due}")
    print(f"  {_c('label', 'New:')}          {newc}")
    print(f"  {_c('label', 'Mature:')}       {mature}")

    if next_str:
        if overdue:
            suffix = _c('warn', ' ago')
            print(f"  {_c('label', 'Next:')}         {next_str}{suffix}")
        else:
            print(f"  {_c('label', 'Next:')}         in {next_str}")

    if leeches > 0:
        print(f"  {_c('label', 'Leeches:')}      {leeches}")

    print(f"  {_c('label', 'Avg ease:')}     {avg_ef:.0%}")
    print()


def _get_key():
    """Read a single keypress. Returns 'LEFT', 'RIGHT', or the character."""
    fd = sys.stdin.fileno()
    old_settings = termios.tcgetattr(fd)
    try:
        tty.setraw(fd)
        ch = sys.stdin.read(1)
        if ch == '\x1b':
            ch2 = sys.stdin.read(1)
            if ch2 == '[':
                ch3 = sys.stdin.read(1)
                if ch3 == 'D':
                    return 'LEFT'
                elif ch3 == 'C':
                    return 'RIGHT'
        elif ch in ('\x03', '\x04'):
            raise KeyboardInterrupt
        return ch
    finally:
        termios.tcsetattr(fd, termios.TCSADRAIN, old_settings)


def _print_flashcard_entry(entry, idx, total):
    """Print one dictionary entry for flashcard review."""
    source_names = {"collins": "柯林斯", "oxford": "牛津"}
    src = source_names.get(entry["source"], entry["source"])
    pos_str = entry.get("pos", "") or "(none)"
    print(f"\n  {_c('dim', f'释义 {idx + 1}/{total}  [{src}]')}")
    print(f"  {_c('label', 'POS:')} {pos_str}")
    _print_entry_body(entry, indent="  ")
    if total > 1:
        print(f"\n  {_c('dim', '← → 切换释义  |  0-3 评分')}")
    else:
        print(f"\n  {_c('dim', '0-3 评分')}")


def review_session(db_dict):
    """Run an interactive review session for due flashcards."""
    conn = _ensure_history_db()
    due = _get_due_cards(conn)

    if not due:
        print("No cards due for review!")
        total = _get_card_count(conn)
        if total > 0:
            delta_str, is_overdue = _get_next_review_info(conn)
            if is_overdue:
                print(f"Deck has {total} cards. Next was due {delta_str} ago — run again to review.")
            else:
                print(f"Deck has {total} cards. Next due in {delta_str}.")
        conn.close()
        return

    total_cards = len(due)

    for i, card in enumerate(due, 1):
        word, ef, interval_days, reps, next_review, total_rev, total_corr = card

        results = search_english(db_dict, word)

        print(f"\n{'=' * 40}")
        print(f"Card {i}/{total_cards}")
        print(f"{'=' * 40}")

        # Front of card
        pron_str = ""
        if results:
            pron = results[0].get("pronunciation", "")
            if pron:
                try:
                    ipas = json.loads(pron)
                    if ipas:
                        pron_str = f"{_c('dim', ' /')}{_c('pron', ' | '.join(ipas))}{_c('dim', '/')}"
                except (json.JSONDecodeError, TypeError):
                    pass
        print(f"\n  {_c('word', word)}{pron_str}")

        try:
            input("\nPress Enter to reveal answer...")
        except (EOFError, KeyboardInterrupt):
            print("\nReview session cancelled.")
            conn.close()
            return

        # Back of card
        if results:
            entry_idx = 0
            num_entries = len(results)

            print("\033[s", end="", flush=True)
            _print_flashcard_entry(results[0], 0, num_entries)

            while True:
                try:
                    key = _get_key()
                except KeyboardInterrupt:
                    print("\nReview session cancelled.")
                    conn.close()
                    return

                if key == 'LEFT':
                    entry_idx = (entry_idx - 1) % num_entries
                    print("\033[u\033[J", end="", flush=True)
                    _print_flashcard_entry(results[entry_idx], entry_idx, num_entries)
                elif key == 'RIGHT':
                    entry_idx = (entry_idx + 1) % num_entries
                    print("\033[u\033[J", end="", flush=True)
                    _print_flashcard_entry(results[entry_idx], entry_idx, num_entries)
                elif key in ('0', '1', '2', '3'):
                    button = int(key)
                    break
        else:
            print(f"\n  {_c('dim', '(word not found in dictionary database)')}")
            # Rating
            while True:
                try:
                    inp = input(
                        f"\n{_c('label', 'Rate:')} "
                        f"0={_c('dim', 'Again')} "
                        f"1={_c('dim', 'Hard')} "
                        f"2={_c('dim', 'Good')} "
                        f"3={_c('dim', 'Easy')}: "
                    ).strip()
                    if not inp:
                        continue
                    button = int(inp)
                    if button not in (0, 1, 2, 3):
                        print("Please enter 0, 1, 2, or 3.")
                        continue
                    break
                except ValueError:
                    print("Please enter a number (0-3).")
                except (EOFError, KeyboardInterrupt):
                    print("\nReview session cancelled.")
                    conn.close()
                    return

        # Apply SM-2
        new_reps, new_int, new_ef = _sm2_schedule(button, reps, interval_days, ef)
        now_str = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        next_review_dt = datetime.now() + timedelta(days=new_int)
        next_review_str = next_review_dt.strftime("%Y-%m-%d %H:%M:%S")

        conn.execute(
            """
            UPDATE flashcards
            SET ease_factor = ?,
                interval_days = ?,
                repetitions = ?,
                next_review = ?,
                last_review = ?,
                total_reviews = total_reviews + 1,
                total_correct = total_correct + ?
            WHERE word = ?
            """,
            (new_ef, new_int, new_reps, next_review_str, now_str,
             1 if button >= 2 else 0, word),
        )
        conn.commit()

    conn.close()
    print(f"\n{_c('label', 'Review session complete.')} {total_cards} card(s) reviewed.")


def _add_word_with_check(db, word):
    """Look up word in dictionary, show summary, then add to flashcards.
    Returns True if the word was added (or already in deck), False if user declined.
    """
    results = search_english(db, word)
    if results:
        r = results[0]
        pos_str = f" ({r['pos']})" if r.get("pos") else ""
        cn_def = r.get("cn_definition", "")
        print(f"{_c('word', word)}{_c('dim', pos_str)}", end="")
        if cn_def:
            # Keep summary brief — first definition line only
            first_line = cn_def.split("；")[0].split("；")[0].split("。")[0]
            if len(first_line) > 60:
                first_line = first_line[:60] + "..."
            print(f"  {first_line}")
        else:
            print()
    else:
        print(f"'{word}' not found in dictionaries.")
        try:
            answer = input("Add anyway? (y/n): ").strip().lower()
        except (EOFError, KeyboardInterrupt):
            print()
            return False
        if answer not in ("y", "yes"):
            return False

    added = add_flashcard(word)
    if added:
        print(f"Added '{word}' to flashcard deck.")
    else:
        print(f"'{word}' is already in your flashcard deck.")
    return True


def _delete_word_with_check(word):
    """Delete a word from the flashcard deck. Returns True if deleted, False if not found."""
    deleted = del_flashcard(word)
    if deleted:
        print(f"Removed '{word}' from flashcard deck.")
    else:
        print(f"'{word}' is not in your flashcard deck.")
    return deleted
