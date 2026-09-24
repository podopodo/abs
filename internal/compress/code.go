package compress

// @ACP D ACP.COMPRESS

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	bodyKeep    = 3  // bodies with at most this many interior lines stay whole
	literalKeep = 12 // long var/const blocks keep this many leading lines
)

// compressCode turns source into an outline: package, imports, types,
// signatures, doc comments and ACP tags stay; long function bodies become
// elision markers with original line ranges. Lines inside bodies that match
// the task query survive so the relevant statement is still visible.
// tags: compress, code_outline
func compressCode(text string, opt Options) (string, int) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	lang := codeExts[strings.ToLower(filepath.Ext(opt.Path))]
	if lang == "" && strings.HasPrefix(strings.TrimSpace(text), "package ") {
		lang = "go"
	}
	var drop []bool
	switch lang {
	case "go":
		drop = goBodies(text, len(lines))
		if drop == nil {
			drop = braceBodies(lines)
		}
	case "python":
		drop = indentBodies(lines, pyDefRE, false)
	case "ruby":
		drop = indentBodies(lines, rbDefRE, true)
	default:
		drop = braceBodies(lines)
	}
	words := queryWords(opt.Query)
	keep := make([]bool, len(lines))
	for i := range lines {
		keep[i] = !drop[i] || queryHit(lines[i], words) || strings.Contains(lines[i], "@ACP")
	}
	prefix := "\t// "
	if lang == "python" || lang == "ruby" {
		prefix = "    # "
	}
	return renderKept(lines, keep, prefix, nil)
}

// goBodies marks interior lines of long Go function bodies and long
// var/const blocks. It returns nil when the file does not parse.
// tags: compress, code_outline
func goBodies(text string, n int) []bool {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", text, parser.ParseComments)
	if err != nil {
		return nil
	}
	drop := make([]bool, n)
	span := func(a, b int) {
		for l := a; l <= b && l <= n; l++ {
			drop[l-1] = true
		}
	}
	for _, d := range f.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			if x.Body == nil {
				continue
			}
			a, b := fset.Position(x.Body.Lbrace).Line+1, fset.Position(x.Body.Rbrace).Line-1
			if b-a+1 > bodyKeep {
				span(a, b)
			}
		case *ast.GenDecl:
			if x.Tok != token.VAR && x.Tok != token.CONST {
				continue
			}
			a, b := fset.Position(x.Pos()).Line, fset.Position(x.End()).Line
			if b-a+1 > literalKeep+2 {
				span(a+literalKeep, b-1)
			}
		}
	}
	return drop
}

var (
	containerRE = regexp.MustCompile(`\b(class|interface|namespace|enum|struct|impl|trait|object|module|record|protocol|extension|package|union)\b`)
	pyDefRE     = regexp.MustCompile(`^(\s*)(async\s+def|def)\s`)
	rbDefRE     = regexp.MustCompile(`^(\s*)def\s`)
	strRE       = regexp.MustCompile(`"(\\.|[^"\\])*"|'(\\.|[^'\\])*'|` + "`[^`]*`")
)

// braceBodies outlines brace languages. Blocks opened by a container
// declaration (class, interface, namespace…) keep their members; any other
// block is a body whose interior is dropped when longer than bodyKeep.
// tags: compress, code_outline
func braceBodies(lines []string) []bool {
	drop := make([]bool, len(lines))
	type block struct {
		container bool
		start     int // first interior line index
	}
	stack := []block{}
	inBody := func() bool {
		for _, b := range stack {
			if !b.container {
				return true
			}
		}
		return false
	}
	for i, raw := range lines {
		ln := strRE.ReplaceAllString(raw, `""`)
		if k := strings.Index(ln, "//"); k >= 0 {
			ln = ln[:k]
		}
		if inBody() {
			drop[i] = true
		}
		for _, r := range ln {
			switch r {
			case '{':
				stack = append(stack, block{container: !inBody() && containerRE.MatchString(ln), start: i + 1})
			case '}':
				if len(stack) == 0 {
					continue
				}
				top := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if !top.container && !inBody() {
					// Closing an outermost body: keep the closing line and
					// restore short bodies whole.
					drop[i] = false
					if i-top.start <= bodyKeep {
						for k := top.start; k < i; k++ {
							drop[k] = false
						}
					}
				}
			}
		}
	}
	return drop
}

// indentBodies outlines indentation languages: a def's deeper-indented body
// is dropped, except a one-line docstring. Ruby keeps the closing `end`.
// tags: compress, code_outline
func indentBodies(lines []string, defRE *regexp.Regexp, ruby bool) []bool {
	drop := make([]bool, len(lines))
	indent := func(s string) int { return len(s) - len(strings.TrimLeft(s, " \t")) }
	for i := 0; i < len(lines); i++ {
		m := defRE.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		base := len(m[1])
		// Skip continuation of a multi-line signature.
		j := i
		for j < len(lines) && !strings.HasSuffix(strings.TrimSpace(lines[j]), ":") && !ruby && j-i < 10 {
			j++
		}
		start := j + 1
		end := start
		for end < len(lines) {
			t := strings.TrimSpace(lines[end])
			if t != "" && indent(lines[end]) <= base {
				break
			}
			end++
		}
		// end is the first line outside the body; trim trailing blanks.
		last := end - 1
		for last >= start && strings.TrimSpace(lines[last]) == "" {
			last--
		}
		from := start
		if from <= last {
			t := strings.TrimSpace(lines[from])
			if strings.HasPrefix(t, `"""`) || strings.HasPrefix(t, `'''`) || strings.HasPrefix(t, "#") {
				from++
			}
		}
		if last-from+1 > bodyKeep {
			for k := from; k <= last; k++ {
				drop[k] = true
			}
		}
		// Nested defs inside a dropped body are already covered.
		if last > i {
			i = last
		}
	}
	return drop
}
