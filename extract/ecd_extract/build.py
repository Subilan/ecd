"""Database build orchestrator — creates schema, runs extraction, inserts data."""

import os
import sqlite3
import sys
import tempfile
from pathlib import Path

from .config import DEFAULT_DB, SCHEMA_SQL
from .parse import parse_tabfile
from .utils import extract_tabfile


def build_db(db_path=None, dicts_to_process=None, dict_paths=None):
    if db_path is None:
        db_path = DEFAULT_DB
    if dicts_to_process is None:
        dicts_to_process = ["collins", "oxford"]
    if dict_paths is None:
        from .config import DEFAULT_COLLINS, DEFAULT_OXFORD
        dict_paths = {"collins": DEFAULT_COLLINS, "oxford": DEFAULT_OXFORD}

    schema = Path(SCHEMA_SQL).read_text(encoding="utf-8")

    db_path = os.path.abspath(db_path)
    if os.path.exists(db_path):
        os.remove(db_path)

    conn = sqlite3.connect(db_path)
    conn.execute("PRAGMA foreign_keys = ON")
    conn.executescript(schema)
    conn.commit()

    with tempfile.TemporaryDirectory() as tmpdir:
        for source in dicts_to_process:
            dict_path = dict_paths[source]
            tabfile = os.path.join(tmpdir, f"{source}.txt")

            print(f"Extracting {source} dictionary...", file=sys.stderr)
            extract_tabfile(dict_path, tabfile)

            print(f"Parsing {source} entries...", file=sys.stderr)
            entry_rows, example_batches, synonym_batches, antonym_batches = parse_tabfile(tabfile, source)

            table = f"{source}_entries"
            print(
                f"Inserting {len(entry_rows)} entries into {table}...",
                file=sys.stderr,
            )

            # Insert entries and track assigned IDs
            conn.execute("BEGIN")
            entry_ids = []
            for row in entry_rows:
                cur = conn.execute(
                    f"INSERT OR IGNORE INTO {table} (word, pos, cn_definition, cross_ref, sense_order, pronunciation, extra_notes) VALUES (?, ?, ?, ?, ?, ?, ?)",
                    row,
                )
                if cur.lastrowid != 0:
                    entry_ids.append(cur.lastrowid)
                else:
                    # Row was ignored due to UNIQUE constraint - find existing ID
                    cur2 = conn.execute(
                        f"SELECT id FROM {table} WHERE word=? AND pos=? AND sense_order=?",
                        (row[0], row[1], row[4]),
                    )
                    entry_ids.append(cur2.fetchone()[0])
            conn.commit()

            # Build example rows with correct entry_id
            example_table = f"{source}_examples"
            example_rows = []
            for entry_id, batch in zip(entry_ids, example_batches):
                for en_text, cn_text, order in batch:
                    example_rows.append((entry_id, en_text, cn_text, order))

            print(
                f"Inserting {len(example_rows)} examples into {example_table}...",
                file=sys.stderr,
            )
            conn.execute("BEGIN")
            conn.executemany(
                f"INSERT INTO {example_table} (entry_id, en_example, cn_example, example_order) VALUES (?, ?, ?, ?)",
                example_rows,
            )
            conn.commit()

            # Insert synonyms (both dictionaries)
            synonym_rows = []
            for entry_id, synonyms in zip(entry_ids, synonym_batches):
                for syn_word in synonyms:
                    synonym_rows.append((source, entry_id, syn_word))
            if synonym_rows:
                conn.execute("BEGIN")
                conn.executemany(
                    "INSERT INTO synonyms (source, entry_id, synonym_word) VALUES (?, ?, ?)",
                    synonym_rows,
                )
                conn.commit()
                print(
                    f"Inserting {len(synonym_rows)} synonyms...",
                    file=sys.stderr,
                )

            # Insert antonyms (both dictionaries)
            antonym_rows = []
            for entry_id, antonyms in zip(entry_ids, antonym_batches):
                for ant_word in antonyms:
                    antonym_rows.append((source, entry_id, ant_word))
            if antonym_rows:
                conn.execute("BEGIN")
                conn.executemany(
                    "INSERT INTO antonyms (source, entry_id, antonym_word) VALUES (?, ?, ?)",
                    antonym_rows,
                )
                conn.commit()
                print(
                    f"Inserting {len(antonym_rows)} antonyms...",
                    file=sys.stderr,
                )

    # Populate FTS5
    print("Populating FTS5 index...", file=sys.stderr)
    for source in dicts_to_process:
        conn.execute(
            f"""
            INSERT INTO entries_fts (source, word, cn_definition, en_example, cn_example)
            SELECT '{source}', e.word, e.cn_definition, x.en_example, x.cn_example
            FROM {source}_entries e
            LEFT JOIN {source}_examples x ON e.id = x.entry_id
            WHERE e.cross_ref IS NULL OR e.cross_ref = ''
            """
        )
    conn.commit()

    # Report
    for source in dicts_to_process:
        total = conn.execute(
            f"SELECT COUNT(*) FROM {source}_entries"
        ).fetchone()[0]
        xref = conn.execute(
            f"SELECT COUNT(*) FROM {source}_entries WHERE cross_ref IS NOT NULL AND cross_ref != ''"
        ).fetchone()[0]
        examples = conn.execute(
            f"SELECT COUNT(*) FROM {source}_examples"
        ).fetchone()[0]
        print(f"  {source}: {total} entries ({xref} xref), {examples} examples")

    conn.close()
    print(f"Done: {db_path}", file=sys.stderr)
