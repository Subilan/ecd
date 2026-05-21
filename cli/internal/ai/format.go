package ai

import (
	"fmt"
	"strings"

	"github.com/Subilan/ecd/internal/i18n"
)

func formatAIResponse(msg AIResultMsg) string {
	parsed, err := ParseJSONResponse(msg.Response)
	if err != nil {
		return aiErrStyle.Render(err.Error())
	}

	if valid, ok := parsed["valid"].(bool); ok && !valid {
		reason, _ := parsed["reason"].(string)
		return formatInvalidResult(msg.CmdDisplay, reason, msg.Cached)
	}

	var b strings.Builder

	header := aiTitleStyle.Render(fmt.Sprintf("[%s]", msg.CmdDisplay))
	if msg.Cached {
		header += aiDimStyle.Render(" " + i18n.T("ai.cached_hint"))
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	switch msg.Kind {
	case "diff":
		formatDiffResult(&b, parsed)
	case "list":
		formatListResult(&b, parsed)
	case "example":
		formatExampleResult(&b, parsed)
	case "explain":
		formatExplainResult(&b, parsed)
	}

	return b.String()
}

func formatInvalidResult(cmdDisplay, reason string, cached bool) string {
	var b strings.Builder
	header := aiTitleStyle.Render(fmt.Sprintf("[%s]", cmdDisplay))
	if cached {
		header += aiDimStyle.Render(" " + i18n.T("ai.cached_hint"))
	}
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(aiWarnStyle.Render("⚠ " + i18n.T("ai.invalid_request")))
	b.WriteString("\n")
	if reason != "" {
		b.WriteString(reason)
		b.WriteString("\n")
	}
	return b.String()
}

func formatDiffResult(b *strings.Builder, parsed map[string]any) {
	if expl, ok := parsed["explanation"].(string); ok {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.explanation")))
		b.WriteString("\n")
		b.WriteString(expl)
		b.WriteString("\n\n")
	}

	if examples, ok := parsed["examples"].([]any); ok {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.examples")))
		b.WriteString("\n")
		for i, ex := range examples {
			entry, ok := ex.(map[string]any)
			if !ok {
				continue
			}
			en, _ := entry["en"].(string)
			zh, _ := entry["zh"].(string)
			if en == "" && zh == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  %d. %s", i+1, en))
			if zh != "" {
				b.WriteString(fmt.Sprintf(" / %s", zh))
			}
			b.WriteString("\n")
		}
	}
}

func formatListResult(b *strings.Builder, parsed map[string]any) {
	var items []any
	var ok bool

	for _, key := range []string{"antonyms", "synonyms", "phrases"} {
		if items, ok = parsed[key].([]any); ok {
			break
		}
	}

	if !ok {
		b.WriteString(aiErrStyle.Render("unexpected response format"))
		return
	}

	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		word, _ := entry["word"].(string)
		def, _ := entry["definition"].(string)
		b.WriteString(fmt.Sprintf("  %s", aiWordStyle.Render(word)))
		if def != "" {
			b.WriteString(aiDimStyle.Render("  —  " + def))
		}
		b.WriteString("\n")
	}
}

func formatExampleResult(b *strings.Builder, parsed map[string]any) {
	examples, ok := parsed["examples"].([]any)
	if !ok {
		return
	}
	b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.examples")))
	b.WriteString("\n")
	for i, ex := range examples {
		entry, ok := ex.(map[string]any)
		if !ok {
			continue
		}
		en, _ := entry["en"].(string)
		zh, _ := entry["zh"].(string)
		if en == "" && zh == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("  %d. %s", i+1, en))
		if zh != "" {
			b.WriteString(fmt.Sprintf(" / %s", zh))
		}
		b.WriteString("\n")
	}
}

func formatExplainResult(b *strings.Builder, parsed map[string]any) {
	if def, ok := parsed["definition"].(string); ok && def != "" {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.definition")))
		b.WriteString("\n")
		b.WriteString(def)
		b.WriteString("\n\n")
	}
	if etym, ok := parsed["etymology"].(string); ok && etym != "" {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.etymology")))
		b.WriteString("\n")
		b.WriteString(etym)
		b.WriteString("\n\n")
	}
	if usage, ok := parsed["usage_notes"].(string); ok && usage != "" {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.usage_notes")))
		b.WriteString("\n")
		b.WriteString(usage)
		b.WriteString("\n\n")
	}
	if examples, ok := parsed["example_sentences"].([]any); ok && len(examples) > 0 {
		b.WriteString(aiLabelStyle.Render(i18n.T("ai.label.examples")))
		b.WriteString("\n")
		for i, ex := range examples {
			entry, ok := ex.(map[string]any)
			if !ok {
				continue
			}
			en, _ := entry["en"].(string)
			zh, _ := entry["zh"].(string)
			if en == "" && zh == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("  %d. %s", i+1, en))
			if zh != "" {
				b.WriteString(fmt.Sprintf(" / %s", zh))
			}
			b.WriteString("\n")
		}
	}
}
