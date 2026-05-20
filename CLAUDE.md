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
# Option A: via installed console script
pip install -e extract/ && ecd-build

# Option B: via python -m (from repo root)
extract/.venv/bin/python -m ecd_extract

# Option C: install in editable mode
extract/.venv/bin/pip install -e extract/
extract/.venv/bin/ecd-build

# Rebuild only one dictionary
extract/.venv/bin/python -m ecd_extract --only oxford

# Run the CLI (only needs stdlib sqlite3)
./ecd hello                 # English → both dictionaries
python3 ecd 水              # Chinese → reverse lookup via FTS5
python3 ecd -s oxford beauty
./ecd                       # Interactive mode (Ctrl-D or .exit to quit)
python3 -m ecdlib            # Same, via module
```

The `extract/.venv/` and `ecd.db` (~80 MB) are git-ignored.

## Architecture

The project extracts two macOS Apple Dictionary `.dictionary` bundles (Collins COBUILD English-Chinese and Oxford Advanced Learner's 8th Ed) into a SQLite database.

### Database schema (`extract/schema.sql`)

Each dictionary has two tables with identical column layouts — separate tables because POS grammar systems and HTML structures differ significantly:

- **`{dict}_entries`** — one row per word sense. Columns: `word`, `pos` (e.g. `N-COUNT`, `noun [U]`), `cn_definition`, `cross_ref` (for pure redirects like "went" → "go"), `sense_order`, `pronunciation` (JSON array of IPA strings, e.g. `["həˈləʊ","həˈloʊ"]`), `extra_notes` (JSON array of `{"type":"...","en":"...","cn":"..."}` for usage notes, regional variants, derived forms, quotations from `<figure class="note">` elements). Unique on `(word, pos, sense_order)`.
- **`{dict}_examples`** — 1:N from entries. Columns: `entry_id` FK, `en_example`, `cn_example`, `example_order`.
- **`entries_fts`** — shared FTS5 virtual table (`unicode61` tokenizer) for Chinese reverse lookup. Columns: `source`, `word`, `cn_definition`, `en_example`, `cn_example`. Populated from both dictionaries' non-xref entries joined with examples.
- **`synonyms`** — synonym relationships from both dictionaries. Columns: `source` (`collins`/`oxford`), `entry_id`, `synonym_word`. Collins: extracted from `<div class="synonym">` blocks. Oxford: extracted from `.xr-g` elements containing `.symbols-synsym` (SYN) markers or `.z_xr "synonyms at"` cross-references.
- **`antonyms`** — antonym relationships from both dictionaries. Same schema as `synonyms` with `antonym_word`. Primarily from Oxford `.xr-g` elements containing `.symbols-oppsym` (OPP) markers.

Cross-reference entries (e.g. "went" = past tense of "go", "mice" = plural of "mouse") have `pos=''` and a non-null `cross_ref` pointing to the canonical word. They are excluded from FTS5.

### Extraction (`extract/ecd_extract/`)

Modular package. Key modules:
- **`build.py`**: `build_db()` top-level orchestrator. For each dictionary it:
  1. Shells out to `pyglossary --read-format AppleDictBin` to produce a tabfile
  2. `parse_tabfile()` reads the tabfile line-by-line, dispatches to per-dictionary parsers
  3. Entries and examples are inserted in explicit transactions, then FTS5 is populated
- **`parse.py`**: `parse_tabfile()` dispatcher — reads tabfile, routes entries to Collins or Oxford parsers
- **`collins.py`**: Collins parser (xref detection, regular entries, DRV entries, notes, synonyms, pronunciation)
- **`oxford.py`**: Oxford parser (xref detection, regular entries across 4 HTML patterns, idioms, pronunciation)
- **`utils.py`**: Shared utilities (`extract_tabfile`, `itertext`, `child_elements`, `clean_ipa`)
- **`config.py`**: Paths, constants, pyglossary location

**Collins word key**: For some entries (e.g. `'cause` = informal "because"), the tabfile key starts with a leading apostrophe, but the HTML `<span class="word_key">` contains the canonical form. The parser prefers `word_key` from HTML when available.

**Collins HTML structure**: `div.collins_en_cn` blocks. POS from `span.st`, definition from `span.def_cn`, examples from `<li>` > two `<p>` tags (excluding `<li>` inside `<figure class="note">`). Xref detection: no `st`/`li` elements + caption matches keyword patterns ("past tense of", "plural of", "means the same as", "→see"). Entries with no `cn_definition` and no examples and no extra_notes are skipped (e.g. "See also:", thesaurus headings like "VERB."/"N."/"ADJ.").

**Collins extra_notes**: `<figure class="note type-*">` elements captured separately from examples. Each figure produces one note combining inline text and `<li>` examples. Quotation figures (`type-quotation`) parsed from individual `.cit` elements (`blockquote p` or `span.quote` for text, `cite` for attribution), with multiple quotes joined by double newlines and `<br>` converted to newlines. Other figures strip dictionary UI labels ("Usage Note :", "Quotations"). Orphan `figure.note` elements outside `.collins_en_cn` are attached to the first entry with a definition.

**Collins derived forms**: `<figure class="note type-drv">` elements are NOT treated as extra_notes. Instead, `_extract_drv_entries()` creates standalone entries with `pos='DRV'`, the derived word from `<b>` tag, and examples from `<li><p>` pairs. Duplicates are deduplicated within the same parent entry by `seen_drv_words` set; cross-parent duplicates use `INSERT OR IGNORE` with fallback ID lookup.

**Collins synonyms**: `<div class="synonym">` blocks containing `<span class="form">` elements (both `<a>` links and plain text) are extracted by `_extract_collins_synonyms()` and stored in the `synonyms` table.

