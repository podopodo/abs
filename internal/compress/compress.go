package compress

// @ACP O ACP.COMPRESS

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Kinds understood by the compressor. "auto" resolves to one of the others.
const (
	KindAuto = "auto"
	KindJSON = "json"
	KindLog  = "log"
	KindCode = "code"
	KindDiff = "diff"
	KindText = "text"
)

// Options tune one compression pass. The zero value is usable.
type Options struct {
	Kind   string // auto, json, log, code, diff, text
	Path   string // optional file name; selects a code language and aids detection
	Budget int    // optional estimated-token ceiling for the output; 0 disables
	Query  string // optional task text; matching lines and items are preferred
}

// Result describes one compression pass. Output is never longer than the input.
type Result struct {
	Kind             string `json:"kind"`
	Output           string `json:"-"`
	OriginalBytes    int    `json:"original_bytes"`
	CompressedBytes  int    `json:"compressed_bytes"`
	OriginalTokens   int    `json:"original_tokens"`
	CompressedTokens int    `json:"compressed_tokens"`
	Elided           int    `json:"elided_lines"`
	ID               string `json:"id,omitempty"`
}

// Saved reports the estimated tokens removed.
// tags: compress
func (r Result) Saved() int { return r.OriginalTokens - r.CompressedTokens }

// Ratio reports compressed/original tokens in [0,1].
// tags: compress
func (r Result) Ratio() float64 {
	if r.OriginalTokens == 0 {
		return 1
	}
	return float64(r.CompressedTokens) / float64(r.OriginalTokens)
}

// EstimateTokens approximates model tokens as one per four bytes. It is an
// estimate for comparing before and after, never provider telemetry.
// tags: compress, savings
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4
}

// Compress shrinks text for a model's context window while keeping the parts a
// model needs: errors, outliers, structure, signatures and changed lines.
// Elided regions are annotated with original line numbers so they can be
// recovered exactly with `acp expand ID --lines A:B` or by reading the file.
// tags: compress
func Compress(input []byte, opt Options) Result {
	text := normalizeNewlines(string(input))
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "�")
	}
	kind := opt.Kind
	if kind == "" || kind == KindAuto {
		kind = Detect(opt.Path, text)
	}
	var out string
	var elided int
	switch kind {
	case KindJSON:
		var ok bool
		out, ok = crushJSON(text, opt)
		if !ok {
			kind = KindText
			out, elided = compressText(text, opt)
		}
	case KindLog:
		out, elided = compressLog(text, opt)
	case KindCode:
		out, elided = compressCode(text, opt)
	case KindDiff:
		out, elided = compressDiff(text, opt)
	default:
		kind = KindText
		out, elided = compressText(text, opt)
	}
	if opt.Budget > 0 && EstimateTokens(out) > opt.Budget {
		var more int
		out, more = fitBudget(out, opt.Budget)
		elided += more
	}
	if len(out) >= len(text) {
		out, elided = text, 0
	}
	return Result{
		Kind:             kind,
		Output:           out,
		OriginalBytes:    len(text),
		CompressedBytes:  len(out),
		OriginalTokens:   EstimateTokens(text),
		CompressedTokens: EstimateTokens(out),
		Elided:           elided,
	}
}

var (
	diffRE   = regexp.MustCompile(`(?m)^(diff --git |\+\+\+ \S|@@ -\d+(,\d+)? \+\d+)`)
	logRE    = regexp.MustCompile(`(?mi)^\s*(\[?\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}|\[?\d{2}:\d{2}:\d{2}|\[?(trace|debug|info|warn|warning|error|fatal)\b|(=== RUN|--- (pass|fail|skip)|ok\s+\S+\s+\d)|time=|level=)`)
	codeExts = map[string]string{
		".go": "go", ".py": "python", ".js": "c", ".jsx": "c", ".mjs": "c", ".cjs": "c", ".ts": "c", ".tsx": "c",
		".java": "c", ".kt": "c", ".kts": "c", ".cs": "c", ".php": "c", ".rs": "c", ".c": "c", ".h": "c",
		".cpp": "c", ".cc": "c", ".hpp": "c", ".swift": "c", ".scala": "c", ".dart": "c", ".rb": "ruby",
	}
)

