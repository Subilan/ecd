package dict

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Subilan/ecd/internal/config"
)

// DB wraps the dictionary SQLite connection and provides query methods.
type DB struct {
	*sql.DB
}

// OpenDictDB opens the dictionary database.
func OpenDictDB() (*DB, error) {
	if config.DBPath == "" {
		return nil, fmt.Errorf("dictionary database not found (set ECD_DB_PATH)")
	}
	db, err := sql.Open("sqlite", config.DBPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("open dict db: %w", err)
	}
	return &DB{db}, nil
}

type searchMode int

const (
	modeExact  searchMode = iota
	modePrefix
)

func (d *DB) searchEntries(query string, source *string, mode searchMode) ([]Entry, error) {
	var whereClause string
	var param string
	switch mode {
	case modeExact:
		whereClause = "e.word = ? COLLATE NOCASE"
		param = query
	case modePrefix:
		whereClause = "e.word LIKE ? COLLATE NOCASE"
		param = query + "%"
	}

	sources := []string{"collins", "oxford"}
	if source != nil {
		sources = []string{*source}
	}

	var orderClause string
	if mode == modeExact {
		orderClause = "e.pos, e.sense_order"
	} else {
		orderClause = "e.word, e.pos, e.sense_order"
	}

	var all []Entry
	for _, src := range sources {
		rows, err := d.Query(fmt.Sprintf(`
			SELECT e.id, e.word, e.pos, e.cn_definition, e.cross_ref,
			       e.sense_order, e.pronunciation, e.extra_notes
			FROM %s_entries e
			WHERE %s
			ORDER BY %s
		`, src, whereClause, orderClause), param)
		if err != nil {
			return nil, fmt.Errorf("%s search: %w", src, err)
		}
		defer rows.Close()

		for rows.Next() {
			var e Entry
			e.Source = src
			var pronJSON, notesJSON sql.NullString
			var crossRef sql.NullString
			if err := rows.Scan(&e.ID, &e.Word, &e.Pos, &e.CnDefinition,
				&crossRef, &e.SenseOrder, &pronJSON, &notesJSON); err != nil {
				return nil, fmt.Errorf("scan entry: %w", err)
			}
			if crossRef.Valid {
				e.CrossRef = crossRef.String
			}
			if pronJSON.Valid {
				json.Unmarshal([]byte(pronJSON.String), &e.Pronunciation)
			}
			if notesJSON.Valid {
				json.Unmarshal([]byte(notesJSON.String), &e.ExtraNotes)
			}

			// Fetch examples
			exRows, err := d.Query(fmt.Sprintf(`
				SELECT en_example, cn_example
				FROM %s_examples
				WHERE entry_id = ?
				ORDER BY example_order
			`, src), e.ID)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("fetch examples: %w", err)
			}
			for exRows.Next() {
				var ex Example
				var en, cn sql.NullString
				if err := exRows.Scan(&en, &cn); err != nil {
					exRows.Close()
					return nil, fmt.Errorf("scan example: %w", err)
				}
				if en.Valid {
					ex.En = en.String
				}
				if cn.Valid {
					ex.Cn = cn.String
				}
				e.Examples = append(e.Examples, ex)
			}
			exRows.Close()

			// Fetch synonyms
			synRows, err := d.Query(
				"SELECT synonym_word FROM synonyms WHERE source=? AND entry_id=? ORDER BY id",
				src, e.ID)
			if err == nil {
				for synRows.Next() {
					var s string
					if err := synRows.Scan(&s); err == nil {
						e.Synonyms = append(e.Synonyms, s)
					}
				}
				synRows.Close()
			}

			// Fetch antonyms
			antRows, err := d.Query(
				"SELECT antonym_word FROM antonyms WHERE source=? AND entry_id=? ORDER BY id",
				src, e.ID)
			if err == nil {
				for antRows.Next() {
					var a string
					if err := antRows.Scan(&a); err == nil {
						e.Antonyms = append(e.Antonyms, a)
					}
				}
				antRows.Close()
			}

			all = append(all, e)
		}
		rows.Close()
	}
	return all, nil
}

// SearchExact searches for entries matching the exact word (case-insensitive).
func (d *DB) SearchExact(query string, source *string) ([]Entry, error) {
	return d.searchEntries(query, source, modeExact)
}

// SearchPrefix searches for entries whose word starts with the query (case-insensitive).
func (d *DB) SearchPrefix(query string, source *string) ([]Entry, error) {
	return d.searchEntries(query, source, modePrefix)
}

// FuzzyMatch holds the result of fuzzy matching.
type FuzzyMatch struct {
	Word  string
	Score float64
}

