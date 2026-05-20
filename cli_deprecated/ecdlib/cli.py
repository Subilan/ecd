"""CLI argument parsing and main entry point."""

import argparse
import os
import sqlite3
import sys

from .config import DB_PATH, _USE_COLOR, _c
from .interactive import interactive
from .search import handle_query, random_word


def main():
    global _USE_COLOR

    parser = argparse.ArgumentParser(
        description="Query the ecd dictionary database"
    )
    parser.add_argument(
        "-s",
        "--source",
        choices=["collins", "oxford"],
        help="Search only one dictionary",
    )
    parser.add_argument(
        "-r",
        "--random",
        action="store_true",
        dest="random_word_flag",
        help="Fetch a random word from the dictionary",
    )
    parser.add_argument(
        "--no-color",
        action="store_true",
        help="Disable ANSI color output",
    )
    parser.add_argument(
        "query",
        nargs="*",
        help="Word or phrase to look up",
    )
    args = parser.parse_args()

    import ecdlib.config as cfg

    if args.no_color or not sys.stdout.isatty():
        cfg._USE_COLOR = False

    if not os.path.exists(DB_PATH):
        print(f"Error: database not found at {DB_PATH}", file=sys.stderr)
        print("Run extract/build_db.py first.", file=sys.stderr)
        sys.exit(1)

    db = sqlite3.connect(DB_PATH)
    db.execute("PRAGMA foreign_keys = ON")

    if args.random_word_flag:
        word = random_word(db, args.source)
        if word:
            handle_query(db, word, args.source)
        else:
            print("No words found in the database.")
    elif args.query:
        query = " ".join(args.query)
        handle_query(db, query, args.source)
    else:
        interactive(db, args.source)


if __name__ == "__main__":
    main()
