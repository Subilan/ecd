package render

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/x/ansi/parser"
	"github.com/mattn/go-runewidth"
)

var noStartRunes = map[rune]bool{
	'，': true, '。': true, '！': true, '？': true,
	'；': true, '：': true, '、': true,
	'）': true, '】': true, '》': true, '」': true, '』': true,
	'〉': true, '〕': true, '〗': true, '〙': true, '〛': true,
	'］': true, '｝': true,
	')': true, ']': true, '}': true,
	'．': true, '…': true,
}

var noEndRunes = map[rune]bool{
	'（': true, '【': true, '《': true, '「': true, '『': true,
	'〈': true, '〔': true, '〘': true, '〚': true,
	'［': true, '｛': true,
	'(': true, '[': true, '{': true,
}

func cjkWrapText(text string, limit int) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = cjkWrapLine(line, limit)
	}
	return strings.Join(lines, "\n")
}

func cjkWrapLine(line string, limit int) string {
	b := []byte(line)
	if len(b) == 0 {
		return ""
	}

	var (
		buf      bytes.Buffer
		word     bytes.Buffer
		space    bytes.Buffer
		curWidth int
		wordLen  int
		pstate   = parser.GroundState
		lastNoEnd bool
	)

	addSpace := func() {
		curWidth += space.Len()
		buf.Write(space.Bytes())
		space.Reset()
	}

	addWord := func() {
		if word.Len() == 0 {
			return
		}
		addSpace()
		curWidth += wordLen
		buf.Write(word.Bytes())
		word.Reset()
		wordLen = 0
	}

	addNewline := func() {
		buf.WriteByte('\n')
		curWidth = 0
		space.Reset()
		lastNoEnd = false
	}

	i := 0
	for i < len(b) {
		state, action := parser.Table.Transition(pstate, b[i])

		if state == parser.Utf8State {
			r, size := utf8.DecodeRune(b[i:])
			if r == utf8.RuneError && size <= 1 {
				// Invalid UTF-8, pass through
				word.WriteByte(b[i])
				wordLen++
				pstate = parser.GroundState
				i++
				continue
			}

			rw := runewidth.RuneWidth(r)
			cluster := b[i : i+size]
			i += size

			if unicode.IsSpace(r) {
				addWord()
				space.Write(cluster)
				lastNoEnd = false
			} else if rw >= 2 {
				// Wide (CJK) character: individually breakable
				addWord()

				totalNeeded := space.Len() + rw
				if curWidth+totalNeeded > limit && curWidth > 0 {
					if isNoStart(r) || lastNoEnd {
						// Kinsoku: keep on current line (tolerate overflow)
						addSpace()
						buf.Write(cluster)
						curWidth += rw
					} else {
						addNewline()
						buf.Write(cluster)
						curWidth += rw
					}
				} else {
					addSpace()
					buf.Write(cluster)
					curWidth += rw
				}
				lastNoEnd = isNoEnd(r)
			} else {
				// Narrow non-ASCII (e.g. accented Latin, symbols)
				word.Write(cluster)
				wordLen += rw
				if curWidth+space.Len()+wordLen > limit && wordLen < limit {
					addNewline()
				}
			}

			pstate = parser.GroundState
			continue
		}

		switch action {
		case parser.PrintAction, parser.ExecuteAction:
			r := rune(b[i])
			switch {
			case r == '\n':
				addWord()
				addNewline()
			case unicode.IsSpace(r):
				addWord()
				space.WriteByte(b[i])
				lastNoEnd = false
			default:
				word.WriteByte(b[i])
				wordLen++
				if curWidth+space.Len()+wordLen > limit && wordLen < limit {
					addNewline()
				}
				lastNoEnd = false
			}
		default:
			// ANSI escape sequence bytes: pass through without counting width
			word.WriteByte(b[i])
		}

		if pstate != parser.Utf8State {
			pstate = state
		}
		i++
	}

	addWord()

	return buf.String()
}

func isNoStart(r rune) bool {
	return noStartRunes[r]
}

func isNoEnd(r rune) bool {
	return noEndRunes[r]
}
