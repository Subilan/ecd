-- Collins entries: word senses
CREATE TABLE IF NOT EXISTS collins_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    word TEXT NOT NULL,
    pos TEXT NOT NULL DEFAULT '',
    cn_definition TEXT,
    cross_ref TEXT,
    sense_order INTEGER NOT NULL DEFAULT 1,
    pronunciation TEXT,
    extra_notes TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_collins_entries_word_pos_sense
    ON collins_entries(word, pos, sense_order);
CREATE INDEX IF NOT EXISTS idx_collins_entries_word ON collins_entries(word);
CREATE INDEX IF NOT EXISTS idx_collins_entries_cross_ref ON collins_entries(cross_ref);

-- Collins examples: 1:N from collins_entries
CREATE TABLE IF NOT EXISTS collins_examples (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES collins_entries(id) ON DELETE CASCADE,
    en_example TEXT,
    cn_example TEXT,
    example_order INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_collins_examples_entry ON collins_examples(entry_id);

-- Oxford entries: word senses
CREATE TABLE IF NOT EXISTS oxford_entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    word TEXT NOT NULL,
    pos TEXT NOT NULL DEFAULT '',
    cn_definition TEXT,
    cross_ref TEXT,
    sense_order INTEGER NOT NULL DEFAULT 1,
    pronunciation TEXT,
    extra_notes TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_oxford_entries_word_pos_sense
    ON oxford_entries(word, pos, sense_order);
CREATE INDEX IF NOT EXISTS idx_oxford_entries_word ON oxford_entries(word);
CREATE INDEX IF NOT EXISTS idx_oxford_entries_cross_ref ON oxford_entries(cross_ref);

-- Oxford examples: 1:N from oxford_entries
CREATE TABLE IF NOT EXISTS oxford_examples (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES oxford_entries(id) ON DELETE CASCADE,
    en_example TEXT,
    cn_example TEXT,
    example_order INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_oxford_examples_entry ON oxford_examples(entry_id);

-- FTS5 for Chinese reverse lookup and fuzzy search
CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
    source,
    word,
    cn_definition,
    en_example,
    cn_example,
    tokenize='unicode61'
);

-- Synonyms from Collins entries
CREATE TABLE IF NOT EXISTS synonyms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id INTEGER NOT NULL REFERENCES collins_entries(id) ON DELETE CASCADE,
    synonym_word TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_synonyms_entry ON synonyms(entry_id);
CREATE INDEX IF NOT EXISTS idx_synonyms_word ON synonyms(synonym_word);
