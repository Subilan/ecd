package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// WrapContent wraps long lines in text to fit within the given width.
// ANSI escape codes are preserved.
func WrapContent(text string, width int) string {
	if width <= 0 {
		return text
	}
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		if lipgloss.Width(line) <= width {
			result = append(result, line)
		} else {
			wrapped := lipgloss.NewStyle().Width(width).Render(line)
			result = append(result, strings.Split(strings.TrimRight(wrapped, " "), "\n")...)
		}
	}
	return strings.Join(result, "\n")
}
