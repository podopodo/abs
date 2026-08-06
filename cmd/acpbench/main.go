package main

// @ACP O ACP.BENCHMARK

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/model"
	"github.com/podopodo/abs/internal/scan"
	scopepkg "github.com/podopodo/abs/internal/scope"
	"github.com/podopodo/abs/internal/textutil"
)

type task struct {
	Name     string   `json:"name"`
	Task     string   `json:"task"`
	Expected []string `json:"expected"`
}
type row struct {
	Name     string   `json:"name"`
	Engine   string   `json:"engine"`
	Selected int      `json:"selected"`
	Bytes    int64    `json:"bytes"`
	Expected int      `json:"expected"`
	Found    int      `json:"found"`
	Recall   float64  `json:"recall"`
	Micros   int64    `json:"micros"`
	Paths    []string `json:"paths"`
}
type report struct {
	Root    string `json:"root"`
	Tasks   int    `json:"tasks"`
	Results []row  `json:"results"`
}

func main() {
	root := flag.String("root", "examples/mixed-stack", "project root")
	tasksPath := flag.String("tasks", "benchmarks/tasks.json", "tasks JSON")
	max := flag.Int("max-files", 18, "file cap")
	out := flag.String("out", "", "optional JSON output path")
	flag.Parse()
	b, err := os.ReadFile(*tasksPath)
	fatalIf(err)
	var tasks []task
	fatalIf(json.Unmarshal(b, &tasks))
	cfg, err := config.Load(*root)
	fatalIf(err)
	p, err := scan.Scan(*root, cfg)
	fatalIf(err)
	r := report{Root: *root, Tasks: len(tasks)}
	for _, t := range tasks {
		start := time.Now()
		sr := scopepkg.Build(p, cfg, t.Task, *max)
		paths := []string{}
		for _, f := range sr.Files {
			paths = append(paths, f.Path)
		}
		r.Results = append(r.Results, measure(p, t, "acp", paths, time.Since(start)))
		start = time.Now()
		r.Results = append(r.Results, measure(p, t, "lexical", lexical(p, t.Task, *max), time.Since(start)))
	}
	enc, err := json.MarshalIndent(r, "", "  ")
	fatalIf(err)
	enc = append(enc, '\n')
	if *out != "" {
		fatalIf(os.WriteFile(*out, enc, 0644))
	} else {
		os.Stdout.Write(enc)
	}
}

func lexical(p *model.Project, task string, max int) []string {
	q := textutil.Words(task)
	type s struct {
		path string
		n    int
	}
	xs := []s{}
	for path, f := range p.Files {
		n := textutil.Overlap(q, f.Tokens)*2 + textutil.Overlap(q, textutil.Words(path))*4
		if n > 0 {
			xs = append(xs, s{path, n})
		}
	}
	sort.Slice(xs, func(i, j int) bool {
		if xs[i].n == xs[j].n {
			return xs[i].path < xs[j].path
		}
		return xs[i].n > xs[j].n
	})
	if len(xs) > max {
		xs = xs[:max]
	}
	out := []string{}
	for _, x := range xs {
		out = append(out, x.path)
	}
	return out
}
func measure(p *model.Project, t task, engine string, paths []string, d time.Duration) row {
	set := map[string]bool{}
	var bytes int64
	for _, x := range paths {
		set[x] = true
		if f := p.Files[x]; f != nil {
			bytes += f.Size
		}
	}
	found := 0
	for _, x := range t.Expected {
		if set[x] {
			found++
		}
	}
	recall := 1.0
	if len(t.Expected) > 0 {
		recall = float64(found) / float64(len(t.Expected))
	}
	return row{Name: t.Name, Engine: engine, Selected: len(paths), Bytes: bytes, Expected: len(t.Expected), Found: found, Recall: recall, Micros: d.Microseconds(), Paths: paths}
}
func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "acpbench:", err)
		os.Exit(1)
	}
}
