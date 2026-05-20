"""Dictionary search functions and query dispatch."""

import difflib
import random

from .config import is_chinese_query
from .db import record_lookup
from .display import print_results_chinese, print_results_english


def search_english(db, word, source=None):
    """Search by English word. Returns list of (source, entry, examples)."""
    sources = [source] if source else ["collins", "oxford"]
    results = []
    for src in sources:
        rows = db.execute(
            f"""
            SELECT e.id, e.word, e.pos, e.cn_definition, e.cross_ref, e.sense_order, e.pronunciation, e.extra_notes
            FROM {src}_entries e
            WHERE e.word = ? COLLATE NOCASE
            ORDER BY e.pos, e.sense_order
            """,
            (word,),
        ).fetchall()

        for row in rows:
            entry_id, w, pos, cn_def, cross_ref, sense_order, pronunciation, extra_notes = row
            examples = db.execute(
                f"""
                SELECT en_example, cn_example
                FROM {src}_examples
                WHERE entry_id = ?
                ORDER BY example_order
                """,
                (entry_id,),
            ).fetchall()
            result = {
                "entry_id": entry_id,
                "source": src,
                "word": w,
                "pos": pos,
                "cn_definition": cn_def,
                "cross_ref": cross_ref,
                "sense_order": sense_order,
                "pronunciation": pronunciation,
                "extra_notes": extra_notes,
                "examples": [(e[0], e[1]) for e in examples],
            }
            # Fetch synonyms for Collins entries
            syn_rows = db.execute(
                "SELECT synonym_word FROM synonyms WHERE source=? AND entry_id=? ORDER BY id",
                (src, entry_id),
            ).fetchall()
            result["synonyms"] = [r[0] for r in syn_rows]
            ant_rows = db.execute(
                "SELECT antonym_word FROM antonyms WHERE source=? AND entry_id=? ORDER BY id",
                (src, entry_id),
            ).fetchall()
            result["antonyms"] = [r[0] for r in ant_rows]
            results.append(result)
    return results


def search_english_prefix(db, word, source=None):
    """Search by English word prefix. Returns list like search_english."""
    sources = [source] if source else ["collins", "oxford"]
    results = []
    for src in sources:
        rows = db.execute(
            f"""
            SELECT e.id, e.word, e.pos, e.cn_definition, e.cross_ref, e.sense_order, e.pronunciation, e.extra_notes
            FROM {src}_entries e
            WHERE e.word LIKE ? COLLATE NOCASE
            ORDER BY e.word, e.pos, e.sense_order
            """,
            (word + "%",),
        ).fetchall()

        for row in rows:
            entry_id, w, pos, cn_def, cross_ref, sense_order, pronunciation, extra_notes = row
            examples = db.execute(
                f"""
                SELECT en_example, cn_example
                FROM {src}_examples
                WHERE entry_id = ?
                ORDER BY example_order
                """,
                (entry_id,),
            ).fetchall()
            result = {
                "entry_id": entry_id,
                "source": src,
                "word": w,
                "pos": pos,
                "cn_definition": cn_def,
                "cross_ref": cross_ref,
                "sense_order": sense_order,
                "pronunciation": pronunciation,
                "extra_notes": extra_notes,
                "examples": [(e[0], e[1]) for e in examples],
            }
            syn_rows = db.execute(
                "SELECT synonym_word FROM synonyms WHERE source=? AND entry_id=? ORDER BY id",
                (src, entry_id),
            ).fetchall()
            result["synonyms"] = [r[0] for r in syn_rows]
            ant_rows = db.execute(
                "SELECT antonym_word FROM antonyms WHERE source=? AND entry_id=? ORDER BY id",
                (src, entry_id),
            ).fetchall()
            result["antonyms"] = [r[0] for r in ant_rows]
            results.append(result)
    return results


def search_english_fuzzy(db, word, source=None):
    """Fuzzy search for English words. Returns list of close match strings."""
    sources = [source] if source else ["collins", "oxford"]
    prefix = word[0] if word else ""
    candidates = set()
    for src in sources:
        rows = db.execute(
            f"SELECT DISTINCT word FROM {src}_entries WHERE word LIKE ? COLLATE NOCASE",
            (prefix + "%",),
        ).fetchall()
        candidates.update(r[0] for r in rows)
    return difflib.get_close_matches(word, candidates, n=5, cutoff=0.75)


def search_chinese(db, text, source=None):
    """Search by Chinese text using FTS5."""
    # FTS5 requires specific syntax
    rows = db.execute(
        f"""
        SELECT source, word, cn_definition, en_example, cn_example
        FROM entries_fts
        WHERE entries_fts MATCH ?
        ORDER BY rank
        LIMIT 50
        """,
        (text,),
    ).fetchall()

    results = {}
    for row in rows:
        src, word, cn_def, en_example, cn_example = row
        if source and src != source:
            continue
        key = (src, word, cn_def or "")
        if key not in results:
            results[key] = []
        if cn_example:
            results[key].append(cn_example)
        elif en_example:
            results[key].append(en_example)

    return results


def random_word(db, source=None):
    """Fetch a random word entry from the database."""
    sources = [source] if source else ["collins", "oxford"]
    src = random.choice(sources)
    row = db.execute(
        f"SELECT word FROM {src}_entries WHERE cn_definition != '' AND cross_ref IS NULL ORDER BY RANDOM() LIMIT 1"
    ).fetchone()
    if row is None:
        return None
    return row[0]


def handle_query(db, query, source):
    import ecdlib.config as cfg

    if is_chinese_query(query):
        results = search_chinese(db, query, source)
        if not results:
            print(f"No results for: {query}")
            return
        print_results_chinese(results)
        record_lookup(query)
        cfg._last_word = query
    else:
        results = search_english(db, query, source)
        if results:
            print_results_english(results)
            record_lookup(query.lower())
            cfg._last_word = query.lower()
            return

        # No exact match — try prefix search
        prefix_results = search_english_prefix(db, query, source)
        if prefix_results:
            distinct_words = sorted(set(r["word"] for r in prefix_results))
            if len(distinct_words) == 1:
                word = distinct_words[0]
                print_results_english(prefix_results)
                record_lookup(word.lower())
                cfg._last_word = word.lower()
            else:
                print(f"Did you mean: {', '.join(distinct_words[:10])}?")
            return

        # No prefix match — try fuzzy search
        fuzzy_matches = search_english_fuzzy(db, query, source)
        if fuzzy_matches:
            print(f"Did you mean: {', '.join(fuzzy_matches)}?")
            return

        # Fall through to Chinese FTS5
        results_cn = search_chinese(db, query, source)
        if results_cn:
            print_results_chinese(results_cn)
            record_lookup(query)
            cfg._last_word = query
        else:
            print(f"No results for: {query}")
