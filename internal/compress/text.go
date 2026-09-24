package compress

// @ACP D ACP.COMPRESS

import (
	"fmt"
	"strings"
)

const (
	diffContext = 1  // unchanged lines kept on each side of a change
	textRepeat  = 2  // identical non-blank lines kept before dropping repeats
	textHead    = 60 // lines kept at the head of long prose
	textTail    = 20 // lines kept at the tail of long prose
	textLong    = 200
)

// compressDiff keeps file headers, hunk headers and changed lines, trimming
// unchanged context to diffContext lines around each change.
// tags: compress, diff_crush
func compressDiff(text string, opt Options) (string, int) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	words := queryWords(opt.Query)
	keep := make([]bool, len(lines))
	isContext := func(s string) bool { return strings.HasPrefix(s, " ") || s == "" }
	for i, ln := range lines {
		if !isContext(ln) || queryHit(ln, words) {
			keep[i] = true
			continue
		}
		for d := 1; d <= diffContext; d++ {
			if i-d >= 0 && isChange(lines[i-d]) || i+d < len(lines) && isChange(lines[i+d]) {
				keep[i] = true
			}
		}
	}
	return renderKept(lines, keep, " ", func(a, b int) string { return "unchanged" })
}

// tags: compress, diff_crush
func isChange(s string) bool {
	return (strings.HasPrefix(s, "+") && !strings.HasPrefix(s, "+++")) || (strings.HasPrefix(s, "-") && !strings.HasPrefix(s, "---"))
}

// compressText handles prose and unknown text: trailing space and blank runs
// collapse, repeated identical lines are dropped after textRepeat, and very
// long text keeps a head, a tail, headings, signal lines and task matches.
// tags: compress, text_crush
func compressText(text string, opt Options) (string, int) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	words := queryWords(opt.Query)
	keep := make([]bool, len(lines))
	count := map[string]int{}
	long := len(lines) > textLong
	for i, raw := range lines {
		ln := strings.TrimRight(raw, " \t")
		lines[i] = ln
		t := strings.TrimSpace(ln)
		if t == "" {
			keep[i] = i == 0 || strings.TrimSpace(lines[i-1]) != ""
			continue
		}
		count[t]++
		if count[t] > textRepeat {
			continue
		}
		if !long {
			keep[i] = true
			continue
		}
		keep[i] = i < textHead || i >= len(lines)-textTail || strings.HasPrefix(t, "#") || important(t) || queryHit(t, words)
	}
	out, elided := renderKept(lines, keep, "", func(a, b int) string {
		repeats := 0
		for _, ln := range lines[a : b+1] {
			if count[strings.TrimSpace(ln)] > textRepeat {
				repeats++
			}
		}
		if repeats > 0 {
			return fmt.Sprintf("[%d repeated]", repeats)
		}
		return ""
	})
	return out, elided
}
