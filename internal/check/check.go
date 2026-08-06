package check

// @ACP O ACP.CHECK

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/model"
)

type Options struct {
	Strict  bool
	Changed bool
}

func Run(p *model.Project, cfg config.Config, opt Options) model.CheckResult {
	findings := []model.Finding{}
	add := func(sev, code, path string, line int, msg string) {
		findings = append(findings, model.Finding{Severity: sev, Code: code, Path: path, Line: line, Message: msg})
	}
	for id, paths := range p.OwnerLists {
		if len(paths) > 1 {
			add("error", "duplicate-owner", "", 0, fmt.Sprintf("%s has owners %s", id, strings.Join(paths, ", ")))
		}
	}
	for id, paths := range p.Dependents {
		if _, ok := p.Owners[id]; !ok {
			add("error", "missing-owner", "", 0, fmt.Sprintf("%s used by %s", id, strings.Join(paths, ", ")))
		}
	}
	for id, path := range p.Owners {
		if len(p.Guards[id]) == 0 {
			add("error", "missing-guard", path, 0, "owner "+id+" has no @ACP G mapping")
		}
	}
	changed := map[string]bool{}
	if opt.Changed {
		for _, x := range gitChanged(p.Root) {
			changed[x] = true
		}
	}
	include := func(path string) bool {
		return !opt.Changed || changed[path] || path == cfg.ContextFile || path == filepath.ToSlash(cfg.StateFile)
	}
	for path, f := range p.Files {
		if !include(path) {
			continue
		}
		for _, risk := range f.Risks {
			if risk.Severity == "warning" && !opt.Strict {
				continue
			}
			findings = append(findings, risk)
		}
		if f.Kind == "go" {
			if _, err := parser.ParseFile(token.NewFileSet(), filepath.Join(p.Root, filepath.FromSlash(path)), nil, parser.AllErrors); err != nil {
				add("error", "go-syntax", path, 0, err.Error())
			}
		}
		validateDependencies(p, f, &findings, opt.Strict)
	}
	for path, f := range p.Files {
		if filepath.Base(path) != cfg.ContextFile {
			continue
		}
		lines := nonemptyLines(f.Text)
		if len(lines) != 4 || !hasPrefixes(lines, []string{"P:", "R:", "B:", "X:"}) {
			add("error", "bad-context", path, 0, "CONTEXT.md must contain exactly P/R/B/X non-empty lines")
		}
	}
	statePath := filepath.ToSlash(cfg.StateFile)
	if f := p.Files[statePath]; f != nil {
		t := strings.TrimSpace(f.Text)
		if t != "" && strings.Contains(strings.ToUpper(t), "DONE") {
			add("error", "stale-state", statePath, 0, "completed STATE must be empty or removed")
		}
	}
	checkEvents(p, &findings, opt)
	checkDuplicates(p, &findings, opt)
	checkHTMLCSS(p, &findings, opt.Strict)
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity < findings[j].Severity
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Code < findings[j].Code
	})
	errors := 0
	warnings := 0
	for _, f := range findings {
		if f.Severity == "error" {
			errors++
		} else {
			warnings++
		}
	}
	return model.CheckResult{Passed: errors == 0, Findings: findings, Stats: map[string]int{"files": len(p.Files), "owners": len(p.Owners), "guards": len(p.Guards), "errors": errors, "warnings": warnings}}
}

func nonemptyLines(s string) []string {
	out := []string{}
	sc := bufio.NewScanner(strings.NewReader(s))
	for sc.Scan() {
		x := strings.TrimSpace(sc.Text())
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}
func hasPrefixes(lines, prefix []string) bool {
	if len(lines) != len(prefix) {
		return false
	}
	for i, p := range prefix {
		if !strings.HasPrefix(lines[i], p) {
			return false
		}
	}
	return true
}

func validateDependencies(p *model.Project, f *model.File, findings *[]model.Finding, strict bool) {
	if !strict {
		return
	}
	for _, d := range f.Dependencies {
		if _, ok := p.Files[d]; !ok {
			*findings = append(*findings, model.Finding{Severity: "warning", Code: "missing-local-dependency", Path: f.Path, Message: "referenced local file not found: " + d})
		}
	}
}

func checkEvents(p *model.Project, findings *[]model.Finding, opt Options) {
	allowed := map[string]bool{}
	constRE := regexp.MustCompile(`(?m)(?:^[A-Z][A-Z0-9_]*\s*=\s*|\bconst\s+[A-Za-z_$][\w$]*\s*=\s*)["']([^"']+)["']`)
	for path, f := range p.Files {
		low := strings.ToLower(path)
		if strings.Contains(low, "contract") || strings.Contains(low, "event") || strings.Contains(low, "schema") {
			for _, m := range constRE.FindAllStringSubmatch(f.Text, -1) {
				allowed[m[1]] = true
			}
		}
	}
	if len(allowed) == 0 {
		return
	}
	emitRE := regexp.MustCompile(`\b(?:publish|emit|dispatch|notify)\s*\(\s*["']([^"']+)["']`)
	for path, f := range p.Files {
		for _, m := range emitRE.FindAllStringSubmatchIndex(f.Text, -1) {
			name := f.Text[m[2]:m[3]]
			if !allowed[name] {
				*findings = append(*findings, model.Finding{Severity: "error", Code: "raw-event", Path: path, Line: 1 + strings.Count(f.Text[:m[0]], "\n"), Message: "unknown literal boundary event: " + name})
			}
		}
	}
}

func checkDuplicates(p *model.Project, findings *[]model.Finding, opt Options) {
	type loc struct{ path, name string }
	bodies := map[[20]byte][]loc{}
	for path, f := range p.Files {
		if opt.Changed { /* global duplicate ownership still useful; body check remains global intentionally */
		}
		for _, b := range functionBodies(f.Kind, f.Text) {
			if len(b.body) < 80 {
				continue
			}
			norm := regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(b.body), " ")
			sum := sha1.Sum([]byte(norm))
			bodies[sum] = append(bodies[sum], loc{path, b.name})
		}
	}
	for _, ls := range bodies {
		if len(ls) < 2 {
			continue
		}
		parts := []string{}
		for _, l := range ls {
			parts = append(parts, l.path+":"+l.name)
		}
		*findings = append(*findings, model.Finding{Severity: "warning", Code: "duplicate-body", Message: "similar function bodies: " + strings.Join(parts, ", ")})
	}
}

