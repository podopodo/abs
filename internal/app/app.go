package app

// @ACP O ACP.CLI

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/podopodo/abs/internal/assets"
	"github.com/podopodo/abs/internal/buildinfo"
	checkpkg "github.com/podopodo/abs/internal/check"
	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/doctor"
	"github.com/podopodo/abs/internal/model"
	"github.com/podopodo/abs/internal/project"
	"github.com/podopodo/abs/internal/query"
	"github.com/podopodo/abs/internal/scan"
	scopepkg "github.com/podopodo/abs/internal/scope"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stdout)
		return 2
	}
	root, rest, err := global(args)
	if err != nil {
		fmt.Fprintln(stderr, "acp:", err)
		return 2
	}
	if len(rest) == 0 {
		usage(stdout)
		return 2
	}
	cmd := rest[0]
	rest = rest[1:]
	switch cmd {
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "acp %s commit=%s date=%s %s/%s\n", buildinfo.Version, buildinfo.Commit, buildinfo.Date, runtime.GOOS, runtime.GOARCH)
		return 0
	case "protocol":
		fmt.Fprint(stdout, assets.Protocol)
		return 0
	case "init":
		return runInit(root, rest, stdout, stderr)
	}
	p, cfg, err := scan.ReadProject(root)
	if err != nil {
		fmt.Fprintln(stderr, "acp scan:", err)
		return 1
	}
	switch cmd {
	case "scope":
		return runScope(p, cfg, rest, stdout, stderr)
	case "check":
		return runCheck(p, cfg, rest, stdout, stderr)
	case "doctor":
		return runDoctor(p, rest, stdout, stderr)
	case "graph":
		return runID(rest, stderr, func(id string) error { return query.Graph(stdout, p, id) })
	case "entries":
		return runID(rest, stderr, func(id string) error { return query.Entries(stdout, p, id) })
	case "guards":
		return runID(rest, stderr, func(id string) error { return query.Guards(stdout, p, id) })
	case "ids":
		query.IDs(stdout, p)
		return 0
	default:
		fmt.Fprintln(stderr, "acp: unknown command", cmd)
		usage(stderr)
		return 2
	}
}

func global(args []string) (string, []string, error) {
	root := "."
	out := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--root" {
			if i+1 >= len(args) {
				return "", nil, errors.New("--root requires a directory")
			}
			root = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(args[i], "--root=") {
			root = strings.TrimPrefix(args[i], "--root=")
			continue
		}
		out = append(out, args[i:]...)
		break
	}
	abs, err := filepath.Abs(root)
	return abs, out, err
}

func runInit(root string, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(errw)
	force := fs.Bool("force", false, "replace ACP template files")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	written, err := project.Init(root, project.InitOptions{Force: *force})
	if err != nil {
		fmt.Fprintln(errw, "acp init:", err)
		return 1
	}
	if len(written) == 0 {
		fmt.Fprintln(out, "ACP already initialized; use --force to replace templates")
	} else {
		for _, x := range written {
			fmt.Fprintln(out, "CREATE", x)
		}
	}
	return 0
}

func runScope(p *model.Project, cfg config.Config, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("scope", flag.ContinueOnError)
	fs.SetOutput(errw)
	jsonOut := fs.Bool("json", false, "emit JSON")
	maxFiles := fs.Int("max-files", 0, "override scope file cap")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	task := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if task == "" {
		fmt.Fprintln(errw, "acp scope: task is required")
		return 2
	}
	r := scopepkg.Build(p, cfg, task, *maxFiles)
	if *jsonOut {
		return writeJSON(out, r, errw)
	}
	scopepkg.PrintText(out, r)
	return 0
}

func runCheck(p *model.Project, cfg config.Config, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(errw)
	jsonOut := fs.Bool("json", false, "emit JSON")
	strict := fs.Bool("strict", false, "include advisory checks")
	changed := fs.Bool("changed", false, "focus detailed checks on git-changed files")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := checkpkg.Run(p, cfg, checkpkg.Options{Strict: *strict, Changed: *changed})
	if *jsonOut {
		if rc := writeJSON(out, r, errw); rc != 0 {
			return rc
		}
	} else {
		checkpkg.PrintText(out, r)
	}
	if !r.Passed {
		return 1
	}
	return 0
}

func runDoctor(p *model.Project, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(errw)
	jsonOut := fs.Bool("json", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	r := doctor.Run(p)
	if *jsonOut {
		return writeJSON(out, r, errw)
	}
	doctor.Print(out, r)
	return 0
}

func runID(args []string, errw io.Writer, fn func(string) error) int {
	if len(args) != 1 {
		fmt.Fprintln(errw, "acp: exactly one SOT ID is required")
		return 2
	}
	if err := fn(args[0]); err != nil {
		fmt.Fprintln(errw, "acp:", err)
		return 1
	}
	return 0
}
func writeJSON(out io.Writer, v any, errw io.Writer) int {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(errw, "acp json:", err)
		return 1
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprint(w, `ACP — Architecture Continuity Protocol

Usage:
  acp [--root DIR] init [--force]
  acp [--root DIR] scope [--json] [--max-files N] "task"
  acp [--root DIR] check [--changed] [--strict] [--json]
  acp [--root DIR] doctor [--json]
  acp [--root DIR] graph ID
  acp [--root DIR] entries ID
  acp [--root DIR] guards ID
  acp [--root DIR] ids
  acp protocol
  acp version

Core scanning is embedded; Python, Node, Java and other runtimes are optional.
`)
}
