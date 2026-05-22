package render

import (
	"strings"
	"testing"
)

func TestWrapContent_WidthZero(t *testing.T) {
	input := "hello world"
	got := WrapContent(input, 0)
	if got != input {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestWrapContent_EmptyString(t *testing.T) {
	got := WrapContent("", 40)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestWrapContent_EnglishWordBoundary(t *testing.T) {
	input := "hello world foo"
	got := WrapContent(input, 10)
	expected := "hello\nworld foo"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestWrapContent_EnglishFits(t *testing.T) {
	input := "short"
	got := WrapContent(input, 10)
	if got != input {
		t.Errorf("expected %q, got %q", input, got)
	}
}

func TestWrapContent_PureCJK(t *testing.T) {
	// Each CJK char is 2 cells wide. Width=8 fits 4 chars.
	input := "这是一个测试文本"
	got := WrapContent(input, 8)
	expected := "这是一个\n测试文本"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestWrapContent_PureCJKOddWidth(t *testing.T) {
	// Width=7: only 3 CJK chars fit (6 cells), 4th would need 8.
	input := "这是一个测试"
	got := WrapContent(input, 7)
	expected := "这是一\n个测试"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestWrapContent_MixedEnglishCJK(t *testing.T) {
	// "test" = 4 cells, "这是一" = 6 cells, total 10 fits in width=10
	input := "test这是一个"
	got := WrapContent(input, 10)
	expected := "test这是一\n个"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestWrapContent_MixedWithSlashSeparator(t *testing.T) {
	// Simulates the dictionary entry pattern: "English text / Chinese text"
	input := "revolt / 叛变中表现"
	got := WrapContent(input, 16)
	// "revolt / " = 9 cells, "叛变中" = 6 cells → 15 fits
	// then "表现" would make 19 > 16, so break before 表
	expected := "revolt / 叛变中\n表现"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestWrapContent_ANSIPreservedEnglish(t *testing.T) {
	input := "\x1b[32mhello world\x1b[0m"
	got := WrapContent(input, 6)
	// "hello" = 5 cells fits, then "world" = 5 cells goes to next line
	// ANSI codes should be preserved
	if !strings.Contains(got, "\x1b[32m") {
		t.Error("ANSI start code lost")
	}
	if !strings.Contains(got, "\x1b[0m") {
		t.Error("ANSI reset code lost")
	}
	if !strings.Contains(got, "\n") {
		t.Error("expected line break")
	}
}

func TestWrapContent_ANSIPreservedCJK(t *testing.T) {
	input := "\x1b[31m这是一个测试\x1b[0m"
	got := WrapContent(input, 8)
	if !strings.Contains(got, "\x1b[31m") {
		t.Error("ANSI start code lost")
	}
	if !strings.Contains(got, "\x1b[0m") {
		t.Error("ANSI reset code lost")
	}
	if !strings.Contains(got, "\n") {
		t.Error("expected line break for CJK text")
	}
}

func TestWrapContent_KinsokuNoStart(t *testing.T) {
	// Width=8: "这是一个" = 8 cells exactly.
	// Next char "，" is noStart — should NOT start a new line.
	// So "这是一个，" stays on one line (overflow to 10 cells).
	input := "这是一个，然后"
	got := WrapContent(input, 8)
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %q", len(lines), got)
	}
	// First line should contain the comma (noStart kept on same line)
	if !strings.Contains(lines[0], "，") {
		t.Errorf("noStart comma should stay on first line, got lines: %v", lines)
	}
}

func TestWrapContent_KinsokuNoEnd(t *testing.T) {
	// "这是一（" — '（' is noEnd, shouldn't end a line.
	// Width=8: "这是一" = 6, "（" = 2, total 8 fits on line.
	// Next char '二' would overflow: 8+2=10>8.
	// Since lastNoEnd is true (（ was the last written), keep '二' on same line.
	input := "这是一（二）三"
	got := WrapContent(input, 8)
	lines := strings.Split(got, "\n")
	if len(lines) < 1 {
		t.Fatal("expected output")
	}
	// '（' and '二' should be on the same line
	for _, line := range lines {
		if strings.Contains(line, "（") && !strings.Contains(line, "二") {
			t.Errorf("noEnd '（' should not end a line without following char, got lines: %v", lines)
		}
	}
}

func TestWrapContent_Multiline(t *testing.T) {
	input := "line one\nline two"
	got := WrapContent(input, 40)
	if got != input {
		t.Errorf("expected unchanged multiline, got %q", got)
	}
}

func TestWrapContent_MultilineWithWrapping(t *testing.T) {
	input := "hello world\n这是一个测试文本"
	got := WrapContent(input, 8)
	lines := strings.Split(got, "\n")
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines (2 from first + 2 from second), got %d: %q", len(lines), got)
	}
}

func TestWrapContent_LongWordOverflow(t *testing.T) {
	// A single word longer than the limit should not cause infinite loop
	input := "superlongword"
	got := WrapContent(input, 5)
	// The word exceeds limit but wordLen(13) is NOT < limit(5), so it won't break.
	// This matches standard wordwrap behavior: overflow without hard break.
	if got != input {
		t.Errorf("long word should overflow gracefully, got %q", got)
	}
}