// Detect guesses the content kind from an optional path and the content itself.
// tags: compress
func Detect(path, text string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case ext == ".json" || ext == ".jsonl" || ext == ".ndjson":
		return KindJSON
	case ext == ".log":
		return KindLog
	case ext == ".diff" || ext == ".patch":
		return KindDiff
	case codeExts[ext] != "":
		return KindCode
	}
	trim := strings.TrimSpace(text)
	if (strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "[")) && looksJSON(trim) {
		return KindJSON
	}
	if len(diffRE.FindAllStringIndex(text, 3)) >= 2 {
		return KindDiff
	}
	lines := strings.Count(text, "\n") + 1
	if hits := len(logRE.FindAllStringIndex(text, -1)); lines >= 4 && hits*3 >= lines {
		return KindLog
	}
	if strings.HasPrefix(trim, "package ") && strings.Contains(text, "\nfunc ") {
		return KindCode
	}
	return KindText
}

// tags: compress
func looksJSON(s string) bool {
	if json.Valid([]byte(s)) {
		return true
	}
	// JSON Lines or concatenated values, as printed by `go list -json`.
	if len(s) > 4<<20 {
		s = s[:4<<20]
	}
	dec := json.NewDecoder(strings.NewReader(s))
	n := 0
	for {
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return err == io.EOF && n > 1
		}
		n++
	}
}

// tags: compress
func normalizeNewlines(s string) string {
	if !strings.Contains(s, "\r") {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}

// elision renders a marker for original lines a..b (1-based, inclusive).
// tags: compress
func elision(prefix string, a, b int, note string) string {
	n := b - a + 1
	s := fmt.Sprintf("%s… %d line", prefix, n)
	if n != 1 {
		s += "s"
	}
	s += fmt.Sprintf(" elided (L%d-%d)", a, b)
	if note != "" {
		s += " " + note
	}
	return s
}

// fitBudget trims output to an estimated token budget. Signal lines (errors,
// warnings, ACP tags) are reserved first with up to half the budget, then the
// head and tail fill the rest. Elision markers already in the output keep
// referencing original line numbers, so they stay valid.
// tags: compress
func fitBudget(out string, budget int) (string, int) {
	lines := strings.Split(out, "\n")
	limit := budget * 4 * 85 / 100 // leave room for drop markers
	if limit < 200 {
		limit = 200
	}
	keep := make([]bool, len(lines))
	size := 0
	take := func(i int, cap int) bool {
		if keep[i] {
			return true
		}
		if size+len(lines[i])+1 > cap {
			return false
		}
		keep[i] = true
		size += len(lines[i]) + 1
		return true
	}
	for i, ln := range lines {
		if important(ln) {
			take(i, limit/2)
		}
	}
	for i, j := 0, len(lines)-1; i <= j; {
		// Head gets priority until two thirds of the limit are spent.
		if size < limit*2/3 {
			if !take(i, limit) {
				break
			}
			i++
			continue
		}
		if !take(j, limit) {
			break
		}
		j--
	}
	var b bytes.Buffer
	dropped := 0
	for i := 0; i < len(lines); {
		if keep[i] {
			b.WriteString(lines[i] + "\n")
			i++
			continue
		}
		j := i
		for j+1 < len(lines) && !keep[j+1] {
			j++
		}
		dropped += j - i + 1
		fmt.Fprintf(&b, "… %d output lines dropped to fit budget %d\n", j-i+1, budget)
		i = j + 1
	}
	return strings.TrimRight(b.String(), "\n"), dropped
}

var importantRE = regexp.MustCompile(`(?i)\b(error|errors|err|fail|failed|failure|fatal|panic|exception|traceback|critical|denied|refused|timeout|timed out|unauthorized|forbidden|invalid|cannot|can't|unable|undefined|null pointer|segfault|warn|warning|deprecated|assert)\b|@ACP|FAIL`)

// important reports whether a line carries a failure or warning signal.
// tags: compress
func important(s string) bool { return importantRE.MatchString(s) }

// queryHit reports whether a line mentions any task word.
// tags: compress
func queryHit(line string, words []string) bool {
	if len(words) == 0 {
		return false
	}
	low := strings.ToLower(line)
	for _, w := range words {
		if strings.Contains(low, w) {
			return true
		}
	}
	return false
}

// queryStop lists task verbs too generic to select lines by.
var queryStop = map[string]bool{
	"change": true, "make": true, "update": true, "keep": true, "with": true, "from": true, "into": true,
	"that": true, "this": true, "when": true, "only": true, "should": true, "must": true, "remove": true,
	"also": true, "then": true, "while": true, "each": true, "every": true, "after": true, "before": true,
	"add": true, "fix": true, "support": true, "using": true, "have": true, "does": true, "same": true,
}

// queryWords extracts distinctive lowercase task words (length >= 4).
// tags: compress
func queryWords(q string) []string {
	out := []string{}
	seen := map[string]bool{"": true}
	for w := range queryStop {
		seen[w] = true
	}
	for _, w := range strings.FieldsFunc(strings.ToLower(q), func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_')
	}) {
		if len(w) >= 4 && !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	return out
}
