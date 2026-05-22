# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Terminology

- **repl** — the dictionary search TUI mode (StateSearch / StateEntryDetail)
- **ai repl** — the AI assistant TUI mode (StateAI)

## Setup (first time)

```bash
python3 -m venv extract/.venv
extract/.venv/bin/pip install -r extract/requirements.txt
```

## Build & Run

```bash
# Build the database (takes ~30s)
extract/.venv/bin/pip install -e extract/
extract/.venv/bin/ecd-build

# Rebuild only one dictionary
extract/.venv/bin/python -m ecd_extract --only oxford

# Run the Go CLI
make build                   # Produces ./ecd binary
./ecd hello                  # CLI mode — one-shot query
./ecd                        # TUI mode — interactive interface
./ecd --ai                   # TUI mode — enter AI assistant directly
./ecd -i suffice             # Oxford idioms
cd cli && go run .           # Dev: run from source
```

The `extract/.venv/` and `ecd.db` (~80 MB) are git-ignored.

## Architecture

The project extracts two macOS Apple Dictionary `.dictionary` bundles (Collins COBUILD English-Chinese and Oxford Advanced Learner's 8th Ed) into a SQLite database, then serves queries through a Go CLI/TUI.

### Extraction → `extract/`

Python (`ecd_extract` package): uses `pyglossary` to convert `.dictionary` bundles to tabfiles, then parses per-dictionary HTML into a SQLite database. Full documentation: [extract/docs/README.md](extract/docs/README.md).

### Go CLI → `cli/`

Single binary (pure Go SQLite via `modernc.org/sqlite`). Two modes:
- **CLI mode** (args/piped stdin): one-shot query with ANSI-colored output
- **TUI mode** (no args + TTY): Bubble Tea interactive interface with `/` commands, SM-2 flashcard reviews, and history

Full documentation: [cli/docs/README.md](cli/docs/README.md).

### AI module → `cli/internal/ai/`

Optional AI-powered features using an OpenAI-compatible API. Gated by configuration — without an API key in `ecd.toml`, AI commands show a helpful prompt but don't work.

**Files:**
- `model.go` — `AIModel` Bubble Tea sub-model: text input, result viewport, spinner, command dispatch
- `init.go` — `InitModel` config form: two `textinput.Model` fields (API key, Base URL), inline model radio list fetched from API, custom model input
- `client.go` — OpenAI client wrapper with retry (3 retries, exponential backoff 1s→2s→4s), only retries on connection/timeout/rate-limit errors
- `cache.go` — File-based cache in `~/.ecd_ai_cache/`, keyed by SHA-256 hash of whitespace-normalized command string
- `prompts.go` — System/user prompt builders for each command

**Config** (`ecd.toml` `[ai]` section):
```toml
[ai]
api_key = ""
base_url = ""
model = ""
cache_enabled = true
```

**TUI states:** `StateAI` (REPL), `StateAIInit` (config form). Entered via `/ai` slash command or `--ai` flag.

**AI mode commands** (all `/` prefixed):
| Command | Description |
|---|---|
| `/back` | Return to dictionary search mode |
| `/init` | Open config form (2 textinputs + model radio list, ↑↓ to navigate, Esc to exit) |
| `/cache on\|off` | Toggle AI response cache |
| `/diff <w1> <w2> ...` | Explain differences between up to 5 words |
| `/ant <word> <one\|some\|many>` | Generate antonyms (structured JSON output) |
| `/syn <word> <one\|some\|many>` | Generate synonyms |
| `/phr <word> <one\|some\|many>` | Generate phrases |
| `/example <word>` | Generate example sentences |
| `/explain <word>` | Detailed word explanation |
| `!` suffix on any command | Bypass cache (e.g. `/syn! happy many`) |

**Key patterns:**
- AI mode always accessible regardless of config — unconfigured users see `[run /init to configure]` in footer
- `/init` form uses native `textinput.Model` for reliable paste/keyboard handling
- Model row in `/init` has no textinput — navigate to it to auto-fetch the model list from the API (10s timeout). On success, radio list and custom input appear inline below. Space=select model, ↑↓ navigate through models and custom row, Enter=save & test
- Spinner (`⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏`) shown during API calls via `tea.Tick`
- Cache bypass with `!` is a no-op when `cache_enabled` is OFF

### Shared render utilities → `cli/internal/render/`

`render.WrapContent(text, width)` wraps long lines to fit a given width while preserving ANSI escape codes. **All viewport content must be wrapped through this function before calling `viewport.SetContent()`** — without it, long lines overflow horizontally. Also re-wrap on `WindowSizeMsg` when existing content is displayed.

```go
m.viewport.SetContent(render.WrapContent(content, m.viewport.Width))
```

### Data flow

`ecd.db` (read-only, ~80MB) holds dictionary data. `~/.ecd_lookup.db` (read-write) holds lookup history and SM-2 flashcard state. Both are SQLite.

## Bubble Tea patterns

### WindowSizeMsg and late-created sub-models

Bubble Tea sends `tea.WindowSizeMsg` once during startup. The main `Model` distributes it to all sub-models created in `NewModel()`. If a sub-model is created later (e.g. on state transition like `/init`), it will NEVER receive this message.

**Rule:** Any sub-model whose `View()` guards on a ready flag or width/height check must have its dimensions set explicitly when created on-the-fly:

```go
// When creating a model after startup:
m.sub = NewSubModel(...)
m.sub, _ = m.sub.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
```

Current sub-models and their creation point:
- `search`, `detail`, `review`, `deck`, `help`, `ai` (AIModel) — created in `NewModel()`, receive `WindowSizeMsg` via distribution loop ✅
- `aiInit` (InitModel) — created on-the-fly in `EnterInitMsg` handler, needs explicit `WindowSizeMsg` ✅
- `reviewModel` — recreated in `startReview()`, but uses viewport defaults (no ready guard), so it works without explicit dimensions ✅

### Prefer textinput.Model for editable fields

When building a form with text input fields, always use Bubble Tea's `textinput.Model`. Never manually parse `tea.KeyMsg.String()` character-by-character — it can't reliably handle paste, arrow keys, or non-ASCII input. Use `textinput.Model` and call its `Update()`/`View()` methods.

## File editing

The user's global CLAUDE.md forbids `sed` for editing, `cat -A`/`od`/`xxd` for reading, and `git checkout`/`git stash` to discard changes. When the native edit tool fails, fall back to `Write` for full-file overwrites.
