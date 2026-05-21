package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Subilan/ecd/internal/dict"
	"github.com/Subilan/ecd/internal/i18n"
)

// ANSI color codes
const (
	reset  = "\033[0m"
	source = "\033[36m" // cyan
	word   = "\033[33m" // yellow
	label  = "\033[32m" // green
	pron   = "\033[35m" // purple
	dim    = "\033[2m"
	warn   = "\033[31m" // red
)

var useColor = true

// DisableColor disables ANSI color output.
func DisableColor() { useColor = false }

// IsColorEnabled returns whether color output is enabled.
func IsColorEnabled() bool { return useColor }

// C wraps text in an ANSI color code if colors are enabled.
func C(name, text string) string {
	if !useColor {
		return text
	}
	var code string
	switch name {
	case "source":
		code = source
	case "word":
		code = word
	case "label":
		code = label
	case "pron":
		code = pron
	case "dim":
		code = dim
	case "warn":
		code = warn
	default:
		return text
	}
	return code + text + reset
}

// bracketColor returns the ANSI color code for brackets based on flashcard status.
func bracketColor(status string) string {
	switch status {
	case "leech":
		return warn
	case "flashcard":
		return label
	default:
		return dim
	}
}

// PrintResultsEnglish prints formatted English search results.
func PrintResultsEnglish(results []dict.Entry, statuses map[string]string) {
	for _, r := range results {
		srcLabel := i18n.T("source." + r.Source)
		posStr := ""
		if r.Pos != "" {
			posStr = " " + r.Pos
		}

		pronStr := ""
		if len(r.Pronunciation) > 0 {
			joined := strings.Join(r.Pronunciation, " | ")
			pronStr = C("dim", " /") + C("pron", joined) + C("dim", "/")
		}

		bc := bracketColor(statuses[r.Word])
		wordPart := C("source", srcLabel) + ": " +
			bcStr("[", bc) + " " + C("word", r.Word) + " " + bcStr("]", bc) +
			pronStr + C("dim", posStr)

		if r.CrossRef != "" {
			wordPart += "  -> see: " + r.CrossRef
		}
		fmt.Println(wordPart)
		printEntryBody(r, "")
		fmt.Println()
		fmt.Println()
	}
}

func bcStr(s, code string) string {
	if !useColor {
		return s
	}
	return code + s + reset
}

// PrintResultsChinese prints formatted Chinese FTS5 search results.
func PrintResultsChinese(results []dict.ChineseResult, statuses map[string]string) {
	for _, r := range results {
		srcLabel := i18n.T("source." + r.Source)
		bc := bracketColor(statuses[r.Word])
		fmt.Print(C("source", srcLabel) + ": " +
			bcStr("[", bc) + " " + C("word", r.Word) + " " + bcStr("]", bc) +
			" " + r.CnDef + "\n")
		for i, ex := range r.Examples {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s\n", ex)
		}
		fmt.Println()
	}
}

func printEntryBody(r dict.Entry, indent string) {
	if r.CnDefinition != "" {
		fmt.Printf("%s%s\n", indent,
			C("label", i18n.T("label.definition")+":")+" "+r.CnDefinition)
	}
	for _, ex := range r.Examples {
		if ex.En != "" && ex.Cn != "" {
			fmt.Printf("%s%s %s / %s\n", indent,
				C("label", i18n.T("label.example")+":"), ex.En, ex.Cn)
		} else if ex.En != "" {
			fmt.Printf("%s%s %s\n", indent,
				C("label", i18n.T("label.example")+":"), ex.En)
		} else if ex.Cn != "" {
			fmt.Printf("%s%s %s\n", indent,
				C("label", i18n.T("label.example_cn")+":"), ex.Cn)
		}
	}

	if len(r.Synonyms) > 0 {
		var parts []string
		for _, s := range r.Synonyms {
			parts = append(parts, C("word", s))
		}
		synText := strings.Join(parts, C("dim", ", "))
		fmt.Printf("%s%s %s\n", indent,
			C("label", i18n.T("label.synonym")+":"), synText)
	}

	if len(r.Antonyms) > 0 {
		var parts []string
		for _, a := range r.Antonyms {
			parts = append(parts, C("word", a))
		}
		antText := strings.Join(parts, C("dim", ", "))
		fmt.Printf("%s%s %s\n", indent,
			C("label", i18n.T("label.antonym")+":"), antText)
	}

	for _, note := range r.ExtraNotes {
		typeLabel := noteTypeLabel(note.Type)
		fmt.Printf("%s%s\n", indent, C("label", "["+typeLabel+"]"))
		enLines := strings.Split(note.En, "\n")
		cnLines := strings.Split(note.Cn, "\n")
		maxLines := len(enLines)
		if len(cnLines) > maxLines {
			maxLines = len(cnLines)
		}
		for i := 0; i < maxLines; i++ {
			var enPart, cnPart string
			if i < len(enLines) {
				enPart = strings.TrimRight(enLines[i], " ")
			}
			if i < len(cnLines) {
				cnPart = strings.TrimRight(cnLines[i], " ")
			}
			if enPart != "" && cnPart != "" {
				fmt.Printf("%s%s / %s\n", indent, enPart, cnPart)
			} else if enPart != "" {
				fmt.Printf("%s%s\n", indent, enPart)
			} else if cnPart != "" {
				fmt.Printf("%s%s\n", indent, cnPart)
			} else {
				fmt.Println()
			}
		}
	}
}

