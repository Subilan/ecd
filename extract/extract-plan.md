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
| `pos` | TEXT | NULLABLE | Part of speech, trimmed. Collins: `"N-COUNT"`, `"ADJ-GRADED"`, `"VERB"`; Oxford: `"noun [U]"`, `"verb [I]"`, `"verb [T]"`. **NULL for pure cross-reference entries** |
| `cn_definition` | TEXT | NULLABLE | Chinese definition for this sense, trimmed. For xref entries: the descriptive text (e.g. `"past tense of go"`, `"（mouse 的复数）"`) |
| `cross_ref` | TEXT | NULLABLE | Target word when this entry is a pure cross-reference. Oxford: extracted from `<a>` inside `xr-g > xr`. Collins: extracted from `<a class="see">` or caption text. **NULL for regular entries** |
| `sense_order` | INTEGER | NOT NULL DEFAULT 1 | Ordinal within a (word, pos) group, starting from 1. Xref entries always `1` |

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
    content='',     -- external content table
    content_rowid='id',
    tokenize='unicode61'
);
```

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
4. **Examples**: Each `<li>` inside `collins_en_cn` → first `<p>` is `en_example`, second `<p>` is `cn_example`. Use `.//text()` on `<p>` to handle `<dfn>` tags wrapping pronunciation words
5. **sense_order**: Enumerate `collins_en_cn` divs within a (word, pos) group

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
2. **POS**: Per `p-g` block: `span.pos` text (use `.//text()` to handle `<a>` tags inside) + `span.gr` text (if present), joined with a space. e.g. `"noun"` + `"[U]"` → `"noun [U]"`
3. **cn_definition**: `span.oalecd8e_chn` within `span.def-g` under each `n-g`
4. **Examples**: Each `span.x-g` within an `n-g` → `span.x.oalecd8e_switch_lang` is `en_example`, the sibling `span.oalecd8e_chn` is `cn_example`
5. **sense_order**: Enumerate `n-g` blocks within each `p-g`

#### Cross-reference entry detection

An Oxford entry is a pure xref when:
- Has `span.xr-g` (in `span.sense-g`)
- Has NO `span.p-g` blocks
- Has NO `span.n-g` blocks

Extraction for xref:
1. **cross_ref**: `<a>` text inside `xr-g > xr` (e.g. `<a href="bword://go">go</a>` → `"go"`)
2. **cn_definition**: Full text of `span.xr-g` (e.g. "past tense of go")
3. **pos**: NULL
4. **sense_order**: 1

### Edge Cases Summary

| Case | Handling |
|------|----------|
| Empty `p-g` blocks in Oxford (no pos, no n-g) | Skip the block entirely |
| Oxford modal verbs (`must`) — `span.pos` outside `p-g` | Check `block-g > pos-g > pos` at entry level, not just inside `p-g` |
| Collins `<dfn>` tags in example `<p>` | Use `.//text()` not `.text` to collect all text nodes |
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

Shell or Python script in project root:
- `ecd <word>` — query both dictionaries, display formatted results
- `ecd -s collins <word>` / `ecd -s oxford <word>` — single dictionary
- `ecd <chinese>` — FTS5 reverse lookup
- For xref entries: display "→ see `<cross_ref>`" and optionally follow the ref

## Verification

```bash
# Build the database
cd extract && /tmp/dict_venv/bin/python build_db.py

# Row counts
sqlite3 ../ecd.db "SELECT 'collins_entries', COUNT(*) FROM collins_entries"
# → expect ~45k (reg) + ~10k (xref) combined
sqlite3 ../ecd.db "SELECT 'oxford_entries', COUNT(*) FROM oxford_entries"
# → expect ~57k

# Xref counts
sqlite3 ../ecd.db "SELECT COUNT(*) FROM collins_entries WHERE cross_ref IS NOT NULL"
# → expect ~10,100
sqlite3 ../ecd.db "SELECT COUNT(*) FROM oxford_entries WHERE cross_ref IS NOT NULL"
# → expect ~14,700

# Spot-check regular entries
sqlite3 ../ecd.db "SELECT word, pos, cn_definition FROM collins_entries WHERE word='abject'"
sqlite3 ../ecd.db "SELECT e.word, e.pos, e.cn_definition, x.en_example, x.cn_example FROM oxford_entries e LEFT JOIN oxford_examples x ON e.id=x.entry_id WHERE e.word='beauty'"

# Spot-check cross-references
sqlite3 ../ecd.db "SELECT word, cn_definition, cross_ref FROM oxford_entries WHERE word='went'"
# → went | past tense of go | go
sqlite3 ../ecd.db "SELECT word, cn_definition, cross_ref FROM collins_entries WHERE word='mice'"
# → mice | Mice is the plural of mouse.（mouse 的复数）| mouse

# CLI tests
./ecd hello
./ecd 你好
./ecd -s oxford beauty
./ecd went
```