type body struct{ name, body string }

func functionBodies(kind, text string) []body {
	out := []body{}
	if kind == "python" {
		re := regexp.MustCompile(`(?m)^(\s*)(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)[^\n]*:\s*$`)
		ms := re.FindAllStringSubmatchIndex(text, -1)
		for i, m := range ms {
			start := m[1]
			end := len(text)
			if i+1 < len(ms) {
				end = ms[i+1][0]
			}
			out = append(out, body{text[m[4]:m[5]], text[start:end]})
		}
		return out
	}
	re := regexp.MustCompile(`(?m)(?:func\s+(?:\([^)]*\)\s*)?|function\s+|(?:public|private|protected|static|async|export|\s)+)([A-Za-z_$][\w$]*)\s*\([^;{]*\)\s*\{`)
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		name := text[m[2]:m[3]]
		if name == "func" {
			continue
		}
		start := m[0]
		open := strings.Index(text[start:m[1]], "{") + start
		if open < start {
			continue
		}
		depth := 0
		end := -1
		for i := open; i < len(text); i++ {
			switch text[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i + 1
					i = len(text)
				}
			}
		}
		if end > open {
			out = append(out, body{name, text[start:end]})
		}
	}
	return out
}

func checkHTMLCSS(p *model.Project, findings *[]model.Finding, strict bool) {
	if !strict {
		return
	}
	htmlClasses := map[string]bool{}
	cssClasses := map[string][]string{}
	classAttr := regexp.MustCompile(`(?i)\bclass\s*=\s*["']([^"']+)["']`)
	selector := regexp.MustCompile(`(?m)\.([A-Za-z_][A-Za-z0-9_-]*)`)
	for path, f := range p.Files {
		if f.Kind == "html" || f.Kind == "template" {
			for _, m := range classAttr.FindAllStringSubmatch(f.Text, -1) {
				for _, c := range strings.Fields(m[1]) {
					if !strings.ContainsAny(c, "{}:$") {
						htmlClasses[c] = true
					}
				}
			}
		}
		if f.Kind == "css" {
			clean := regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(f.Text, "")
			for _, m := range selector.FindAllStringSubmatch(clean, -1) {
				cssClasses[m[1]] = append(cssClasses[m[1]], path)
			}
		}
	}
	for c, paths := range cssClasses {
		if !htmlClasses[c] && !strings.HasPrefix(c, "js-") {
			*findings = append(*findings, model.Finding{Severity: "warning", Code: "unused-css-class", Path: paths[0], Message: "selector not found in scanned templates: ." + c})
		}
	}
}

func gitChanged(root string) []string {
	cmd := exec.Command("git", "-C", root, "diff", "--name-only", "--diff-filter=ACMR", "HEAD")
	b, err := cmd.Output()
	if err != nil {
		return nil
	}
	out := []string{}
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		out = append(out, filepath.ToSlash(strings.TrimSpace(sc.Text())))
	}
	return out
}

func PrintText(w io.Writer, r model.CheckResult) {
	for _, f := range r.Findings {
		loc := f.Path
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", loc, f.Line)
		}
		fmt.Fprintf(w, "%s %s %s %s\n", strings.ToUpper(f.Severity), f.Code, loc, f.Message)
	}
	if r.Passed {
		fmt.Fprintf(w, "ACP CHECK PASS files=%d owners=%d guards=%d warnings=%d\n", r.Stats["files"], r.Stats["owners"], r.Stats["guards"], r.Stats["warnings"])
	} else {
		fmt.Fprintf(w, "ACP CHECK FAIL errors=%d warnings=%d\n", r.Stats["errors"], r.Stats["warnings"])
	}
}

var _ = os.ErrNotExist
