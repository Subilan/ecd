package search

import (
	"github.com/Subilan/ecd/internal/config"
	"github.com/Subilan/ecd/internal/dict"
	"github.com/Subilan/ecd/internal/history"
	"github.com/Subilan/ecd/internal/i18n"
)

// Context holds the shared state for search operations.
type Context struct {
	DictDB    *dict.DB
	HistoryDB *history.DB
	LastWord  *string
}

// HandleQuery performs the 4-tier search dispatch for a query.
// Returns results and the word that should be recorded (if any).
func HandleQuery(ctx *Context, query string, source *string) *QueryResult {
	var srcPtr *string
	if source != nil && *source != "" {
		srcPtr = source
	}

	if config.IsChineseQuery(query) {
		results, err := ctx.DictDB.SearchChinese(query, srcPtr)
		if err != nil || len(results) == 0 {
			return &QueryResult{NotFound: true, Query: query}
		}
		ctx.HistoryDB.RecordLookup(query)
		*ctx.LastWord = query
		return &QueryResult{Chinese: results}
	}

	// English exact match
	entries, err := ctx.DictDB.SearchExact(query, srcPtr)
	if err == nil && len(entries) > 0 {
		ctx.HistoryDB.RecordLookup(query)
		*ctx.LastWord = query
		return &QueryResult{Entries: entries}
	}

	// English prefix match
	prefixEntries, err := ctx.DictDB.SearchPrefix(query, srcPtr)
	if err == nil && len(prefixEntries) > 0 {
		// Count distinct words
		distinct := make(map[string]bool)
		for _, e := range prefixEntries {
			distinct[e.Word] = true
		}
		if len(distinct) == 1 {
			// Single distinct word — show results
			var word string
			for w := range distinct {
				word = w
			}
			ctx.HistoryDB.RecordLookup(word)
			*ctx.LastWord = word
			return &QueryResult{Entries: prefixEntries}
		}
		// Multiple distinct words — show suggestions
		suggestions := sortedKeys(distinct)
		if len(suggestions) > 10 {
			suggestions = suggestions[:10]
		}
		return &QueryResult{Suggestions: suggestions, Query: query}
	}

	// English fuzzy match
	fuzzyMatches, _ := ctx.DictDB.SearchFuzzy(query, srcPtr)
	if len(fuzzyMatches) > 0 {
		return &QueryResult{Suggestions: fuzzyMatches, Query: query}
	}

	// Fall through to Chinese FTS5
	cnResults, err := ctx.DictDB.SearchChinese(query, srcPtr)
	if err == nil && len(cnResults) > 0 {
		ctx.HistoryDB.RecordLookup(query)
		*ctx.LastWord = query
		return &QueryResult{Chinese: cnResults}
	}

	return &QueryResult{NotFound: true, Query: query}
}

// QueryResult represents the outcome of a search query.
type QueryResult struct {
	Entries     []dict.Entry
	Chinese     []dict.ChineseResult
	Suggestions []string
	NotFound    bool
	Query       string
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple bubble sort for small sets
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// SynonymResult holds synonyms/antonyms grouped by source.
type SynonymResult struct {
	Source string
	Word   string
	Items  []string
}

// SearchSynonyms finds synonyms for a word.
func SearchSynonyms(ctx *Context, word string, source *string) ([]SynonymResult, *QueryResult) {
	return searchXref(ctx, word, source, "synonym")
}

// SearchAntonyms finds antonyms for a word.
func SearchAntonyms(ctx *Context, word string, source *string) ([]SynonymResult, *QueryResult) {
	return searchXref(ctx, word, source, "antonym")
}

func searchXref(ctx *Context, word string, source *string, xrefType string) ([]SynonymResult, *QueryResult) {
	entries, err := ctx.DictDB.SearchExact(word, source)
	if err != nil || len(entries) == 0 {
		return nil, &QueryResult{NotFound: true, Query: word}
	}

	grouped := make(map[string]map[string]bool) // source -> word -> bool
	var order []string
	srcOrder := make(map[string]string) // word -> source

	for _, e := range entries {
		var items []string
		if xrefType == "synonym" {
			items = e.Synonyms
		} else {
			items = e.Antonyms
		}
		for _, item := range items {
			if _, ok := grouped[e.Source]; !ok {
				grouped[e.Source] = make(map[string]bool)
			}
			if !grouped[e.Source][item] {
				grouped[e.Source][item] = true
				order = append(order, item)
				srcOrder[item] = e.Source
			}
		}
	}

	var results []SynonymResult
	seenSrc := make(map[string]bool)
	for _, item := range order {
		src := srcOrder[item]
		key := src
		if !seenSrc[key] {
			seenSrc[key] = true
			results = append(results, SynonymResult{Source: src, Word: word})
		}
		// Append to last matching result
		for i := range results {
			if results[i].Source == src {
				results[i].Items = append(results[i].Items, item)
			}
		}
	}

	if len(results) == 0 {
		return nil, nil // no xref items found
	}
	return results, nil
}

// FormatSuggestions formats the suggestions list for "did you mean" display.
func FormatSuggestions(suggestions []string) string {
	result := ""
	for i, s := range suggestions {
		if i > 0 {
			result += ", "
		}
		result += s
	}
	return result
}

// NotFoundMessage returns the localized "no results" message.
func NotFoundMessage(query string) string {
	return i18n.T("search.no_results", query)
}

// DidYouMeanMessage returns the localized "did you mean" message.
func DidYouMeanMessage(words string) string {
	return i18n.T("search.did_you_mean", words)
}
