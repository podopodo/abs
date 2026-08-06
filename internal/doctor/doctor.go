package doctor

// @ACP O ACP.DOCTOR

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sort"

	"github.com/podopodo/abs/internal/model"
)

type Tool struct {
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Available bool   `json:"available"`
	Purpose   string `json:"purpose"`
}
type Result struct {
	OS    string         `json:"os"`
	Arch  string         `json:"arch"`
	Root  string         `json:"root"`
	Files int            `json:"files"`
	Kinds map[string]int `json:"kinds"`
	Tools []Tool         `json:"tools"`
}

func Run(p *model.Project) Result {
	tools := []Tool{}
	for _, x := range []struct{ name, purpose string }{{"git", "changed-file graph"}, {"rg", "faster text search"}, {"shellcheck", "shell validation"}, {"stylelint", "CSS validation"}, {"eslint", "JS/TS validation"}, {"htmlhint", "HTML validation"}, {"sqlfluff", "SQL validation"}, {"hadolint", "Dockerfile validation"}} {
		path, err := exec.LookPath(x.name)
		tools = append(tools, Tool{Name: x.name, Path: path, Available: err == nil, Purpose: x.purpose})
	}
	kinds := map[string]int{}
	for _, f := range p.Files {
		kinds[f.Kind]++
	}
	return Result{OS: runtime.GOOS, Arch: runtime.GOARCH, Root: p.Root, Files: len(p.Files), Kinds: kinds, Tools: tools}
}

func Print(w io.Writer, r Result) {
	fmt.Fprintf(w, "ACP %s/%s root=%s files=%d\n", r.OS, r.Arch, r.Root, r.Files)
	ks := []string{}
	for k := range r.Kinds {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	for _, k := range ks {
		fmt.Fprintf(w, "KIND %-12s %d\n", k, r.Kinds[k])
	}
	for _, t := range r.Tools {
		status := "missing"
		if t.Available {
			status = t.Path
		}
		fmt.Fprintf(w, "TOOL %-12s %-24s # %s\n", t.Name, status, t.Purpose)
	}
	fmt.Fprintln(w, "Core scanners are embedded; optional tools improve precision but are not required.")
}
