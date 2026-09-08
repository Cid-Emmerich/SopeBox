package captions

import "strings"

// Span is a run of text with the index of the transcript word it belongs
// to (-1 for the speaker prefix), so the UI can highlight the spoken word.
type Span struct {
	Text string
	Word int
}

// Line is one wrapped caption row.
type Line []Span

// Wrap lays out a speaker prefix and words into rows of at most width
// cells. Each word keeps its transcript index.
func Wrap(prefix string, words []string, width int) []Line {
	if width < 4 {
		width = 4
	}
	var lines []Line
	var cur Line
	col := 0
	push := func(s Span) {
		n := len([]rune(s.Text))
		if col+n > width && col > 0 {
			lines = append(lines, cur)
			cur = nil
			col = 0
			s.Text = strings.TrimLeft(s.Text, " ")
			n = len([]rune(s.Text))
		}
		cur = append(cur, s)
		col += n
	}
	if prefix != "" {
		push(Span{Text: prefix + " ", Word: -1})
	}
	for i, w := range words {
		if i > 0 {
			push(Span{Text: " " + w, Word: i})
		} else {
			push(Span{Text: w, Word: i})
		}
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}
	return lines
}

// WrapText wraps plain text into rows of at most width runes.
func WrapText(s string, width int) []string {
	if width < 1 {
		return nil
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, w := range words {
			if line == "" {
				line = w
			} else if len([]rune(line))+1+len([]rune(w)) <= width {
				line += " " + w
			} else {
				out = append(out, line)
				line = w
			}
		}
		out = append(out, line)
	}
	return out
}
