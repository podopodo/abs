package scope

// @ACP O ACP.SCOPE

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/model"
	"github.com/podopodo/abs/internal/textutil"
)

type scored struct {
	path    string
	score   float64
	reasons map[string]bool
}

var closureTerms = map[string][]string{
	"persistence": {"repository", "store", "storage", "serializer", "model", "migration", "database", "json", "sql"},
	"worker":      {"worker", "job", "queue", "retry", "scheduler", "cron", "background"},
	"contract":    {"contract", "schema", "validator", "request", "response", "event", "route", "api"},
	"security":    {"auth", "authorization", "permission", "tenant", "role", "policy", "access"},
	"payment":     {"payment", "billing", "invoice", "refund", "capture", "settlement", "credit"},
	"state":       {"state", "status", "transition", "workflow", "lifecycle", "machine"},
	"interface":   {"html", "template", "component", "css", "style", "form", "frontend", "view"},
	"operations":  {"shell", "deploy", "docker", "workflow", "pipeline", "migration", "infra"},
}

func Build(p *model.Project, cfg config.Config, task string, maxFiles int) model.ScopeResult {
	if maxFiles <= 0 {
		maxFiles = cfg.MaxScopeFiles
	}
	clauses := textutil.Clauses(task)
	query := textutil.Words(task)
	scores := map[string]*scored{}
	add := func(path string, n float64, reason string) {
		if _, ok := p.Files[path]; !ok {
			return
		}
		s := scores[path]
		if s == nil {
			s = &scored{path: path, reasons: map[string]bool{}}
			scores[path] = s
		}
		s.score += n
		s.reasons[reason] = true
	}
	for path, f := range p.Files {
		if !candidateFile(path, f, query) {
			continue
		}
		hit := textutil.Overlap(query, f.Tokens)
		if hit > 0 {
			add(path, float64(hit*2), "task terms")
		}
		pathWords := textutil.Words(path)
		if n := textutil.Overlap(query, pathWords); n > 0 {
			add(path, float64(n*4), "path match")
		}
		for _, sym := range f.Symbols {
			if textutil.Overlap(query, textutil.Words(sym)) > 0 {
				add(path, 3, "symbol match")
			}
		}
	}
	evidences := make([]model.ClauseEvidence, 0, len(clauses))
	conceptOwners := map[string]bool{}
	for _, cl := range clauses {
		cw := textutil.Words(cl)
		type candidate struct {
			id, path string
			score    int
		}
		cand := []candidate{}
		for id, path := range p.Owners {
			f := p.Files[path]
			core := textutil.Words(strings.ReplaceAll(id, ".", " ") + " " + path + " " + strings.Join(f.Symbols, " "))
			sc := textutil.Overlap(cw, core)*5 + textutil.Overlap(cw, f.Tokens)
			if sc > 0 {
				cand = append(cand, candidate{id, path, sc})
			}
		}
		sort.Slice(cand, func(i, j int) bool {
			if cand[i].score == cand[j].score {
				return cand[i].id < cand[j].id
			}
			return cand[i].score > cand[j].score
		})
		ev := model.ClauseEvidence{Clause: cl}
		limit := 2
		if len(cand) < limit {
			limit = len(cand)
		}
		for _, c := range cand[:limit] {
			ev.IDs = append(ev.IDs, c.id)
			ev.Paths = append(ev.Paths, c.path)
			conceptOwners[c.path] = true
			add(c.path, float64(14+c.score), "clause "+short(cl, 42))
		}
		ev.Covered = len(ev.Paths) > 0
		evidences = append(evidences, ev)
	}
	low := strings.ToLower(task)
	for category, hints := range closureTerms {
		active := false
		for _, h := range hints {
			if strings.Contains(low, h) {
				active = true
				break
			}
		}
		if !active {
			continue
		}
		for path, f := range p.Files {
			if !candidateFile(path, f, query) {
				continue
			}
			hay := strings.ToLower(path + " " + strings.Join(f.Symbols, " "))
			for _, h := range hints {
				if strings.Contains(hay, h) {
					add(path, 8, "closure "+category)
					break
				}
			}
		}
	}
	// Close around strongest owners: dependents, guards, imports, callers and local contexts.
	seed := rank(scores)
	seedLimit := 8
	if len(seed) < seedLimit {
		seedLimit = len(seed)
	}
	for _, s := range seed[:seedLimit] {
		for _, d := range p.Files[s.path].Dependencies {
			add(d, 2, "dependency of "+s.path)
		}
		for _, r := range p.ImportedBy[s.path] {
			add(r, 2, "caller of "+s.path)
		}
		for id, op := range p.Owners {
			if op == s.path {
				for _, d := range p.Dependents[id] {
					add(d, 4, "dependent "+id)
				}
				for _, g := range p.Guards[id] {
					add(g, 6, "guard "+id)
				}
			}
		}
	}
	complexity := len(clauses) + len(conceptOwners)
	if complexity >= 10 && maxFiles < 24 {
		maxFiles = 24
	} else if complexity >= 7 && maxFiles < 20 {
		maxFiles = 20
	}
	ranked := rank(scores)
	selected := []*scored{}
	for _, s := range ranked {
		if len(selected) >= maxFiles {
			break
		}
		selected = append(selected, s)
	}
	// Owners resolved from clauses must survive cap.
	present := map[string]bool{}
	for _, s := range selected {
		present[s.path] = true
	}
	for owner := range conceptOwners {
		if present[owner] {
			continue
		}
		if len(selected) < maxFiles {
			selected = append(selected, scores[owner])
		} else {
			replace := len(selected) - 1
			for replace >= 0 && conceptOwners[selected[replace].path] {
				replace--
			}
			if replace >= 0 {
				selected[replace] = scores[owner]
			}
		}
		present[owner] = true
	}
	// Keep at least one relevant guard when tags exist.
	hasGuard := false
	for _, s := range selected {
		for _, gs := range p.Guards {
			for _, g := range gs {
				if g == s.path {
					hasGuard = true
				}
			}
		}
	}
	if !hasGuard {
		for id, owner := range p.Owners {
			if !present[owner] {
				continue
			}
			if gs := p.Guards[id]; len(gs) > 0 {
				g := gs[0]
				if len(selected) < maxFiles {
					selected = append(selected, scoresOr(scores, g))
				} else if len(selected) > 0 {
					selected[len(selected)-1] = scoresOr(scores, g)
				}
				break
			}
		}
	}
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].score == selected[j].score {
			return selected[i].path < selected[j].path
		}
		return selected[i].score > selected[j].score
	})
	result := model.ScopeResult{Task: task, Clauses: evidences}
	used := map[string]bool{}
	for _, s := range selected {
		if s == nil || used[s.path] {
			continue
		}
		used[s.path] = true
		rs := []string{}
		for r := range s.reasons {
			rs = append(rs, r)
		}
		sort.Strings(rs)
		result.Files = append(result.Files, model.ScopeItem{Path: s.path, Score: s.score, Reasons: rs})
	}
	result.Contexts = contextChain(p, cfg, result.Files)
	for _, e := range evidences {
		if !e.Covered {
			result.Warnings = append(result.Warnings, "unresolved clause: "+e.Clause)
		}
	}
	return result
}

