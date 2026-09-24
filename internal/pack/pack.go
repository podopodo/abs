package pack

// @ACP O ACP.PACK

import (
	"fmt"
	"io"
	"strings"

	"github.com/podopodo/abs/internal/compress"
	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/model"
	"github.com/podopodo/abs/internal/scope"
)

// minPartial is the smallest remaining budget worth spending on a partial file.
const minPartial = 150

// File is one scoped file as packed into the capsule.
type File struct {
	Path             string   `json:"path"`
	Kind             string   `json:"kind"`
	Reasons          []string `json:"reasons,omitempty"`
	OriginalTokens   int      `json:"original_tokens"`
	CompressedTokens int      `json:"compressed_tokens"`
	Content          string   `json:"content,omitempty"`
	Skipped          bool     `json:"skipped,omitempty"`
}

// Result is a scope capsule with the evidence itself, compressed to a budget.
type Result struct {
	Task           string                 `json:"task"`
	Budget         int                    `json:"budget"`
	UsedTokens     int                    `json:"used_tokens"`
	OriginalTokens int                    `json:"original_tokens"`
	Clauses        []model.ClauseEvidence `json:"clauses"`
	Contexts       []File                 `json:"contexts,omitempty"`
	Files          []File                 `json:"files"`
	Warnings       []string               `json:"warnings,omitempty"`
}

// Build scopes a task, then inlines context files verbatim and each scoped
// file in rank order as a task-aware compressed outline until the estimated
// token budget is spent. Files that do not fit are listed as paths only.
// tags: pack, compress
func Build(p *model.Project, cfg config.Config, task string, maxFiles, budget int) Result {
	if budget <= 0 {
		budget = cfg.PackBudget
	}
	s := scope.Build(p, cfg, task, maxFiles)
	r := Result{Task: task, Budget: budget, Clauses: s.Clauses, Warnings: s.Warnings}
	for _, c := range s.Contexts {
		f := p.Files[c]
		if f == nil {
			continue
		}
		t := compress.EstimateTokens(f.Text)
		r.Contexts = append(r.Contexts, File{Path: c, Kind: "context", OriginalTokens: t, CompressedTokens: t, Content: strings.TrimRight(f.Text, "\n")})
		r.UsedTokens += t
		r.OriginalTokens += t
	}
	for _, it := range s.Files {
		f := p.Files[it.Path]
		if f == nil {
			continue
		}
		pf := File{Path: it.Path, Reasons: it.Reasons}
		left := budget - r.UsedTokens
		c := compress.Compress([]byte(f.Text), compress.Options{Path: it.Path, Query: task})
		if c.CompressedTokens > left {
			if left < minPartial {
				pf.Kind, pf.Skipped = c.Kind, true
				pf.OriginalTokens = c.OriginalTokens
				r.OriginalTokens += c.OriginalTokens
				r.Files = append(r.Files, pf)
				continue
			}
			c = compress.Compress([]byte(f.Text), compress.Options{Path: it.Path, Query: task, Budget: left})
		}
		pf.Kind = c.Kind
		pf.OriginalTokens, pf.CompressedTokens = c.OriginalTokens, c.CompressedTokens
		pf.Content = c.Output
		r.UsedTokens += c.CompressedTokens
		r.OriginalTokens += c.OriginalTokens
		r.Files = append(r.Files, pf)
	}
	return r
}

// PrintText writes the capsule in ACP's line-oriented style.
// tags: pack
func PrintText(w io.Writer, r Result) {
	fmt.Fprintln(w, "TASK "+r.Task)
	saved := 0
	if r.OriginalTokens > 0 {
		saved = 100 - r.UsedTokens*100/r.OriginalTokens
	}
	fmt.Fprintf(w, "BUDGET %d/%d est-tokens; evidence %d -> %d (-%d%%)\n", r.UsedTokens, r.Budget, r.OriginalTokens, r.UsedTokens, saved)
	fmt.Fprintln(w, "NOTE elisions cite original line numbers; read that file range when a body matters")
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
		fmt.Fprintf(w, "\n== CFILE %s\n%s\n", c.Path, c.Content)
	}
	for _, f := range r.Files {
		why := ""
		if len(f.Reasons) > 0 {
			why = " # " + strings.Join(f.Reasons, "; ")
		}
		if f.Skipped {
			fmt.Fprintf(w, "\n== SKIP %s %s ~%d tokens; over budget%s\n", f.Path, f.Kind, f.OriginalTokens, why)
			continue
		}
		fmt.Fprintf(w, "\n== F %s %s %d->%d%s\n%s\n", f.Path, f.Kind, f.OriginalTokens, f.CompressedTokens, why, f.Content)
	}
	for _, warning := range r.Warnings {
		fmt.Fprintln(w, "WARN "+warning)
	}
}
