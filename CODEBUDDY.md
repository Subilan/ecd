# CODEBUDDY.md

This file provides guidance to CodeBuddy Code when working with code in this repository.

## Commands

### First-time setup

```bash
python3 -m venv extract/.venv
extract/.venv/bin/pip install -r extract/requirements.txt
extract/.venv/bin/pip install -e extract/
```

### Build

```bash
make build                     # Produces ./ecd binary (Go 1.25+)
cd cli && go run .             # Dev: run from source
```

### Test & Lint

```bash
make test                      # Runs: go vet ./... && go test ./... (in cli/)
cd cli && go vet ./...         # Vet only
cd cli && go test ./...        # Tests only
cd cli && go test ./... -run TestFoo  # Single test
```

### Dictionary database (extract)

```bash
extract/.venv/bin/ecd-build                     # Full build (both dictionaries, ~30s)
extract/.venv/bin/python -m ecd_extract --only oxford  # Rebuild one dictionary only
```

### Running the app

```bash
./ecd hello                    # CLI: exact match
./ecd surprisingl              # CLI: prefix match
./ecd rondevus                 # CLI: fuzzy match
./ecd 全面的                   # CLI: Chinese reverse lookup (FTS5)
./ecd -r                       # CLI: random word
./ecd -s oxford beauty         # CLI: filter by dictionary source
./ecd -i suffice               # CLI: Oxford idioms
./ecd                          # TUI: interactive mode
./ecd --ai                     # TUI: enter AI assistant directly
./ecd --no-color hello         # CLI: disable ANSI colors
./ecd --config /path/to/ecd.toml hello  # Custom config path
echo hello | ./ecd             # CLI: piped stdin
```

## Architecture