**Oxford synonyms/antonyms**: `.xr-g` cross-reference elements are parsed by `_extract_oxford_xrefs()`. `.symbols-oppsym` (OPP) markers yield antonyms; `.symbols-synsym` (SYN) markers and `.z_xr` elements containing "synonyms at" yield synonym cross-references. Stored in the `synonyms` and `antonyms` tables.

**Collins pronunciation**: `span.pron` at `word_entry` level (one set per word, shared by all senses). Extracted from `span.pron.type_uk` and `span.pron.type_us`. IPA text contains HTML markup (`<u>`, `<sup>`) stripped by `_clean_ipa()`. Applied to all entries for the word in `parse_tabfile()`.

**Oxford HTML structure**: Four patterns:
- Pattern 1 (e.g. "water"): `span.p-g` blocks under `.entry`, each with `.pos` + `n-g` children containing `.gr` tags and `.x-g` examples.
- Pattern 2 (e.g. "beauty"): No `p-g` — `n-g` blocks are direct children of `h-g`, POS from `top-g > block-g > pos-g > pos`.
- Pattern 3 (e.g. "cause" verb, "abandon" noun, "above" adj): `p-g` block present but `.def-g` and `.x-g` are direct children of `p-g` (no `.n-g` wrapper). Handled by `_oxford_parse_pg_direct()`.
- Pattern 4 (e.g. "incantation", "A1"): No `p-g` or `n-g` — `def-g` sits directly under `h-g`. POS from `top-g > block-g > pos-g > pos`, grammar from `top-g > .gr`. Handled by `_oxford_parse_hg_direct()`. Two sub-patterns: (4a) direct `def-g` under `h-g` with sibling `.x-g` examples, (4b) `ids-g` idioms where each `id-g > sense-g` becomes a separate entry with `IDM` prefix in POS.

Xref detection in Oxford: `.sense-g .xr-g` present, no `.p-g`, no `.n-g`; OR `.entry > .derived` present with no `.p-g`, `.n-g` (derived-form redirects like "abasement" → "abase").

**Oxford pronunciation**: `span.ei-g` blocks containing `span.phon-gb` (UK) and `span.phon-usgb`/`span.phon-us` (US). Two placement patterns: (a) word-level in `top-g > ei-g` (e.g. "refuse" verb/noun are separate `<span class="entry">` elements, each with own `top-g > ei-g`), (b) per-POS inside `p-g > ei-g` (e.g. "record" has `ei-g` inside noun `p-g` and verb `p-g` with different IPA). `_extract_oxford_pronunciation()` checks the POS-group container first, falls back to `top-g`.

**Entry filtering**: Both Oxford and Collins parsers skip entries that have no `cn_definition` and no examples — these are noise (thesaurus headings, "See also:" links, abbreviation expansions without Chinese translation). Entries with examples but no Chinese definition are kept (English-only definitions with example sentences still have value).

### CLI (`ecd` + `ecdlib/`)

The root `ecd` file is a thin wrapper that imports from the `ecdlib/` package. The package is modular:
- **`cli.py`**: `main()` — argument parsing, DB connection, dispatch to interactive/query/random
- **`interactive.py`**: `interactive()` REPL loop with all `.` commands
- **`search.py`**: `search_english()`, `search_english_prefix()`, `search_english_fuzzy()`, `search_chinese()`, `random_word()`, `handle_query()` dispatch
- **`display.py`**: `print_results_english()`, `print_results_chinese()`, `_print_entry_body()`
- **`flashcards.py`**: `review_session()`, `print_deck_stats()`, `_add_word_with_check()`, `_get_key()`, `_print_flashcard_entry()`, SM-2 helpers. Review session supports left/right arrow keys to cycle through all dictionary entries for a word.
- **`sm2.py`**: `_sm2_schedule()` algorithm and constants
- **`db.py`**: `_ensure_history_db()`, `record_lookup()`, `add_flashcard()`
- **`config.py`**: Colors, paths, CJK detection, readline setup, shared mutable state

Auto-detects Chinese (CJK range) vs. English queries. English queries use `LIKE` with `COLLATE NOCASE`; Chinese queries use FTS5 MATCH. Falls back to Chinese search if English search finds nothing.

Displays pronunciation (parsed from JSON) inline on the word line with dimmed `/` delimiters, alongside definitions, synonyms, and antonyms from both dictionaries. Extra notes (`[用法]`, `[名言]`, `[释义补充]`, `[注]`) are displayed with the label on its own line followed by content without indentation. Records lookup history in `~/.ecd_lookup.db` (separate from the stateless `ecd.db`) — upserts the queried word with count + last-query timestamp on any result-bearing query (exact match, prefix match with single result, Chinese FTS5 hit). "Did you mean" suggestions are not recorded.

When run without arguments, enters interactive mode with a `> ` prompt and sets the terminal title to "ecd". Commands: `.exit`/`.quit`/`.q` to quit, `.add [word]` to add a word to flashcard deck with dictionary lookup preview, `.del [word]` to remove a word from the flashcard deck, `.auto-add [on|off]` to toggle auto-adding looked-up words, `.review` for SM-2 spaced repetition review (all dictionary entries shown; ←/→ arrow keys cycle through senses, 0-3 to rate), `.deck` for deck statistics, `.reset` to clear all flashcard data, `.syn [word]` to show synonyms from both dictionaries, `.ant [word]` to show antonyms.

## File editing

The user's global CLAUDE.md forbids `sed` for editing, `cat -A`/`od`/`xxd` for reading, and `git checkout`/`git stash` to discard changes. When the native edit tool fails, fall back to `Write` for full-file overwrites.
