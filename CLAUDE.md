# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Setup (first time)

```bash
python3 -m venv extract/.venv
extract/.venv/bin/pip install -r extract/requirements.txt
```

## Build & Run

```bash
# Build the database (takes ~30s)
extract/.venv/bin/python extract/build_db.py

# Rebuild only one dictionary
extract/.venv/bin/python extract/build_db.py --only oxford

# Run the CLI (only needs stdlib sqlite3)
python3 ecd hello          # English → both dictionaries
python3 ecd 水             # Chinese → reverse lookup via FTS5
python3 ecd -s oxford beauty
```

The `extract/.venv/` and `ecd.db` (~80 MB) are git-ignored.

## Architecture

The project extracts two macOS Apple Dictionary `.dictionary` bundles (Collins COBUILD English-Chinese and Oxford Advanced Learner's 8th Ed) into a SQLite database.

### Database schema (`extract/schema.sql`)

Each dictionary has two tables with identical column layouts — separate tables because POS grammar systems and HTML structures differ significantly:

- **`{dict}_entries`** — one row per word sense. Columns: `word`, `pos` (e.g. `N-COUNT`, `noun [U]`), `cn_definition`, `cross_ref` (for pure redirects like "went" → "go"), `sense_order`. Unique on `(word, pos, sense_order)`.
- **`{dict}_examples`** — 1:N from entries. Columns: `entry_id` FK, `en_example`, `cn_example`, `example_order`.
- **`entries_fts`** — shared FTS5 virtual table (`unicode61` tokenizer) for Chinese reverse lookup. Populated from both dictionaries' non-xref entries joined with examples.

Cross-reference entries (e.g. "went" = past tense of "go", "mice" = plural of "mouse") have `pos=''` and a non-null `cross_ref` pointing to the canonical word. They are excluded from FTS5.

### Extraction (`extract/build_db.py`)

`build_db()` is the top-level orchestrator. For each dictionary it:
1. Shells out to `pyglossary --read-format AppleDictBin` to produce a tabfile
2. `parse_tabfile()` reads the tabfile line-by-line, dispatches to per-dictionary parsers
3. Entries and examples are inserted in explicit transactions, then FTS5 is populated

**Collins HTML structure**: `div.collins_en_cn` blocks. POS from `span.st`, definition from `span.def_cn`, examples from `<li>` > two `<p>` tags. Xref detection: no `st`/`li` elements + caption matches keyword patterns ("past tense of", "plural of", "means the same as", "→see").

**Oxford HTML structure**: Two patterns:
- Pattern 1 (e.g. "water"): `span.p-g` blocks under `.entry`, each with `.pos` + `n-g` children containing `.gr` tags and `.x-g` examples.
- Pattern 2 (e.g. "beauty"): No `p-g` — `n-g` blocks are direct children of `h-g`, POS from `top-g > block-g > pos-g > pos`.

Xref detection in Oxford: `.sense-g .xr-g` present, no `.p-g`, no `.n-g`.

### CLI (`ecd`)

Python script that auto-detects Chinese (CJK range) vs. English queries. English queries use `LIKE` with `COLLATE NOCASE`; Chinese queries use FTS5 MATCH. Falls back to Chinese search if English search finds nothing.

## File editing

The user's global CLAUDE.md forbids `sed` for editing, `cat -A`/`od`/`xxd` for reading, and `git checkout`/`git stash` to discard changes. When the native edit tool fails, fall back to `Write` for full-file overwrites.
