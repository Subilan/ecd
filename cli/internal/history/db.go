package history

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Subilan/ecd/internal/config"
)

// Flashcard represents a flashcard record from the history DB.
type Flashcard struct {
	Word         string
	Created      string
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	NextReview   string
	LastReview   *string
	TotalReviews int
	TotalCorrect int
}

// DeckStats holds aggregate flashcard deck statistics.
type DeckStats struct {
	Total     int
	Due       int
	New       int
	Mature    int
	Leeches   int
	AvgEase   float64
	NextDelta string
	Overdue   bool
}

// DB wraps the history database connection.
type DB struct {
	*sql.DB
}

// OpenHistoryDB opens (and creates if needed) the history database.
func OpenHistoryDB() (*DB, error) {
	if config.HistoryDB == "" {
		return nil, fmt.Errorf("history database path not set")
	}
	db, err := sql.Open("sqlite", config.HistoryDB)
	if err != nil {
		return nil, fmt.Errorf("open history db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite serialization

	if err := ensureSchema(db); err != nil {
		db.Close()
		return nil, err
	}
	return &DB{db}, nil
}

func ensureSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS lookup_history (
			word TEXT NOT NULL PRIMARY KEY,
			count INTEGER NOT NULL DEFAULT 1,
			last_query TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`)
	if err != nil {
		return fmt.Errorf("create lookup_history: %w", err)
	}
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS flashcards (
			word TEXT NOT NULL PRIMARY KEY,
			created TEXT NOT NULL DEFAULT (datetime('now')),
			ease_factor REAL NOT NULL DEFAULT 2.5,
			interval_days INTEGER NOT NULL DEFAULT 0,
			repetitions INTEGER NOT NULL DEFAULT 0,
			next_review TEXT NOT NULL DEFAULT (datetime('now', '+10 minutes')),
			last_review TEXT,
			total_reviews INTEGER NOT NULL DEFAULT 0,
			total_correct INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return fmt.Errorf("create flashcards: %w", err)
	}
	return nil
}

// RecordLookup records a word lookup in the history.
func (h *DB) RecordLookup(word string) error {
	_, err := h.Exec(`
		INSERT INTO lookup_history (word, count, last_query)
		VALUES (?, 1, datetime('now'))
		ON CONFLICT(word) DO UPDATE SET
			count = count + 1,
			last_query = datetime('now')
	`, strings.ToLower(word))
	return err
}

// AddFlashcard adds a word to the flashcard deck. Returns true if newly added.
func (h *DB) AddFlashcard(word string) (bool, error) {
	result, err := h.Exec("INSERT INTO flashcards (word) VALUES (?)", word)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return false, nil
		}
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// DelFlashcard removes a word from the flashcard deck. Returns true if deleted.
func (h *DB) DelFlashcard(word string) (bool, error) {
	result, err := h.Exec("DELETE FROM flashcards WHERE word = ?", word)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

// GetDueCards fetches cards where next_review <= now, up to limit.
func (h *DB) GetDueCards(limit int) ([]Flashcard, error) {
	rows, err := h.Query(`
		SELECT word, ease_factor, interval_days, repetitions, next_review,
		       total_reviews, total_correct
		FROM flashcards
		WHERE next_review <= datetime('now')
		ORDER BY next_review
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cards []Flashcard
	for rows.Next() {
		var c Flashcard
		if err := rows.Scan(&c.Word, &c.EaseFactor, &c.IntervalDays,
			&c.Repetitions, &c.NextReview, &c.TotalReviews, &c.TotalCorrect); err != nil {
			return nil, err
		}
		cards = append(cards, c)
	}
	return cards, nil
}

// GetDeckStats computes aggregate flashcard statistics.
func (h *DB) GetDeckStats() (*DeckStats, error) {
	s := &DeckStats{}

	h.QueryRow("SELECT COUNT(*) FROM flashcards").Scan(&s.Total)
	if s.Total == 0 {
		return s, nil
	}

	h.QueryRow("SELECT COUNT(*) FROM flashcards WHERE next_review <= datetime('now')").Scan(&s.Due)
	h.QueryRow("SELECT COUNT(*) FROM flashcards WHERE repetitions = 0").Scan(&s.New)
	h.QueryRow("SELECT COUNT(*) FROM flashcards WHERE interval_days >= 21").Scan(&s.Mature)
	h.QueryRow("SELECT COUNT(*) FROM flashcards WHERE ease_factor <= 1.3").Scan(&s.Leeches)

	var avgEF sql.NullFloat64
	h.QueryRow("SELECT AVG(ease_factor) FROM flashcards").Scan(&avgEF)
	if avgEF.Valid {
		s.AvgEase = avgEF.Float64
	}

	// Next review info
	var nextReview sql.NullString
	h.QueryRow("SELECT MIN(next_review) FROM flashcards").Scan(&nextReview)
	if nextReview.Valid {
		nextDT, err := time.Parse("2006-01-02 15:04:05", nextReview.String)
		if err == nil {
			now := time.Now()
			delta := nextDT.Sub(now)
			absSecs := int(delta.Abs().Seconds())
			if delta <= 0 {
				s.Overdue = true
			}

			days := absSecs / 86400
			hours := (absSecs % 86400) / 3600
			mins := (absSecs % 3600) / 60

			var parts []string
			if days > 0 {
				if hours > 0 {
					parts = append(parts, fmt.Sprintf("%dd %dh", days, hours))
				} else {
					parts = append(parts, fmt.Sprintf("%dd", days))
				}
			} else if hours > 0 {
				if mins > 0 {
					parts = append(parts, fmt.Sprintf("%dh %dm", hours, mins))
				} else {
					parts = append(parts, fmt.Sprintf("%dh", hours))
				}
			} else {
				parts = append(parts, fmt.Sprintf("%dm", mins))
			}
			s.NextDelta = strings.Join(parts, " ")
		}
	}

	return s, nil
}

// UpdateFlashcard updates the SM-2 fields after a review.
func (h *DB) UpdateFlashcard(word string, easeFactor float64, intervalDays, repetitions int, correct int) error {
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	nextReview := time.Now().Add(time.Duration(intervalDays) * 24 * time.Hour).Format("2006-01-02 15:04:05")
	_, err := h.Exec(`
		UPDATE flashcards
		SET ease_factor = ?, interval_days = ?, repetitions = ?,
		    next_review = ?, last_review = ?,
		    total_reviews = total_reviews + 1,
		    total_correct = total_correct + ?
		WHERE word = ?
	`, easeFactor, intervalDays, repetitions, nextReview, nowStr, correct, word)
	return err
}

// ResetAll removes all flashcard data.
func (h *DB) ResetAll() error {
	_, err := h.Exec("DELETE FROM flashcards")
	return err
}

// GetFlashcardStatuses returns a map of word -> status for a batch of words.
// Status is "leech" when ease_factor <= 1.3, "flashcard" when merely present.
func (h *DB) GetFlashcardStatuses(words []string) map[string]string {
	if len(words) == 0 {
		return nil
	}
	statuses := make(map[string]string, len(words))
	for _, w := range words {
		statuses[w] = ""
	}

	placeholders := make([]string, len(words))
	args := make([]interface{}, len(words))
	for i, w := range words {
		placeholders[i] = "?"
		args[i] = w
	}

	rows, err := h.Query(
		"SELECT word, ease_factor FROM flashcards WHERE word IN ("+
			strings.Join(placeholders, ",")+")",
		args...)
	if err != nil {
		return statuses
	}
	defer rows.Close()

	for rows.Next() {
		var word string
		var ef float64
		if err := rows.Scan(&word, &ef); err != nil {
			continue
		}
		if ef <= 1.3 {
			statuses[word] = "leech"
		} else {
			statuses[word] = "flashcard"
		}
	}
	return statuses
}
