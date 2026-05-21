package ai

import (
	"fmt"
	"strings"

	"github.com/Subilan/ecd/internal/i18n"
)

func responseLang() string {
	if i18n.GetLang() == i18n.LangEN {
		return "English"
	}
	return "Chinese"
}

func countDescription(level string) string {
	switch level {
	case "one":
		return "exactly 1"
	case "some":
		return "3 to 5"
	case "many":
		return "as many as possible while ensuring accuracy (up to 15)"
	default:
		return "3 to 5"
	}
}

// DiffPrompt builds the system and user prompts for /diff.
func DiffPrompt(words []string) (system, user string) {
	lang := responseLang()
	system = fmt.Sprintf(
		`You are an English language expert. Given a list of English words, explain their differences in nuance, usage, and context. If any of the words are not real English words, set valid to false and explain why in reason (in %s). Respond in JSON format with exactly this structure: {"valid": true/false, "reason": "explanation if invalid, empty string if valid", "explanation": "one paragraph explaining differences in %s", "examples": ["example sentence 1", "example sentence 2"]}. When valid is false, explanation and examples can be empty.`,
		lang, lang,
	)
	user = fmt.Sprintf("Explain the differences between these words: %s.", strings.Join(words, ", "))
	return
}

// AntPrompt builds prompts for /ant.
func AntPrompt(word, countLevel string) (system, user string) {
	lang := responseLang()
	system = fmt.Sprintf(
		`You are an English language expert. Generate %s true antonyms for the given English word. A true antonym must be a direct, widely recognized opposite — the pair should appear together in a standard thesaurus. Do NOT suggest words that are merely loosely opposite in connotation (e.g., "explore" and "neglect" are NOT antonyms; "hot" and "cold" ARE). If the input is not a recognized English word, or if the word has no true antonyms, set valid to false and explain why in reason (in %s). Respond in JSON format with exactly this structure: {"valid": true/false, "reason": "explanation if invalid, empty string if valid", "antonyms": [{"word": "the antonym", "definition": "brief %s definition"}]}. When valid is false, antonyms can be empty. Output only the JSON object, no other text.`,
		countDescription(countLevel), lang, lang,
	)
	user = fmt.Sprintf("Generate antonyms for: %s", word)
	return
}

// SynPrompt builds prompts for /syn.
func SynPrompt(word, countLevel string) (system, user string) {
	lang := responseLang()
	system = fmt.Sprintf(
		`You are an English language expert. Generate %s true synonyms for the given English word. A true synonym must share the same core meaning and be interchangeable in most contexts. Do NOT suggest words that are merely loosely related or vaguely similar. If the input is not a recognized English word, or if the word has no true synonyms, set valid to false and explain why in reason (in %s). Respond in JSON format with exactly this structure: {"valid": true/false, "reason": "explanation if invalid, empty string if valid", "synonyms": [{"word": "the synonym", "definition": "brief %s definition"}]}. When valid is false, synonyms can be empty. Output only the JSON object, no other text.`,
		countDescription(countLevel), lang, lang,
	)
	user = fmt.Sprintf("Generate synonyms for: %s", word)
	return
}

// PhrPrompt builds prompts for /phr.
func PhrPrompt(word, countLevel string) (system, user string) {
	lang := responseLang()
	system = fmt.Sprintf(
		`You are an English language expert. Generate %s common English phrases or collocations that include the given word. Each phrase should be a natural expression. If the input is not a recognized English word, or if the word is too obscure for common phrases, set valid to false and explain why in reason (in %s). Respond in JSON format with exactly this structure: {"valid": true/false, "reason": "explanation if invalid, empty string if valid", "phrases": [{"word": "the phrase", "definition": "brief %s definition or translation"}]}. When valid is false, phrases can be empty. Output only the JSON object, no other text.`,
		countDescription(countLevel), lang, lang,
	)
	user = fmt.Sprintf("Generate phrases containing: %s", word)
	return
}

// ExamplePrompt builds prompts for /example.
func ExamplePrompt(word string) (system, user string) {
	lang := responseLang()
	system = fmt.Sprintf(
		`You are an English language expert. Generate 3 to 5 example sentences demonstrating how to use the given English word in different natural contexts. Sentences should vary in structure. Provide each sentence in English with a Chinese translation. If the input is not a recognized English word, or is a proper noun/person name/place name that has no practical need for example sentences, set valid to false and explain why in reason (in %s). Respond in JSON format with exactly this structure: {"valid": true/false, "reason": "explanation if invalid, empty string if valid", "word": "the word", "examples": [{"en": "English sentence", "zh": "Chinese translation"}]}. When valid is false, examples can be empty. Output only the JSON object, no other text.`,
		lang,
	)
	user = fmt.Sprintf("Generate example sentences for: %s", word)
	return
}

// ExplainPrompt builds prompts for /explain.
func ExplainPrompt(word string) (system, user string) {
	lang := responseLang()
	system = fmt.Sprintf(
		`You are an English language expert. Provide a detailed explanation of the given English word: a clear definition (in %s), brief etymology, usage notes (in %s), and example sentences. If the input is not a recognized English word, set valid to false and explain why in reason (in %s). Respond in JSON format with exactly this structure: {"valid": true/false, "reason": "explanation if invalid, empty string if valid", "word": "the word", "definition": "clear %s definition", "etymology": "brief etymology or empty string", "usage_notes": "usage notes in %s or empty string", "example_sentences": ["example 1", "example 2"]}. When valid is false, other fields can be empty. Output only the JSON object, no other text.`,
		lang, lang, lang, lang, lang,
	)
	user = fmt.Sprintf("Explain the word: %s", word)
	return
}
