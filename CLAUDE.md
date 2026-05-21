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
extract/.venv/bin/pip install -e extract/
extract/.venv/bin/ecd-build

# Rebuild only one dictionary
extract/.venv/bin/python -m ecd_extract --only oxford

# Run the Go CLI
make build                   # Produces ./ecd-go binary
./ecd-go hello               # CLI mode — one-shot query
./ecd-go                     # TUI mode — interactive interface
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

### Data flow

`ecd.db` (read-only, ~80MB) holds dictionary data. `~/.ecd_lookup.db` (read-write) holds lookup history and SM-2 flashcard state. Both are SQLite.

The old Python CLI (`ecdlib/`, `ecd`) is archived in `cli_deprecated/`.

## File editing

The user's global CLAUDE.md forbids `sed` for editing, `cat -A`/`od`/`xxd` for reading, and `git checkout`/`git stash` to discard changes. When the native edit tool fails, fall back to `Write` for full-file overwrites.
