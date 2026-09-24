package compress

// @ACP D ACP.COMPRESS

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	logHead       = 5  // leading lines always kept
	logTail       = 12 // trailing lines always kept; summaries live here
	logRepeats    = 2  // occurrences kept per line template
	logStackAfter = 30 // continuation lines kept after an error line
	logLeadIn     = 2  // lines kept before an error line
)

var (
	// UUIDs, hex addresses, hex-ish IDs containing a digit, then bare numbers.
	volatileRE = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b|\b0x[0-9a-f]+\b|\b[0-9a-f]*[0-9][0-9a-f]*\b|\d+(\.\d+)?`)
	passRE     = regexp.MustCompile(`^\s*(=== (RUN|PAUSE|CONT|NAME)|--- (PASS|SKIP)|PASSED|✓|√|ok\s|\s*PASS\b|.* PASSED\b|.*\[no test files\]|\s*\.+\s*$)`)
	stackRE    = regexp.MustCompile(`^(\s+|goroutine \d|\s*at |\s*File "|Caused by|\s*\.\.\. \d+ more|\S+\.(go|py|js|ts|java|rb|rs|php|cs):\d+)`)
)

// logTemplate normalizes volatile parts so repeated shapes compare equal.
// tags: compress, log_crush
func logTemplate(line string) string {
	if m := passRE.FindString(line); m != "" {
		return "pass:" + strings.TrimSpace(m)
	}
	return volatileRE.ReplaceAllString(strings.TrimSpace(line), "#")
}

// compressLog keeps the head, tail, every error with its stack, task matches
// and the first occurrences of each line shape. Repeats collapse to a counted
// elision marker that names the original line range.
// tags: compress, log_crush
func compressLog(text string, opt Options) (string, int) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	n := len(lines)
	words := queryWords(opt.Query)
	keep := make([]bool, n)
	seen := map[string]int{}
	stack := 0
	for i, ln := range lines {
		pass := passRE.MatchString(ln)
		switch {
		case i == 0 || i == n-1 || (i < logHead || i >= n-logTail) && !pass:
			keep[i] = true
		case important(ln) && !pass:
			keep[i] = true
			// Lead-in: the test header and indented output before a failure.
			for k := i - 1; k >= 0 && k >= i-logLeadIn; k-- {
				if strings.HasPrefix(lines[k], "=== RUN") || strings.HasPrefix(lines[k], " ") || strings.HasPrefix(lines[k], "\t") {
					keep[k] = true
				}
			}
			stack = logStackAfter
			continue
		case stack > 0 && stackRE.MatchString(ln):
			keep[i] = true
			stack--
			continue
		case queryHit(ln, words):
			keep[i] = true
		}
		stack = 0
		t := logTemplate(ln)
		seen[t]++
		if !keep[i] && seen[t] <= logRepeats && !strings.HasPrefix(t, "pass:") {
			keep[i] = true
		}
		if strings.TrimSpace(ln) == "" && i > 0 && strings.TrimSpace(lines[i-1]) == "" {
			keep[i] = false
		}
	}
	return renderKept(lines, keep, "", func(a, b int) string {
		shapes := map[string]int{}
		for _, ln := range lines[a : b+1] {
			shapes[logTemplate(ln)]++
		}
		return describeShapes(shapes)
	})
}

// describeShapes summarizes the dominant repeated line shapes in a run.
// tags: compress, log_crush
func describeShapes(shapes map[string]int) string {
	type kv struct {
		t string
		n int
	}
	list := make([]kv, 0, len(shapes))
	for t, n := range shapes {
		list = append(list, kv{t, n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n == list[j].n {
			return list[i].t < list[j].t
		}
		return list[i].n > list[j].n
	})
	if len(list) == 0 {
		return ""
	}
	top := list[0].t
	if strings.HasPrefix(top, "pass:") {
		top = strings.TrimPrefix(top, "pass:") + " …"
	}
	if len(top) > 60 {
		top = top[:59] + "…"
	}
	if len(list) == 1 {
		return fmt.Sprintf("[%d× %q]", list[0].n, top)
	}
	return fmt.Sprintf("[%d shapes; top %d× %q]", len(list), list[0].n, top)
}

// renderKept writes kept lines and one elision marker per dropped run.
// tags: compress
func renderKept(lines []string, keep []bool, prefix string, describe func(a, b int) string) (string, int) {
	var b strings.Builder
	elided := 0
	for i := 0; i < len(lines); {
		if keep[i] {
			b.WriteString(lines[i])
			b.WriteByte('\n')
			i++
			continue
		}
		j := i
		blank := true
		for {
			blank = blank && strings.TrimSpace(lines[j]) == ""
			if j+1 >= len(lines) || keep[j+1] {
				break
			}
			j++
		}
		if blank {
			i = j + 1 // blank runs vanish without a marker
			continue
		}
		note := ""
		if describe != nil {
			note = describe(i, j)
		}
		if j == i && len(lines[i]) < 60 {
			b.WriteString(lines[i]) // a marker would not be shorter
			b.WriteByte('\n')
		} else {
			b.WriteString(elision(prefix, i+1, j+1, note))
			b.WriteByte('\n')
			elided += j - i + 1
		}
		i = j + 1
	}
	return strings.TrimRight(b.String(), "\n"), elided
}
