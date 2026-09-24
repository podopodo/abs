package scan

// @ACP O ACP.SCAN

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/model"
	"github.com/podopodo/abs/internal/textutil"
)

// tags: scan
func Scan(root string, cfg config.Config) (*model.Project, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	p := &model.Project{Root: abs, Files: map[string]*model.File{}, Owners: map[string]string{}, OwnerLists: map[string][]string{}, Dependents: map[string][]string{}, Guards: map[string][]string{}, ImportedBy: map[string][]string{}}
	excluded := map[string]bool{}
	for _, x := range cfg.Exclude {
		excluded[x] = true
	}
	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == abs {
			return nil
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			// .acp holds compression caches; it is never project source.
			if excluded[d.Name()] || rel == ".acp" {
				return filepath.SkipDir
			}
			return nil
		}
		if excluded[d.Name()] || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > cfg.MaxFileBytes {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if isBinary(b) {
			return nil
		}
		text := string(b)
		kind := classify(rel, text)
		tags := []model.Tag{}
		if kind != "docs" {
			tags = extractTags(text)
		}
		entries := extractEntries(kind, text)
		f := &model.File{Path: rel, Kind: kind, Size: info.Size(), Modified: info.ModTime(), Text: text, Tokens: textutil.Words(rel + " " + text), Tags: tags, Entries: entries, Dependencies: extractDependencies(text), Risks: extractRisks(kind, rel, text)}
		f.Symbols = extractSymbols(entries, tags, text)
		p.Files[rel] = f
		for _, t := range tags {
			for _, id := range t.IDs {
				switch t.Role {
				case model.OwnerTag:
					p.OwnerLists[id] = append(p.OwnerLists[id], rel)
					if _, ok := p.Owners[id]; !ok {
						p.Owners[id] = rel
					}
				case model.DependentTag:
					p.Dependents[id] = append(p.Dependents[id], rel)
				case model.GuardTag:
					p.Guards[id] = append(p.Guards[id], rel)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for rel, f := range p.Files {
		resolved := []string{}
		for _, dep := range f.Dependencies {
			if r := resolveDependency(rel, dep, p.Files); r != "" {
				resolved = append(resolved, r)
				p.ImportedBy[r] = append(p.ImportedBy[r], rel)
			}
		}
		f.Dependencies = textutil.Unique(resolved)
	}
	normalizeProject(p)
	return p, nil
}

func normalizeProject(p *model.Project) {
	for _, m := range []map[string][]string{p.OwnerLists, p.Dependents, p.Guards, p.ImportedBy} {
		for k, v := range m {
			sort.Strings(v)
			m[k] = textutil.Unique(v)
		}
	}
	for _, f := range p.Files {
		sort.Slice(f.Entries, func(i, j int) bool { return f.Entries[i].Line < f.Entries[j].Line })
	}
}

func isBinary(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	n := len(b)
	if n > 4096 {
		n = 4096
	}
	zeros := 0
	s := bufio.NewScanner(strings.NewReader(string(b[:n])))
	_ = s
	for _, c := range b[:n] {
		if c == 0 {
			zeros++
		}
	}
	return zeros > 0
}

func ReadProject(root string) (*model.Project, config.Config, error) {
	cfg, err := config.Load(root)
	if err != nil {
		return nil, cfg, err
	}
	p, err := Scan(root, cfg)
	return p, cfg, err
}

func File(root, rel string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fs.ErrNotExist
	}
	return b, err
}
