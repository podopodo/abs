package query

// @ACP O ACP.QUERY

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/podopodo/abs/internal/model"
)

func Graph(w io.Writer, p *model.Project, id string) error {
	id = strings.ToUpper(id)
	owner, ok := p.Owners[id]
	if !ok {
		return fmt.Errorf("unknown SOT ID %s", id)
	}
	fmt.Fprintln(w, "ID "+id)
	fmt.Fprintln(w, "O "+owner)
	for _, x := range p.Dependents[id] {
		fmt.Fprintln(w, "D "+x)
	}
	for _, x := range p.Guards[id] {
		fmt.Fprintln(w, "G "+x)
	}
	if f := p.Files[owner]; f != nil {
		for _, x := range f.Dependencies {
			fmt.Fprintln(w, "IMPORT "+x)
		}
		for _, x := range p.ImportedBy[owner] {
			fmt.Fprintln(w, "CALLER "+x)
		}
	}
	return nil
}

func Entries(w io.Writer, p *model.Project, id string) error {
	id = strings.ToUpper(id)
	paths := []string{}
	if o := p.Owners[id]; o != "" {
		paths = append(paths, o)
	}
	paths = append(paths, p.Dependents[id]...)
	paths = unique(paths)
	if len(paths) == 0 {
		return fmt.Errorf("unknown SOT ID %s", id)
	}
	for _, path := range paths {
		f := p.Files[path]
		if f == nil {
			continue
		}
		for _, e := range f.Entries {
			fmt.Fprintf(w, "E %s %s:%d %s\n", e.Kind, path, e.Line, e.Name)
		}
	}
	return nil
}

func Guards(w io.Writer, p *model.Project, id string) error {
	id = strings.ToUpper(id)
	gs := p.Guards[id]
	if len(gs) == 0 {
		return fmt.Errorf("no guards for %s", id)
	}
	for _, g := range gs {
		fmt.Fprintln(w, "G "+g)
	}
	return nil
}

func IDs(w io.Writer, p *model.Project) {
	ids := []string{}
	for id := range p.Owners {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintf(w, "%s\t%s\n", id, p.Owners[id])
	}
}
func unique(in []string) []string {
	m := map[string]bool{}
	o := []string{}
	for _, x := range in {
		if x != "" && !m[x] {
			m[x] = true
			o = append(o, x)
		}
	}
	sort.Strings(o)
	return o
}