This project is an English-Chinese dictionary CLI/TUI tool. It extracts two macOS Apple Dictionary bundles (Collins COBUILD English-Chinese and Oxford Advanced Learner's 8th Ed) into a SQLite database, then serves queries through a Go CLI/TUI.

### Top-level structure

| Directory | Purpose |
|-----------|---------|
| `extract/` | Python pipeline: extracts `.dictionary` bundles → tabfiles → SQLite (`ecd.db`) |
| `cli/` | Go binary: CLI mode (one-shot queries) and TUI mode (Bubble Tea interactive interface) |
| `cli/docs/` | Design docs for CLI architecture, TUI, search, flashcards, i18n |

### Extraction pipeline (`extract/`)

Python package `ecd_extract` uses `pyglossary` to convert macOS `.dictionary` bundles to tabfiles, then parses per-dictionary HTML into SQLite via `lxml`/`cssselect`.

Key files:
- `ecd_extract/cli.py` — CLI entry point (`python -m ecd_extract`)
- `ecd_extract/build.py` — orchestrates full database build
- `ecd_extract/collins.py` — Collins COBUILD HTML parser
- `ecd_extract/oxford.py` — Oxford Advanced Learner's HTML parser
- `schema.sql` — SQLite DDL (entries, examples, FTS5, synonyms, antonyms, idioms)

Full details: `extract/docs/README.md`

### Go CLI (`cli/`)

Single binary, zero runtime deps (pure Go SQLite via `modernc.org/sqlite`).

```
main.go → flag parse → CLI mode (cli/)  → one-shot query + output + exit
                     → TUI mode (tui/)  → Bubble Tea interactive interface
```

Mode selection: CLI mode when args are provided or stdin is piped; TUI mode when no args and stdin is a TTY.

**Internal packages:**

| Package | Responsibility |
|---------|---------------|
| `config/` | DB paths (`ecd.db`, `~/.ecd_lookup.db`), CJK detection, driver init, TOML config |
| `dict/` | Read-only dictionary DB queries: exact, prefix, fuzzy, Chinese FTS5, idioms, random |
| `search/` | 4-tier search dispatch (exact → prefix → fuzzy → Chinese FTS5), synonym/antonym lookup |
| `history/` | Read-write user DB: lookup history and SM-2 flashcard state (`~/.ecd_lookup.db`) |
| `sm2/` | Pure math SM-2 spaced repetition algorithm (no external deps) |
| `cli/` | ANSI-colored terminal output for CLI mode |
| `tui/` | Bubble Tea interactive interface (search view, detail, review, deck, help, AI) |
| `i18n/` | Chinese/English string tables |
| `render/` | ANSI-safe text wrapping for viewport content |
| `ai/` | Optional AI features: OpenAI-compatible API client, cache, prompts, TUI sub-model |
| `repl/` | Shared REPL base types, messages, and styles |

**Key dependencies (go.mod):** Bubble Tea + Bubbles + Lipgloss (TUI framework), `modernc.org/sqlite` (pure Go SQLite), `go-toml` v2 (config), `go-openai` (AI client).

### Data flow

Two SQLite databases:

```
ecd.db (~80MB, read-only)                ~/.ecd_lookup.db (read-write)
├── collins_entries / collins_examples   ├── lookup_history
├── oxford_entries / oxford_examples     └── flashcards (SM-2 state)
├── entries_fts (FTS5)
├── synonyms / antonyms
└── oxford_idioms
       ↑ dict.DB (read-only)                ↑ history.DB (read-write)
       └── search.Context ──────────────────┘
              └── tui / cli (presentation)
```

The config file (`ecd.toml`) sets `db_path` (default: `./ecd.db`) and `lookup_db` (default: `~/.ecd_lookup.db`). The `--config` flag can override the path.

### Search dispatch (4 tiers)

For English queries: exact match (NOCASE) → prefix match (NOCASE) → fuzzy match (bigram similarity ≥0.75) → falls through to Chinese FTS5.

For Chinese queries: FTS5 MATCH directly, ranked by relevance, deduplicated by `(source, word, cn_definition)`.

Multi-word prefix results become "did you mean" suggestions. Single-word prefix results auto-resolve to that word's entries.

### AI module (`cli/internal/ai/`)

Optional OpenAI-compatible API integration. **Files:**
- `model.go` — `AIModel` Bubble Tea sub-model (text input, result viewport, spinner, command dispatch)
- `init.go` — `InitModel` config form (API key, Base URL, model radio list)
- `client.go` — OpenAI client wrapper with retry (3 retries, exponential backoff 1s→2s→4s)
- `cache.go` — File cache at `~/.ecd_ai_cache/`, keyed by SHA-256 of whitespace-normalized command
- `prompts.go` — System/user prompt builders per command
- `commands.go` — Command parsing and routing
- `format.go` — AI response formatting

**Config** (`ecd.toml` `[ai]` section):
```toml
[ai]
api_key = ""
base_url = ""
model = ""
cache_enabled = true
```

Without an API key, AI commands show a prompt to run `/init` but don't function. In `/init`, model list is fetched from the API (10s timeout) — on success, radio list and custom input appear inline.

**AI commands:** `/diff`, `/ant`, `/syn`, `/phr`, `/example`, `/explain`, `/init`, `/cache`, `/back`. Append `!` to bypass cache (e.g., `/syn! happy many`).

## Bubble Tea patterns and pitfalls

### WindowSizeMsg and late-created sub-models

Bubble Tea sends `tea.WindowSizeMsg` once at startup. The main `Model` distributes it to all sub-models created in `NewModel()`. If a sub-model is created later (e.g., `/init` form on state transition), it will NEVER receive this message automatically.

**Rule:** Any sub-model created on-the-fly whose `View()` depends on width/height or a ready flag must receive explicit dimensions:

```go
m.sub = NewSubModel(...)
m.sub, _ = m.sub.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
```

Sub-models affected:
- `aiInit` (InitModel) — created in `EnterInitMsg` handler, needs explicit `WindowSizeMsg`
- `reviewModel` — recreated in `startReview()`, but works without it (uses viewport defaults, no ready guard)

### render.WrapContent is mandatory

All viewport content MUST be wrapped through `render.WrapContent(text, width)` before `viewport.SetContent()`. Without it, long lines overflow horizontally. Also re-wrap on `WindowSizeMsg` when existing content is displayed.

```go
m.viewport.SetContent(render.WrapContent(content, m.viewport.Width))
```

### Use textinput.Model for editable fields

When building a form with text input fields, use Bubble Tea's `textinput.Model`. Never manually parse `tea.KeyMsg.String()` character-by-character — it can't handle paste, arrow keys, or non-ASCII input reliably.