func candidateFile(path string, f *model.File, query map[string]int) bool {
	base := filepath.Base(path)
	if base == "CONTEXT.md" || path == "PROTOCOL.md" || path == ".agent/STATE.md" {
		return false
	}
	if f.Kind == "docs" {
		for _, w := range []string{"doc", "docs", "documentation", "readme", "changelog"} {
			if query[w] > 0 {
				return true
			}
		}
		return false
	}
	return true
}

func scoresOr(m map[string]*scored, path string) *scored {
	if s := m[path]; s != nil {
		return s
	}
	return &scored{path: path, score: 1, reasons: map[string]bool{"guard closure": true}}
}
func rank(m map[string]*scored) []*scored {
	out := make([]*scored, 0, len(m))
	for _, s := range m {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].score == out[j].score {
			return out[i].path < out[j].path
		}
		return out[i].score > out[j].score
	})
	return out
}
func short(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func contextChain(p *model.Project, cfg config.Config, items []model.ScopeItem) []string {
	set := map[string]bool{}
	if _, ok := p.Files[cfg.ContextFile]; ok {
		set[cfg.ContextFile] = true
	}
	for _, it := range items {
		dir := filepath.Dir(filepath.FromSlash(it.Path))
		for dir != "." && dir != "" {
			rel := filepath.ToSlash(filepath.Join(dir, cfg.ContextFile))
			if _, ok := p.Files[rel]; ok {
				set[rel] = true
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	out := []string{}
	for x := range set {
		out = append(out, x)
	}
	sort.Strings(out)
	return out
}

func PrintText(w io.Writer, r model.ScopeResult) {
	fmt.Fprintln(w, "TASK "+r.Task)
	for _, c := range r.Clauses {
		status := "MISS"
		if c.Covered {
			status = "OK"
		}
		fmt.Fprintf(w, "C %s %s", status, c.Clause)
		if len(c.IDs) > 0 {
			fmt.Fprintf(w, " -> %s", strings.Join(c.IDs, ","))
		}
		fmt.Fprintln(w)
	}
	for _, c := range r.Contexts {
		fmt.Fprintln(w, "CFILE "+c)
	}
	for _, f := range r.Files {
		fmt.Fprintf(w, "F %s", f.Path)
		if len(f.Reasons) > 0 {
			fmt.Fprintf(w, " # %s", strings.Join(f.Reasons, "; "))
		}
		fmt.Fprintln(w)
	}
	for _, warning := range r.Warnings {
		fmt.Fprintln(w, "WARN "+warning)
	}
}
