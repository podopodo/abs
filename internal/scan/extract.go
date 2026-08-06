package scan

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/podopodo/abs/internal/model"
	"github.com/podopodo/abs/internal/textutil"
)

var tagRE = regexp.MustCompile(`@ACP\s+([ODG])\s+([A-Z][A-Z0-9_.+-]*)`)
var quotedRE = regexp.MustCompile(`["']([^"']+)["']`)

func lineAt(text string, offset int) int { return 1 + strings.Count(text[:offset], "\n") }

func extractTags(text string) []model.Tag {
	out := []model.Tag{}
	for i, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !isCommentLine(trimmed) {
			continue
		}
		for _, m := range tagRE.FindAllStringSubmatch(trimmed, -1) {
			role := model.TagRole(m[1])
			ids := strings.Split(m[2], "+")
			out = append(out, model.Tag{Role: role, IDs: ids, Line: i + 1})
		}
	}
	return out
}

func isCommentLine(s string) bool {
	prefixes := []string{"//", "#", "/*", "*", "<!--", "--", ";", "REM ", "rem "}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func addMatches(text string, kind string, patterns ...*regexp.Regexp) []model.Entry {
	out := []model.Entry{}
	seen := map[string]bool{}
	for _, re := range patterns {
		for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
			if len(m) < 4 || m[2] < 0 {
				continue
			}
			name := strings.TrimSpace(text[m[2]:m[3]])
			key := kind + ":" + name
			if name == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, model.Entry{Name: name, Kind: kind, Line: lineAt(text, m[0])})
		}
	}
	return out
}

var (
	pyFunc      = regexp.MustCompile(`(?m)^\s*(?:async\s+)?def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	goFunc      = regexp.MustCompile(`(?m)^func\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	jsFunc      = regexp.MustCompile(`(?m)(?:function\s+([A-Za-z_$][\w$]*)\s*\(|(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s*)?\([^)]*\)\s*=>)`)
	classRE     = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:abstract\s+)?class\s+([A-Za-z_$][\w$]*)`)
	phpFunc     = regexp.MustCompile(`(?m)^\s*(?:public|protected|private|static|final|abstract|\s)*function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	rubyFunc    = regexp.MustCompile(`(?m)^\s*def\s+([A-Za-z_][A-Za-z0-9_!?=]*)`)
	shellFunc   = regexp.MustCompile(`(?m)^\s*(?:function\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*\(\)\s*\{`)
	routeRE     = regexp.MustCompile(`(?m)(?:route|router|app)\s*\.\s*(?:get|post|put|patch|delete|use)\s*\(\s*["']([^"']+)`)
	htmlID      = regexp.MustCompile(`(?i)\bid\s*=\s*["']([^"']+)["']`)
	htmlForm    = regexp.MustCompile(`(?i)<form\b[^>]*\baction\s*=\s*["']([^"']*)["']`)
	cssCustom   = regexp.MustCompile(`(?m)(--[A-Za-z0-9_-]+)\s*:`)
	cssKeyframe = regexp.MustCompile(`(?i)@keyframes\s+([A-Za-z_][\w-]*)`)
	sqlTable    = regexp.MustCompile(`(?i)\b(?:create|alter)\s+table\s+(?:if\s+not\s+exists\s+)?["` + "`" + `]?([A-Za-z_][\w.]*)`)
	gqlType     = regexp.MustCompile(`(?m)^\s*(?:type|input|interface|enum|scalar|union)\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

func extractEntries(kind, text string) []model.Entry {
	switch kind {
	case "python":
		return append(addMatches(text, "function", pyFunc), addMatches(text, "route", routeRE)...)
	case "go":
		return addMatches(text, "function", goFunc)
	case "javascript", "typescript", "template":
		out := addMatches(text, "function", jsFunc)
		out = append(out, addMatches(text, "class", classRE)...)
		out = append(out, addMatches(text, "route", routeRE)...)
		return out
	case "php":
		return addMatches(text, "function", phpFunc)
	case "ruby":
		return addMatches(text, "function", rubyFunc)
	case "shell":
		return addMatches(text, "function", shellFunc)
	case "html":
		out := addMatches(text, "form", htmlForm)
		out = append(out, addMatches(text, "element", htmlID)...)
		return out
	case "css":
		out := addMatches(text, "custom-property", cssCustom)
		out = append(out, addMatches(text, "keyframe", cssKeyframe)...)
		return out
	case "sql":
		return addMatches(text, "table", sqlTable)
	case "graphql":
		return addMatches(text, "type", gqlType)
	default:
		return append(addMatches(text, "function", pyFunc, goFunc, phpFunc, rubyFunc, shellFunc), addMatches(text, "route", routeRE)...)
	}
}

var importPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s*(?:source|\.)\s+["\']?\$\([^)]*\)/([^"\']+)`),
	regexp.MustCompile(`(?m)^\s*(?:from\s+[^\s]+\s+)?import\s+(?:[^"'\n]+\s+from\s+)?["']([^"']+)["']`),
	regexp.MustCompile(`(?m)\brequire\s*\(\s*["']([^"']+)["']\s*\)`),
	regexp.MustCompile(`(?m)^\s*#?include\s*[<"]([^>"]+)[>"]`),
	regexp.MustCompile(`(?m)^\s*(?:source|\.)\s+["']?([^\s"']+)`),
	regexp.MustCompile(`(?i)(?:src|href|action)\s*=\s*["']([^"'#]+)["']`),
	regexp.MustCompile(`(?i)(?:include|extends|import)\s+["']([^"']+)["']`),
	regexp.MustCompile(`(?i)@(?:import|use|forward)\s+(?:url\()?\s*["']([^"']+)["']`),
	regexp.MustCompile(`(?i)\bREFERENCES\s+([A-Za-z_][\w.]*)`),
}