func noteTypeLabel(noteType string) string {
	labels := map[string]string{
		"usage":     i18n.T("note.usage"),
		"drv":       i18n.T("note.drv"),
		"regional":  i18n.T("note.regional"),
		"sense":     i18n.T("note.sense"),
		"quotation": i18n.T("note.quotation"),
		"phrase":    i18n.T("note.phrase"),
		"note":      i18n.T("note.general"),
	}
	if label, ok := labels[noteType]; ok {
		return label
	}
	return noteType
}

// PrintIdioms prints Oxford idiom results in CLI mode.
func PrintIdioms(idioms []dict.Idiom, word string) {
	fmt.Printf("%s\n",
		C("label", i18n.T("idiom.found", len(idioms), word)))
	for _, idiom := range idioms {
		fmt.Printf("  %s", C("word", idiom.IdiomPhrase))
		if idiom.CnDefinition != "" {
			fmt.Printf("  %s", idiom.CnDefinition)
		}
		fmt.Println()
		for _, ex := range idiom.Examples {
			fmt.Printf("    %s %s\n",
				C("label", i18n.T("label.example")+":"), ex)
		}
	}
}

// PrintFlashcardEntry prints one dictionary entry for flashcard review.
func PrintFlashcardEntry(entry dict.Entry, idx, total int) {
	src := i18n.T("source." + entry.Source)
	posStr := entry.Pos
	if posStr == "" {
		posStr = "(none)"
	}
	fmt.Printf("\n  %s\n", C("dim", fmt.Sprintf(i18n.T("review.entry_n_of"), idx+1, total, src)))
	fmt.Printf("  %s %s\n", C("label", i18n.T("label.pos")+":"), posStr)
	printEntryBody(entry, "  ")

	rating := fmt.Sprintf("0=%s 1=%s 2=%s 3=%s",
		i18n.T("review.again"), i18n.T("review.hard"),
		i18n.T("review.good"), i18n.T("review.easy"))
	if total > 1 {
		fmt.Printf("\n  %s\n", C("dim", i18n.T("review.switch_entry")+"  |  "+rating))
	} else {
		fmt.Printf("\n  %s\n", C("dim", rating))
	}
}

// PrintDeckStats prints flashcard deck statistics.
func PrintDeckStats(stats *DeckStats) {
	if stats.Total == 0 {
		fmt.Println(i18n.T("deck.empty"))
		return
	}

	fmt.Printf("\n%s\n", C("label", i18n.T("deck.stats_title")))
	fmt.Printf("  %s        %d\n", C("label", i18n.T("deck.total")+":"), stats.Total)
	fmt.Printf("  %s          %d\n", C("label", i18n.T("deck.due")+":"), stats.Due)
	fmt.Printf("  %s          %d\n", C("label", i18n.T("deck.new")+":"), stats.New)
	fmt.Printf("  %s       %d\n", C("label", i18n.T("deck.mature")+":"), stats.Mature)

	if stats.NextDelta != "" {
		if stats.Overdue {
			fmt.Printf("  %s         %s%s\n",
				C("label", i18n.T("deck.next")+":"),
				stats.NextDelta,
				C("warn", " ago"))
		} else {
			fmt.Printf("  %s         in %s\n",
				C("label", i18n.T("deck.next")+":"),
				stats.NextDelta)
		}
	}

	if stats.Leeches > 0 {
		fmt.Printf("  %s      %d\n", C("label", i18n.T("deck.leeches")+":"), stats.Leeches)
	}

	fmt.Printf("  %s     %.0f%%\n", C("label", i18n.T("deck.avg_ease")+":"), stats.AvgEase*100)
	fmt.Println()
}

// DeckStats mirrors history.DeckStats for the cli package.
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

// PrintFlashcardSummary prints a brief summary of a word for "add" preview.
func PrintFlashcardSummary(entry dict.Entry) {
	posStr := ""
	if entry.Pos != "" {
		posStr = fmt.Sprintf(" (%s)", entry.Pos)
	}
	fmt.Print(C("word", entry.Word) + C("dim", posStr))
	if entry.CnDefinition != "" {
		firstLine := strings.Split(entry.CnDefinition, "；")[0]
		firstLine = strings.Split(firstLine, ";")[0]
		firstLine = strings.Split(firstLine, ".")[0]
		if len([]rune(firstLine)) > 60 {
			firstLine = string([]rune(firstLine)[:60]) + "..."
		}
		fmt.Printf("  %s", firstLine)
	}
	fmt.Println()
}

// ParseNotesFromJSON parses the extra_notes JSON column.
func ParseNotesFromJSON(raw string) []dict.Note {
	var notes []dict.Note
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &notes); err != nil {
			return nil
		}
	}
	return notes
}
