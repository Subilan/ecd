"""Tabfile parsing dispatcher — routes entries to Collins or Oxford parser."""

import sys

from lxml import html

from .collins import (
    _extract_collins_pronunciation,
    is_collins_xref,
    parse_collins_regular,
    parse_collins_xref,
)
from .oxford import is_oxford_xref, parse_oxford_regular, parse_oxford_xref


def parse_tabfile(filepath, source):
    """Parse a tabfile, return (entry_rows, example_batches, synonym_batches, antonym_batches)."""
    entry_rows = []
    example_batches = []
    synonym_batches = []
    antonym_batches = []

    with open(filepath, encoding="utf-8") as f:
        for lineno, line in enumerate(f, 1):
            if lineno % 5000 == 0:
                print(f"  {source}: {lineno} lines", file=sys.stderr)

            parts = line.split("\t", 1)
            if len(parts) < 2:
                continue
            word = parts[0].split("|")[0].strip()
            html_text = parts[1]

            try:
                tree = html.fromstring(html_text)
            except Exception:
                continue

            # Prefer word_key from HTML (handles Collins entries like "'cause"
            # whose tabfile key has a leading apostrophe while the HTML has "cause")
            if source == "collins":
                wk_els = tree.cssselect(".word_key")
                if wk_els:
                    word = wk_els[0].text_content().strip()

            if source == "collins":
                if is_collins_xref(tree):
                    entries, entry_synonyms = parse_collins_xref(tree, word)
                else:
                    entries, entry_synonyms = parse_collins_regular(tree, word)
                entry_antonyms = [[] for _ in entries]
                # Apply word-level pronunciation to all Collins entries
                pron = _extract_collins_pronunciation(tree)
                for e in entries:
                    e["pronunciation"] = pron
            else:
                if is_oxford_xref(tree):
                    entries, entry_synonyms, entry_antonyms = parse_oxford_xref(tree, word)
                else:
                    entries, entry_synonyms, entry_antonyms = parse_oxford_regular(tree, word)

            for entry in entries:
                entry_rows.append(
                    (
                        entry["word"],
                        entry["pos"],
                        entry["cn_definition"],
                        entry["cross_ref"],
                        entry["sense_order"],
                        entry["pronunciation"],
                        entry.get("extra_notes"),
                    )
                )
                example_batches.append(
                    [
                        (en, cn, i + 1)
                        for i, (en, cn) in enumerate(entry["examples"])
                    ]
                )
                synonym_batches.append(entry_synonyms.pop(0) if entry_synonyms else [])
                antonym_batches.append(entry_antonyms.pop(0) if entry_antonyms else [])

    return entry_rows, example_batches, synonym_batches, antonym_batches
