package render

// WrapContent wraps long lines in text to fit within the given width.
// CJK characters are broken at any character boundary while English text
// breaks at word boundaries. ANSI escape codes are preserved.
// Kinsoku (避头尾) rules are applied to CJK punctuation.
func WrapContent(text string, width int) string {
	if width <= 0 {
		return text
	}
	return cjkWrapText(text, width)
}
