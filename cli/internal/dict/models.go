package dict

// Entry represents one word sense from either dictionary.
type Entry struct {
	ID            int      `json:"id"`
	Source        string   `json:"source"`
	Word          string   `json:"word"`
	Pos           string   `json:"pos"`
	CnDefinition  string   `json:"cn_definition"`
	CrossRef      string   `json:"cross_ref"`
	SenseOrder    int      `json:"sense_order"`
	Pronunciation []string `json:"pronunciation"`
	ExtraNotes    []Note   `json:"extra_notes"`
	Examples      []Example `json:"examples"`
	Synonyms      []string `json:"synonyms"`
	Antonyms      []string `json:"antonyms"`
}

// Example is an English-Chinese example sentence pair.
type Example struct {
	En string `json:"en"`
	Cn string `json:"cn"`
}

// Note is an extra annotation attached to a dictionary entry.
type Note struct {
	Type string `json:"type"`
	En   string `json:"en"`
	Cn   string `json:"cn"`
}

// ChineseResult is a FTS5 search result grouped by (source, word, definition).
type ChineseResult struct {
	Source   string
	Word     string
	CnDef    string
	Examples []string
}

// Idiom is an Oxford idiom entry from the oxford_idioms table.
type Idiom struct {
	IdiomPhrase  string   `json:"idiom_phrase"`
	CnDefinition string   `json:"cn_definition"`
	Examples     []string `json:"-"`
}