// SearchFuzzy returns up to 5 close matches for the word with cutoff >= 0.75.
func (d *DB) SearchFuzzy(query string, source *string) ([]string, error) {
	if len(query) == 0 {
		return nil, nil
	}
	prefix := string([]rune(query)[0])

	sources := []string{"collins", "oxford"}
	if source != nil {
		sources = []string{*source}
	}

	candidates := make(map[string]bool)
	for _, src := range sources {
		rows, err := d.Query(fmt.Sprintf(
			"SELECT DISTINCT word FROM %s_entries WHERE word LIKE ? COLLATE NOCASE",
			src), prefix+"%")
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var w string
			if err := rows.Scan(&w); err == nil {
				candidates[w] = true
			}
		}
		rows.Close()
	}

	// Compute fuzzy scores using a simplified Ratcliff/Obershelp-like approach
	type scored struct {
		word  string
		score float64
	}
	var scoredCandidates []scored
	for w := range candidates {
		s := similarity(strings.ToLower(w), strings.ToLower(query))
		if s >= 0.75 {
			scoredCandidates = append(scoredCandidates, scored{w, s})
		}
	}

	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].score > scoredCandidates[j].score
	})

	if len(scoredCandidates) > 5 {
		scoredCandidates = scoredCandidates[:5]
	}

	result := make([]string, len(scoredCandidates))
	for i, sc := range scoredCandidates {
		result[i] = sc.word
	}
	return result, nil
}

// similarity computes a simple string similarity (0.0 to 1.0) based on
// common bigram overlap, approximating difflib's SequenceMatcher behavior.
func similarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}
	// Use a simple matching approach: count matching characters in order
	matches := 0
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		if a[i] == b[j] {
			matches++
			i++
			j++
		} else {
			// Try to find the next match
			found := false
			for k := j + 1; k < len(b); k++ {
				if a[i] == b[k] {
					matches++
					i++
					j = k + 1
					found = true
					break
				}
			}
			if !found {
				i++
			}
		}
	}
	// Ratio: 2 * matches / (len(a) + len(b))
	return 2.0 * float64(matches) / float64(len(a)+len(b))
}

// SearchChinese performs an FTS5 full-text search for Chinese text.
func (d *DB) SearchChinese(text string, source *string) ([]ChineseResult, error) {
	rows, err := d.Query(`
		SELECT source, word, cn_definition, en_example, cn_example
		FROM entries_fts
		WHERE entries_fts MATCH ?
		ORDER BY rank
		LIMIT 50
	`, text)
	if err != nil {
		return nil, fmt.Errorf("fts5 search: %w", err)
	}
	defer rows.Close()

	type key struct {
		src   string
		word  string
		cnDef string
	}
	grouped := make(map[key][]string)
	var order []key

	for rows.Next() {
		var src, word, cnDef string
		var enEx, cnEx sql.NullString
		if err := rows.Scan(&src, &word, &cnDef, &enEx, &cnEx); err != nil {
			return nil, fmt.Errorf("scan fts5: %w", err)
		}
		if source != nil && src != *source {
			continue
		}
		k := key{src, word, cnDef}
		if _, ok := grouped[k]; !ok {
			order = append(order, k)
		}
		ex := cnEx.String
		if ex == "" {
			ex = enEx.String
		}
		if ex != "" {
			grouped[k] = append(grouped[k], ex)
		}
	}

	var results []ChineseResult
	for _, k := range order {
		results = append(results, ChineseResult{
			Source:   k.src,
			Word:     k.word,
			CnDef:    k.cnDef,
			Examples: grouped[k],
		})
	}
	return results, nil
}

// RandomWord returns a random word from the dictionary.
func (d *DB) RandomWord(source *string) (string, error) {
	sources := []string{"collins", "oxford"}
	if source != nil {
		sources = []string{*source}
	}
	// Simple pseudo-random: alternate or use one source
	src := sources[0]
	if source == nil {
		// Alternate: use collins for deterministic behavior in CLI mode
		src = "collins"
	}

	var word string
	err := d.QueryRow(fmt.Sprintf(
		"SELECT word FROM %s_entries WHERE cn_definition != '' AND cross_ref IS NULL ORDER BY RANDOM() LIMIT 1",
		src)).Scan(&word)
	if err != nil {
		return "", err
	}
	return word, nil
}

// GetIdioms returns all Oxford idioms for a word.
func (d *DB) GetIdioms(word string) ([]Idiom, error) {
	rows, err := d.Query(
		"SELECT idiom_phrase, cn_definition, examples FROM oxford_idioms WHERE word = ? ORDER BY id",
		word,
	)
	if err != nil {
		return nil, fmt.Errorf("idioms: %w", err)
	}
	defer rows.Close()

	var idioms []Idiom
	for rows.Next() {
		var i Idiom
		var examplesJSON sql.NullString
		if err := rows.Scan(&i.IdiomPhrase, &i.CnDefinition, &examplesJSON); err != nil {
			return nil, fmt.Errorf("scan idiom: %w", err)
		}
		if examplesJSON.Valid {
			var pairs [][]string
			if err := json.Unmarshal([]byte(examplesJSON.String), &pairs); err == nil {
				for _, p := range pairs {
					if len(p) == 2 {
						i.Examples = append(i.Examples, p[0]+" / "+p[1])
					}
				}
			}
		}
		idioms = append(idioms, i)
	}
	return idioms, rows.Err()
}
