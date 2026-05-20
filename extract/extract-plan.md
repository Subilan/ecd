# Dictionary SQLite Database — Extraction Plan

## Context

Extract two macOS Apple Dictionary bundles (Collins COBUILD English-Chinese and Oxford Advanced Learner's English-Chinese 8th Ed) into a SQLite database for local lookup. The HTML content within each `.dictionary` bundle needs to be parsed, structured, and stored in a relational schema that supports efficient word lookup (English → Chinese) and reverse lookup (Chinese → English).

## Project Structure

```
ecd/
├── extract/                  # Extraction + DB build project (self-contained)
│   ├── schema.sql            # DDL for all tables
│   ├── build_db.py           # Extraction & population script
│   └── requirements.txt      # pyglossary, lxml (venv: /tmp/dict_venv)
├── ecd.db                    # Generated SQLite database (in .gitignore)
└── ecd                       # CLI lookup script (step 3)
```

## Schema Design

Each dictionary gets two tables: `{dict}_entries` for word senses, `{dict}_examples` for example sentences (1:N from entries).

### `collins_entries` / `oxford_entries`

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | INTEGER PK | — | Auto-increment row ID |
| `word` | TEXT | NOT NULL | Headword, trimmed. e.g. `"abject"`, `"water"`, `"went"` |
| `pos` | TEXT | NULLABLE | Part of speech, trimmed. Collins: `"N-COUNT"`, `"ADJ-GRADED"`, `"VERB"`; Oxford: `"noun [U]"`, `"verb [I]"`, `"verb [T]"`, `"IDM phrase"` for idioms. **NULL for pure cross-reference entries** |
| `cn_definition` | TEXT | NULLABLE | Chinese definition for this sense, trimmed. For xref entries: the descriptive text (e.g. `"past tense of go"`, `"（mouse 的复数）"`) |
| `cross_ref` | TEXT | NULLABLE | Target word when this entry is a pure cross-reference. Oxford: extracted from `<a>` inside `xr-g > xr`. Collins: extracted from `<a class="see">` or caption text. **NULL for regular entries** |
| `sense_order` | INTEGER | NOT NULL DEFAULT 1 | Ordinal within a (word, pos) group, starting from 1. Xref entries always `1` |
| `pronunciation` | TEXT | NULLABLE | JSON array of IPA strings, e.g. `["rɪˈfjuːz"]` for single-region, `["ˈrekɔːd","ˈrekərd"]` for UK+US variants. NULL when no pronunciation found. POS-based differentiation is handled by the entry's `pos` column — entries for different POS of the same word may have different pronunciations |
| `extra_notes` | TEXT | NULLABLE | JSON array of `{"type": "...", "en": "...", "cn": "..."}` objects. Collins: extracted from `<figure class="note type-*">` elements (usage notes, regional variants, derived forms, quotations, etc.). Oxford: reserved for future use. NULL when no extra notes |

**Constraints & Indexes:**
- `UNIQUE (word, pos, sense_order)` — composite unique covering the primary lookup pattern
- `CREATE INDEX idx_{dict}_entries_word ON {dict}_entries(word)` — for prefix/equality lookup by word alone
- `CREATE INDEX idx_{dict}_entries_cross_ref ON {dict}_entries(cross_ref)` — for reverse-following references ("what links to X?")

**Why (word, pos, sense_order) instead of (word, pos) alone:** The same word+POS pair can have multiple senses (e.g. Oxford "beauty" noun [U] has "美，美丽" and "魅力" as separate `n-g` blocks). `sense_order` disambiguates them.

**Cross-reference handling:** ~14,700 Oxford entries and ~10,100 Collins entries are pure cross-references (e.g. "went" → "go", "mice" → "mouse"). These have `pos = NULL`, `cn_definition` = the descriptive label, `cross_ref` = the canonical word, and no rows in the examples table. Entries that have BOTH definitions AND xref labels (e.g. Oxford "better" labeled "comparative of good" but with full definitions) are treated as regular entries (`cross_ref = NULL`).

### `collins_examples` / `oxford_examples`

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| `id` | INTEGER PK | — | Auto-increment row ID |
| `entry_id` | INTEGER | NOT NULL | FK referencing `{dict}_entries(id)` |
| `en_example` | TEXT | NULLABLE | English example sentence, trimmed. NULL if sense has no example |
| `cn_example` | TEXT | NULLABLE | Chinese translation of the example, trimmed. NULL when `en_example` is NULL |
| `example_order` | INTEGER | NOT NULL DEFAULT 1 | Ordinal within the parent entry |

**Constraints & Indexes:**
- `FOREIGN KEY (entry_id) REFERENCES {dict}_entries(id) ON DELETE CASCADE`
- `CREATE INDEX idx_{dict}_examples_entry ON {dict}_examples(entry_id)` — for JOIN queries

### FTS5 (Full-Text Search)

For reverse lookup (Chinese → English) and fuzzy search:

```sql
CREATE VIRTUAL TABLE entries_fts USING fts5(
    source,         -- 'collins' or 'oxford'
    word,
    cn_definition,
    en_example,
    cn_example,
    tokenize='unicode61'
);
```

### Synonyms Table

```sql
CREATE TABLE IF NOT EXISTS synonyms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL CHECK(source IN ('collins', 'oxford')),
    entry_id INTEGER NOT NULL,
    synonym_word TEXT NOT NULL
);
CREATE INDEX idx_synonyms_entry ON synonyms(source, entry_id);
CREATE INDEX idx_synonyms_word ON synonyms(synonym_word);
```

Collins: extracted from `<div class="synonym">` blocks. Oxford: extracted from `.xr-g` elements containing `.symbols-synsym` (SYN) markers or `.z_xr "synonyms at"` cross-references.

### Antonyms Table

```sql
CREATE TABLE IF NOT EXISTS antonyms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL CHECK(source IN ('collins', 'oxford')),
    entry_id INTEGER NOT NULL,
    antonym_word TEXT NOT NULL
);
CREATE INDEX idx_antonyms_entry ON antonyms(source, entry_id);
CREATE INDEX idx_antonyms_word ON antonyms(antonym_word);
```

Primarily from Oxford `.xr-g` elements containing `.symbols-oppsym` (OPP) markers.

### Why Separate Tables Per Dictionary

- Collins uses COBUILD grammar codes, Oxford uses traditional labels with grammar tags
- HTML structures differ significantly — separate extraction logic per dictionary is cleaner
- Querying can target one dictionary without a WHERE filter on every query
- Identical column layouts make UNION queries trivial when querying both

## Extraction Logic

Based on verified HTML structures from actual tabfile inspection.

### Collins

HTML structure: `div.word_entry` → `div.collins_en_cn` blocks, each containing one sense.

#### Regular entry extraction

1. **Word**: `span.word_key` text, fallback to `<h1>` text
2. **POS**: `span.st` text (COBUILD codes like `N-COUNT`, `ADJ-GRADED`, `VERB`). Use `.//text()` to handle inner elements
3. **cn_definition**: `span.def_cn` text within each `collins_en_cn` div
4. **Examples**: Each `<li>` inside `collins_en_cn` → first `<p>` is `en_example`, second `<p>` is `cn_example`. Use `.//text()` on `<p>` to handle `<dfn>` tags wrapping pronunciation words. **Exclude `<li>` inside `<figure class="note">` elements** — those are extra_notes, not examples.
5. **extra_notes**: Extracted from `<figure class="note type-*">` elements via `_extract_notes_from_figures()`, **excluding `type-drv`**. Each figure produces one note combining inline definition text with any `<li>` examples. Quotation notes (`type-quotation`) are parsed from individual `.cit` elements (each containing `blockquote > p` or `span.quote` for quote text and `cite` for attribution), with quotes joined by double newlines. `<br>` tags in `span.quote` are converted to newlines. Orphan figures outside `.collins_en_cn` are attached to the first entry with a definition. Stored as JSON array of `{"type": "<type>", "en": "...", "cn": "..."}`.
6. **Derived forms**: `<figure class="note type-drv">` elements are handled by `_extract_drv_entries()` → standalone entries with `pos='DRV'`, word from `<b>` tag, examples from `<li><p>` pairs. Same-pronunciation as parent entry.
7. **Synonyms**: `<div class="synonym">` blocks → `_extract_collins_synonyms()` extracts each `<span class="form">` (link text or plain text). Stored in `synonyms` table with `source='collins'`.

#### Oxford synonym & antonym extraction

`.xr-g` cross-reference elements are parsed by `_extract_oxford_xrefs()` per entry container (`n-g`, `p-g`, `h-g`, `sense-g`):
- `.symbols-oppsym` (OPP) markers → antonyms
- `.symbols-synsym` (SYN) markers → synonyms (e.g. "radical" sense 1 → "far-reaching")
- `.z_xr` elements containing "synonyms at" → synonym cross-references

Stored in `synonyms` and `antonyms` tables with `source='oxford'`, one entry per sense (not duplicated across senses).
6. **sense_order**: Enumerate `collins_en_cn` divs within a (word, pos) group
7. **pronunciation**: `_extract_collins_pronunciation()` — `span.pron` at `word_entry` level → `span.pron.type_uk` and `span.pron.type_us`. IPA text has HTML markup (`<u>`, `<sup>`) stripped by `_clean_ipa()`. Applied to all entries for the word in `parse_tabfile()`. Stored as JSON array.

#### Cross-reference entry detection

A Collins entry is a pure xref when:
- No `span.st` (POS tag) present
- No `<li>` examples
- Has `<a class="see" href="bword://...">` OR caption text matches "past tense of", "plural of", "is the ... of", etc.

Extraction for xref:
1. **cross_ref**: From `<a class="see">` text (first match), or parsed from caption pattern ("X is the past tense of **Y**")
2. **cn_definition**: `div.caption` text (the full description)
3. **pos**: NULL
4. **sense_order**: 1

### Oxford

HTML structure: `span.entry` → `span.p-g` (POS groups) → `span.n-g` (sense groups).

#### Regular entry extraction

1. **Word**: `span.h` text, fallback to `<h1>` text
2. **POS**: Depends on pattern:
   - Patterns 1/3: Per `p-g` block: `span.pos` text + `span.gr` text (if present), joined with a space.
   - Pattern 2: Per `n-g`: `span.pos` from `top-g > block-g > pos-g` + `.gr` spans inside `n-g`. For run-on entries (e.g. "radically"): fallback to `.top-g > .pos-g .pos`. Handled by `_oxford_pos_for_ng()`.
   - Pattern 4: `span.pos` from `top-g > block-g > pos-g` + `.gr` spans in `top-g` or `h-g`. Handled by `_oxford_make_pos()`.
   - Pattern 4b (idioms): POS is `"IDM <phrase>"` from `.id` text inside `.id-g`.
3. **cn_definition**: `span.oalecd8e_chn` within `span.def-g` under each `n-g` (Patterns 1/2), or directly under `h-g`/`p-g` (Patterns 3/4), or under `ids-g > id-g > sense-g > def-g` (Pattern 4b)
4. **Examples**: Each `span.x-g` within an `n-g` or directly under `h-g`/`p-g` → `span.x.oalecd8e_switch_lang` is `en_example`, the sibling `span.oalecd8e_chn` is `cn_example`
5. **sense_order**: Enumerate sense blocks within each `p-g` (Patterns 1/3) or `h-g` (Patterns 2/4)
6. **pronunciation**: `_extract_oxford_pronunciation()` — `span.ei-g` blocks containing `span.phon-gb` (UK) and `span.phon-usgb`/`span.phon-us` (US). Two placement patterns: (a) word-level in `top-g > ei-g` (e.g. "refuse" verb/noun are separate entries), (b) per-POS inside `p-g > ei-g` (e.g. "record" noun `p-g` has `ˈrekɔːd`, verb `p-g` has `rɪˈkɔːd`). Checks the POS-group container first, falls back to `top-g`. For Pattern 4, checks `h-g` first, falls back to `top-g`.

#### Cross-reference entry detection

An Oxford entry is a pure xref when:
- Has `span.xr-g` (in `span.sense-g`) AND NO `span.p-g` blocks AND NO `span.n-g` blocks
- OR has `.entry > span.derived` with a parent-word link and no `.p-g`, `.n-g`, `.def-g`, `.ids-g` (derived-form redirects like "abasement" → "abase")

Extraction for xref:
1. **cross_ref**: For `.xr-g` entries: `<a>` text inside `xr-g > xr` (e.g. `<a href="bword://go">go</a>` → `"go"`). For `.derived` entries: `<a>` text inside `.derived`.
2. **cn_definition**: For `.xr-g` entries: full text of `span.xr-g` (e.g. "past tense of go"). For `.derived` entries: combined "See 见词条" text from `.de_e` and `.de_c` spans.
3. **pos**: NULL
4. **sense_order**: 1

### Edge Cases Summary

| Case | Handling |
|------|----------|
| Empty `p-g` blocks in Oxford (no pos, no n-g) | Skip the block entirely |
| Oxford Pattern 4: `def-g` directly under `h-g` (no `p-g`/`n-g`) | `_oxford_parse_hg_direct()` handles — POS from `top-g > block-g > pos`, grammar from `top-g > .gr`. Examples from sibling `.x-g` in `h-g` |
| Oxford Pattern 4b: `ids-g` idioms under `h-g` | Each `id-g > sense-g` becomes a separate entry with `IDM <phrase>` POS |
| Oxford derived-form xref (`.entry > .derived`, e.g. "abasement") | Treat as cross-reference — `cn_definition` from `.de_e`/`.de_c`, `cross_ref` from `<a>` link |
| Oxford run-on entries with `.derived` but also `.n-g` content (e.g. "radically") | Treated as regular entries (Pattern 2). POS fallback from `.top-g > .pos-g .pos` when `.block-g` absent. Entry passes filter if it has examples even without a definition. |
| Oxford modal verbs (`must`) — `span.pos` outside `p-g` | Check `block-g > pos-g > pos` at entry level, not just inside `p-g` |
| Collins `<dfn>` tags in example `<p>` | Use `.//text()` not `.text` to collect all text nodes |
| Collins `<figure class="note type-*">` elements | Exclude from examples; extract as `extra_notes` JSON. Each figure produces one note combining inline text and `<li>` examples. Quotation notes parsed from `.cit > blockquote p` or `.cit > span.quote` + `.cit > cite` |
| Collins `<figure class="note type-drv">` derived forms | Create standalone entries with `pos='DRV'`, word from `<b>` tag, examples from `<li><p>` pairs. Deduplicated within parent by `seen_drv_words` set |
| Collins `<div class="synonym">` blocks | Extract `<span class="form">` children (both `<a>` links and plain text) into `synonyms` table keyed to entry ID |
| Collins orphan `figure.note` outside `.collins_en_cn` (quotations) | Attach to first entry with a definition |
| Collins level1–level5 frequency markers | Ignore |
| Pure cross-reference entries | `pos=NULL`, `cn_definition`=description, `cross_ref`=target word |
| Hybrid entries (xref label + own definitions, e.g. Oxford "better") | Treat as regular entries; ignore the xref label |
| Collins "see also" references in phrases | Extract first `bword://` target from `<a class="see">` |
| **All text fields** | **MUST be `.strip()`'d before INSERT** |

## Implementation Steps

### Step 1: `extract/schema.sql`

DDL for all 4 tables (`collins_entries`, `collins_examples`, `oxford_entries`, `oxford_examples`) + `entries_fts` virtual table. Use `IF NOT EXISTS` for idempotency.

### Step 2: `extract/build_db.py`

Single script with these responsibilities:

1. Read `schema.sql` from the same directory, execute against `ecd.db` (in project root)
2. Extract each `.dictionary` bundle to a temp tabfile via `pyglossary`
3. Parse each tabfile line: split on first `\t`, parse HTML with `lxml.html.fromstring()`
4. Detect entry type (regular vs. xref)
5. Extract fields per the rules above
6. Batch INSERT into tables (use `executemany` for performance; consider wrapping in a single transaction)
7. Populate FTS5 from entries + examples JOIN
8. Clean up temp tabfiles

Arguments:
- `--collins-path` / `--oxford-path` — override default dictionary paths
- `--output` — override output db path (default: `../ecd.db` from the extract dir)
- `--only` — process only one dictionary (`collins` or `oxford`)

### Step 3: `ecd` CLI

Python script in project root:
- `ecd <word>` — query both dictionaries, display formatted results with pronunciation, synonyms, and examples
- `ecd -s collins <word>` / `ecd -s oxford <word>` — single dictionary
- `ecd <chinese>` — FTS5 reverse lookup
- For xref entries: display "→ see `<cross_ref>`" and optionally follow the ref
- **Pronunciation display**: Parsed from JSON array, displayed inline on the word line as ` /IPA1 | IPA2/` (purple IPA, dimmed `/` delimiters).
- **Synonym/Antonym display**: Entries show `同义: synonym1, ...` and `反义: antonym1, ...` (dim commas).
- **Extra notes display**: Label (`[用法]`, `[名言]`, `[释义补充]`, `[注]`) on its own line, content on following lines without indentation. Multi-line notes (quotations, usage notes with examples) have each segment on its own line. Separate quotes are joined with a blank line.
- **Lookup history**: Stored in `~/.ecd_lookup.db` (separate from stateless `ecd.db`). `record_lookup()` upserts the queried word with count + timestamp on any result-bearing query (exact match, prefix match with single result, Chinese FTS5 hit). "Did you mean" suggestions and empty results are not recorded. History survives `ecd.db` rebuilds.
- **Interactive mode**: Sets terminal title to "ecd". Commands: `.add [word]` (add word to flashcard deck with dictionary lookup preview), `.auto-add [on|off]` (toggle auto-adding looked-up words), `.review` (SM-2 spaced repetition review with Again/Hard/Good/Easy rating), `.deck` (deck statistics), `.reset` (clear all flashcard data), `.syn [word]` (show synonyms from both dictionaries), `.ant [word]` (show antonyms), `.exit`/`.quit`/`.q` to exit.
- **Flashcard deck**: SM-2 algorithm with ease factor, interval days, repetition count. Cards stored in `~/.ecd_lookup.db` `flashcards` table. Flashcard data is independent of dictionary data — only stores word + SM-2 metadata, looks up `ecd.db` for display during review.

## Verification

```bash
# Build the database
cd extract && /tmp/dict_venv/bin/python build_db.py

# Row counts
sqlite3 ../ecd.db "SELECT 'collins_entries', COUNT(*) FROM collins_entries"
# → expect ~81k (includes ~3.5k DRV entries)
sqlite3 ../ecd.db "SELECT 'oxford_entries', COUNT(*) FROM oxford_entries"
# → expect ~88k

# Xref counts
sqlite3 ../ecd.db "SELECT COUNT(*) FROM collins_entries WHERE cross_ref IS NOT NULL AND cross_ref != ''"
# → expect ~230
sqlite3 ../ecd.db "SELECT COUNT(*) FROM oxford_entries WHERE cross_ref IS NOT NULL AND cross_ref != ''"
# → expect ~3,300 (includes .derived redirects)

# DRV entries (Collins derived forms)
sqlite3 ../ecd.db "SELECT COUNT(*) FROM collins_entries WHERE pos = 'DRV'"
# → expect ~3,500

# Synonyms (both dictionaries)
sqlite3 ../ecd.db "SELECT source, COUNT(*) FROM synonyms GROUP BY source"
# → expect collins ~85k, oxford ~7,200

# Antonyms
sqlite3 ../ecd.db "SELECT source, COUNT(*) FROM antonyms GROUP BY source"
# → expect oxford ~1,500

# Extra notes
sqlite3 ../ecd.db "SELECT COUNT(*) FROM collins_entries WHERE extra_notes IS NOT NULL"
# → expect ~10,000

# Spot-check regular entries
sqlite3 ../ecd.db "SELECT word, pos, cn_definition FROM collins_entries WHERE word='abject'"
sqlite3 ../ecd.db "SELECT e.word, e.pos, e.cn_definition, x.en_example, x.cn_example FROM oxford_entries e LEFT JOIN oxford_examples x ON e.id=x.entry_id WHERE e.word='beauty'"

# Spot-check Pattern 4 entry (Oxford)
sqlite3 ../ecd.db "SELECT word, pos, cn_definition FROM oxford_entries WHERE word='incantation'"

# Spot-check idiom (Oxford)
sqlite3 ../ecd.db "SELECT word, pos, cn_definition FROM oxford_entries WHERE word='aback'"

# Spot-check extra_notes (Collins)
sqlite3 ../ecd.db "SELECT word, extra_notes FROM collins_entries WHERE word='prefer'"

# Spot-check cross-references
sqlite3 ../ecd.db "SELECT word, cn_definition, cross_ref FROM oxford_entries WHERE word='went'"
# → went | past tense of go | go
sqlite3 ../ecd.db "SELECT word, cn_definition, cross_ref FROM collins_entries WHERE word='mice'"
# → mice | Mice is the plural of mouse.（mouse 的复数）| mouse
sqlite3 ../ecd.db "SELECT word, cn_definition, cross_ref FROM oxford_entries WHERE word='abasement'"
# → abasement | See 见词条 | abase (derived-form xref)

# Spot-check pronunciation
sqlite3 ../ecd.db "SELECT word, pos, pronunciation FROM oxford_entries WHERE word='record'"
# → noun entries show ["ˈrekɔːd","ˈrekərd"]; verb entries show ["rɪˈkɔːd","rɪˈkɔːrd"]
sqlite3 ../ecd.db "SELECT COUNT(*) FROM collins_entries WHERE pronunciation IS NOT NULL"
# → ~60k entries have pronunciation

# Lookup history (separate DB)
sqlite3 ~/.ecd_lookup.db "SELECT * FROM lookup_history"

# CLI tests
./ecd hello
./ecd 水
./ecd -s oxford beauty
./ecd went
```