func extractDependencies(text string) []string {
	out := []string{}
	for _, re := range importPatterns {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			if len(m) > 1 {
				out = append(out, strings.TrimSpace(m[1]))
			}
		}
	}
	sort.Strings(out)
	return textutil.Unique(out)
}

func extractSymbols(entries []model.Entry, tags []model.Tag, text string) []string {
	out := []string{}
	for _, e := range entries {
		out = append(out, e.Name)
	}
	for _, t := range tags {
		out = append(out, t.IDs...)
	}
	for _, m := range regexp.MustCompile(`\b[A-Z][A-Z0-9_]{2,}\b`).FindAllString(text, -1) {
		out = append(out, m)
	}
	sort.Strings(out)
	return textutil.Unique(out)
}

func extractRisks(kind, path, text string) []model.Finding {
	out := []model.Finding{}
	add := func(sev, code, msg string, re *regexp.Regexp) {
		for _, m := range re.FindAllStringIndex(text, -1) {
			out = append(out, model.Finding{Severity: sev, Code: code, Path: path, Line: lineAt(text, m[0]), Message: msg})
		}
	}
	add("warning", "swallowed-error", "possible swallowed broad error", regexp.MustCompile(`(?ms)(?:except\s+(?:Exception|BaseException)\s*:\s*(?:pass|return\s+None)|catch\s*\([^)]*(?:Exception|Error)[^)]*\)\s*\{\s*\})`))
	if kind == "shell" {
		add("error", "shell-curl-pipe", "remote script piped directly to shell", regexp.MustCompile(`(?m)\bcurl\b[^\n|]*\|\s*(?:sudo\s+)?(?:sh|bash)\b`))
		add("error", "shell-rm-variable", "recursive deletion uses an unguarded variable", regexp.MustCompile(`(?m)\brm\s+-[^\n]*r[^\n]*f[^\n]*\s+\$\{?[A-Za-z_]`))
		add("warning", "shell-unquoted-var", "possibly unquoted shell variable", regexp.MustCompile(`(?m)(?:^|[ ;|])\$[A-Za-z_][A-Za-z0-9_]*(?:[ ;|]|$)`))
	}
	if kind == "html" || kind == "template" {
		add("warning", "html-inline-handler", "inline event handler increases coupling", regexp.MustCompile(`(?i)\son(?:click|change|submit|load|error)\s*=`))
	}
	if kind == "css" {
		add("warning", "css-important", "!important may bypass the canonical style path", regexp.MustCompile(`(?i)!important`))
	}
	if strings.Contains(strings.ToLower(filepath.Base(path)), "dockerfile") || kind == "docker" {
		add("warning", "docker-latest", "unpinned latest container image", regexp.MustCompile(`(?im)^\s*FROM\s+[^\s]+:latest\b`))
	}
	return out
}

func resolveDependency(from, dep string, all map[string]*model.File) string {
	dep = strings.TrimSpace(strings.Split(dep, "?")[0])
	if dep == "" || strings.Contains(dep, "://") || strings.HasPrefix(dep, "#") {
		return ""
	}
	dep = strings.TrimPrefix(dep, "./")
	base := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.Dir(from), dep)))
	cands := []string{base, base + ".go", base + ".py", base + ".js", base + ".ts", base + ".tsx", base + ".jsx", base + ".css", base + ".scss", base + ".html", filepath.ToSlash(filepath.Join(base, "index.ts")), filepath.ToSlash(filepath.Join(base, "index.js"))}
	for _, c := range cands {
		if _, ok := all[c]; ok {
			return c
		}
	}
	return ""
}

func compactReason(kind string, value any) string { return fmt.Sprintf("%s:%v", kind, value) }
