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
python3 ecd                 # Interactive mode (Ctrl-D or .exit to quit)
```

The `extract/.venv/` and `ecd.db` (~80 MB) are git-ignored.

## Architecture

The project extracts two macOS Apple Dictionary `.dictionary` bundles (Collins COBUILD English-Chinese and Oxford Advanced Learner's 8th Ed) into a SQLite database.

### Database schema (`extract/schema.sql`)

Each dictionary has two tables with identical column layouts — separate tables because POS grammar systems and HTML structures differ significantly:

- **`{dict}_entries`** — one row per word sense. Columns: `word`, `pos` (e.g. `N-COUNT`, `noun [U]`), `cn_definition`, `cross_ref` (for pure redirects like "went" → "go"), `sense_order`, `pronunciation` (JSON array of IPA strings, e.g. `["həˈləʊ","həˈloʊ"]`). Unique on `(word, pos, sense_order)`.
- **`{dict}_examples`** — 1:N from entries. Columns: `entry_id` FK, `en_example`, `cn_example`, `example_order`.
- **`entries_fts`** — shared FTS5 virtual table (`unicode61` tokenizer) for Chinese reverse lookup. Populated from both dictionaries' non-xref entries joined with examples.

Cross-reference entries (e.g. "went" = past tense of "go", "mice" = plural of "mouse") have `pos=''` and a non-null `cross_ref` pointing to the canonical word. They are excluded from FTS5.

### Extraction (`extract/build_db.py`)

`build_db()` is the top-level orchestrator. For each dictionary it:
1. Shells out to `pyglossary --read-format AppleDictBin` to produce a tabfile
2. `parse_tabfile()` reads the tabfile line-by-line, dispatches to per-dictionary parsers
3. Entries and examples are inserted in explicit transactions, then FTS5 is populated

**Collins word key**: For some entries (e.g. `'cause` = informal "because"), the tabfile key starts with a leading apostrophe, but the HTML `<span class="word_key">` contains the canonical form. The parser prefers `word_key` from HTML when available.

**Collins HTML structure**: `div.collins_en_cn` blocks. POS from `span.st`, definition from `span.def_cn`, examples from `<li>` > two `<p>` tags. Xref detection: no `st`/`li` elements + caption matches keyword patterns ("past tense of", "plural of", "means the same as", "→see"). Entries with no `cn_definition` and no examples are skipped (e.g. "See also:", thesaurus headings like "VERB."/"N."/"ADJ.").

**Collins pronunciation**: `span.pron` at `word_entry` level (one set per word, shared by all senses). Extracted from `span.pron.type_uk` and `span.pron.type_us`. IPA text contains HTML markup (`<u>`, `<sup>`) stripped by `_clean_ipa()`. Applied to all entries for the word in `parse_tabfile()`.

**Oxford HTML structure**: Three patterns:
- Pattern 1 (e.g. "water"): `span.p-g` blocks under `.entry`, each with `.pos` + `n-g` children containing `.gr` tags and `.x-g` examples.
- Pattern 2 (e.g. "beauty"): No `p-g` — `n-g` blocks are direct children of `h-g`, POS from `top-g > block-g > pos-g > pos`.
- Pattern 3 (e.g. "cause" verb, "abandon" noun, "above" adj): `p-g` block present but `.def-g` and `.x-g` are direct children of `p-g` (no `.n-g` wrapper). Handled by `_oxford_parse_pg_direct()`.

Xref detection in Oxford: `.sense-g .xr-g` present, no `.p-g`, no `.n-g`.

**Oxford pronunciation**: `span.ei-g` blocks containing `span.phon-gb` (UK) and `span.phon-usgb`/`span.phon-us` (US). Two placement patterns: (a) word-level in `top-g > ei-g` (e.g. "refuse" verb/noun are separate `<span class="entry">` elements, each with own `top-g > ei-g`), (b) per-POS inside `p-g > ei-g` (e.g. "record" has `ei-g` inside noun `p-g` and verb `p-g` with different IPA). `_extract_oxford_pronunciation()` checks the POS-group container first, falls back to `top-g`.

**Entry filtering**: Both Oxford and Collins parsers skip entries that have no `cn_definition` and no examples — these are noise (thesaurus headings, "See also:" links, abbreviation expansions without Chinese translation). Entries with examples but no Chinese definition are kept (English-only definitions with example sentences still have value).

### CLI (`ecd`)

Python script that auto-detects Chinese (CJK range) vs. English queries. English queries use `LIKE` with `COLLATE NOCASE`; Chinese queries use FTS5 MATCH. Falls back to Chinese search if English search finds nothing.

Displays pronunciation (parsed from JSON) alongside definitions. Records lookup history in `~/.ecd_lookup.db` (separate from the stateless `ecd.db`) — upserts the queried word with count + last-query timestamp on any result-bearing query (exact match, prefix match with single result, Chinese FTS5 hit). "Did you mean" suggestions are not recorded.

When run without arguments, enters interactive mode with a `> ` prompt. Supports `.exit`, `.quit`, `.q` to quit, Ctrl-D/Ctrl-C also exit.

## File editing

The user's global CLAUDE.md forbids `sed` for editing, `cat -A`/`od`/`xxd` for reading, and `git checkout`/`git stash` to discard changes. When the native edit tool fails, fall back to `Write` for full-file overwrites.
