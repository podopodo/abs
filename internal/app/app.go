package app

// @ACP O ACP.CLI

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/podopodo/abs/internal/assets"
	"github.com/podopodo/abs/internal/buildinfo"
	checkpkg "github.com/podopodo/abs/internal/check"
	"github.com/podopodo/abs/internal/compress"
	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/doctor"
	"github.com/podopodo/abs/internal/model"
	packpkg "github.com/podopodo/abs/internal/pack"
	"github.com/podopodo/abs/internal/project"
	"github.com/podopodo/abs/internal/query"
	"github.com/podopodo/abs/internal/scan"
	scopepkg "github.com/podopodo/abs/internal/scope"
)

// stdin is swappable for tests.
var stdin io.Reader = os.Stdin

// tags: cli
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
	case "compress":
		return runCompress(root, rest, stdout, stderr)
	case "run":
		return runCommand(root, rest, stdout, stderr)
	case "expand":
		return runExpand(root, rest, stdout, stderr)
	case "savings":
		return runSavings(root, rest, stdout, stderr)
	}
	p, cfg, err := scan.ReadProject(root)
	if err != nil {
		fmt.Fprintln(stderr, "acp scan:", err)
		return 1
	}
	switch cmd {
	case "scope":
		return runScope(p, cfg, rest, stdout, stderr)
	case "pack":
		return runPack(p, cfg, rest, stdout, stderr)
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

// tags: cli
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

// tags: cli
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
		fmt.Fprintln(out, "NEXT read PROTOCOL.md: rules plus when, how and why to use each acp command; point your agent at it")
	}
	return 0
}

// tags: cli
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

// tags: cli
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

// tags: cli
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

// tags: cli
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

// tags: cli
func writeJSON(out io.Writer, v any, errw io.Writer) int {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintln(errw, "acp json:", err)
		return 1
	}
	return 0
}

// tags: cli
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
  acp [--root DIR] pack [--json] [--budget TOKENS] [--max-files N] "task"
  acp [--root DIR] compress [--kind K] [--budget TOKENS] [--query TEXT] [--json] [--no-store] [FILE...]
  acp [--root DIR] run [--kind K] [--budget TOKENS] [--query TEXT] [--no-store] -- COMMAND [ARGS...]
  acp [--root DIR] expand [--lines A:B] ID
  acp [--root DIR] savings [--json] [--reset]
  acp protocol
  acp version

Core scanning is embedded; Python, Node, Java and other runtimes are optional.
Compression kinds: auto, json, log, code, diff, text. Token counts are estimates (bytes/4).
`)
}

// compressFlags registers the flags shared by compress and run.
// tags: cli, compress
func compressFlags(fs *flag.FlagSet) (*string, *int, *string, *bool) {
	kind := fs.String("kind", "auto", "content kind: auto, json, log, code, diff, text")
	budget := fs.Int("budget", 0, "estimated-token ceiling for the output")
	query := fs.String("query", "", "task text; matching lines and items are kept")
	noStore := fs.Bool("no-store", false, "do not cache the original for acp expand")
	return kind, budget, query, noStore
}

// parseInterspersed accepts flags before or after positional arguments.
// tags: cli
func parseInterspersed(fs *flag.FlagSet, args []string) ([]string, error) {
	pos := []string{}
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		args = fs.Args()
		if len(args) == 0 {
			return pos, nil
		}
		pos = append(pos, args[0])
		args = args[1:]
	}
}

// tags: cli, compress
func validKind(k string) bool {
	switch k {
	case compress.KindAuto, compress.KindJSON, compress.KindLog, compress.KindCode, compress.KindDiff, compress.KindText:
		return true
	}
	return false
}

// emitCompressed prints compressed output plus a one-line footer that names
// the savings and how to reverse any elision.
// tags: cli, compress, ccr
func emitCompressed(root string, in []byte, r compress.Result, source string, noStore bool, out, errw io.Writer, extra string) compress.Result {
	fmt.Fprintln(out, strings.TrimRight(r.Output, "\n"))
	if r.Elided == 0 && r.Saved() <= 0 {
		if extra != "" {
			fmt.Fprintln(out, "[acp] "+extra)
		}
		return r
	}
	if !noStore {
		id, err := compress.OpenStore(root).Put(in, r, source)
		if err != nil {
			fmt.Fprintln(errw, "acp: cache:", err)
		}
		r.ID = id
	}
	footer := fmt.Sprintf("[acp] %s %d->%d est-tokens (-%d%%)", r.Kind, r.OriginalTokens, r.CompressedTokens, int((1-r.Ratio())*100+0.5))
	if extra != "" {
		footer += " " + extra
	}
	if r.ID != "" && r.Elided > 0 {
		footer += fmt.Sprintf("; expand: acp expand %s --lines A:B", r.ID)
	} else if r.ID != "" {
		footer += "; original: acp expand " + r.ID
	}
	fmt.Fprintln(out, footer)
	return r
}

// tags: cli, compress
func runCompress(root string, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("compress", flag.ContinueOnError)
	fs.SetOutput(errw)
	kind, budget, query, noStore := compressFlags(fs)
	jsonOut := fs.Bool("json", false, "emit JSON with stats and output")
	inputs, err := parseInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if !validKind(*kind) {
		fmt.Fprintln(errw, "acp compress: unknown kind", *kind)
		return 2
	}
	type item struct {
		compress.Result
		Source string `json:"source"`
		Output string `json:"output"`
	}
	if len(inputs) == 0 {
		inputs = []string{"-"}
	}
	items := []item{}
	for _, name := range inputs {
		var b []byte
		var err error
		if name == "-" {
			b, err = io.ReadAll(stdin)
		} else {
			b, err = os.ReadFile(name)
		}
		if err != nil {
			fmt.Fprintln(errw, "acp compress:", err)
			return 1
		}
		path := name
		if name == "-" {
			path = ""
		}
		r := compress.Compress(b, compress.Options{Kind: *kind, Path: path, Budget: *budget, Query: *query})
		source := name
		if name == "-" {
			source = "stdin"
		}
		if *jsonOut {
			if !*noStore && (r.Elided > 0 || r.Saved() > 0) {
				if id, err := compress.OpenStore(root).Put(b, r, source); err == nil {
					r.ID = id
				} else {
					fmt.Fprintln(errw, "acp: cache:", err)
				}
			}
			items = append(items, item{Result: r, Source: source, Output: r.Output})
			continue
		}
		if len(inputs) > 1 {
			fmt.Fprintln(out, "== "+name)
		}
		emitCompressed(root, b, r, source, *noStore, out, errw, "")
	}
	if *jsonOut {
		if len(items) == 1 {
			return writeJSON(out, items[0], errw)
		}
		return writeJSON(out, items, errw)
	}
	return 0
}

// runCommand executes a command, compresses its combined output and keeps
// the command's exit code, so it can wrap test runners and builds.
// tags: cli, compress
func runCommand(root string, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(errw)
	kind, budget, query, noStore := compressFlags(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	argv := fs.Args()
	if len(argv) == 0 {
		fmt.Fprintln(errw, "acp run: command is required: acp run -- COMMAND [ARGS...]")
		return 2
	}
	if !validKind(*kind) {
		fmt.Fprintln(errw, "acp run: unknown kind", *kind)
		return 2
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = stdin
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	code := 0
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			fmt.Fprintln(errw, "acp run:", err)
			return 127
		}
		code = exit.ExitCode()
	}
	b := []byte(buf.String())
	r := compress.Compress(b, compress.Options{Kind: *kind, Budget: *budget, Query: *query})
	emitCompressed(root, b, r, strings.Join(argv, " "), *noStore, out, errw, fmt.Sprintf("exit=%d", code))
	return code
}

// tags: cli, ccr
func runExpand(root string, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("expand", flag.ContinueOnError)
	fs.SetOutput(errw)
	lines := fs.String("lines", "", "original line range A:B (1-based, inclusive)")
	pos, err := parseInterspersed(fs, args)
	if err != nil {
		return 2
	}
	if len(pos) != 1 {
		fmt.Fprintln(errw, "acp expand: exactly one ID is required")
		return 2
	}
	text, err := compress.OpenStore(root).Get(pos[0])
	if err != nil {
		fmt.Fprintln(errw, "acp expand:", err)
		return 1
	}
	if *lines != "" {
		if text, err = compress.Lines(text, *lines); err != nil {
			fmt.Fprintln(errw, "acp expand:", err)
			return 2
		}
	}
	fmt.Fprint(out, text)
	return 0
}

// tags: cli, savings
func runSavings(root string, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("savings", flag.ContinueOnError)
	fs.SetOutput(errw)
	jsonOut := fs.Bool("json", false, "emit JSON")
	reset := fs.Bool("reset", false, "delete cached originals and the ledger")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	st := compress.OpenStore(root)
	if *reset {
		if err := st.Clear(); err != nil {
			fmt.Fprintln(errw, "acp savings:", err)
			return 1
		}
		fmt.Fprintln(out, "ACP SAVINGS RESET")
		return 0
	}
	s, err := st.Summary()
	if err != nil {
		fmt.Fprintln(errw, "acp savings:", err)
		return 1
	}
	if *jsonOut {
		return writeJSON(out, s, errw)
	}
	pct := 0
	if s.OriginalTokens > 0 {
		pct = s.SavedTokens * 100 / s.OriginalTokens
	}
	fmt.Fprintf(out, "ACP SAVINGS runs=%d original=%d compressed=%d saved=%d (-%d%%)\n", s.Runs, s.OriginalTokens, s.CompressedTokens, s.SavedTokens, pct)
	kinds := make([]string, 0, len(s.ByKind))
	for k := range s.ByKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		v := s.ByKind[k]
		fmt.Fprintf(out, "K %s runs=%d original=%d compressed=%d saved=%d\n", k, v.Runs, v.OriginalTokens, v.CompressedTokens, v.SavedTokens)
	}
	fmt.Fprintln(out, "NOTE "+s.Note)
	return 0
}

// tags: cli, pack
func runPack(p *model.Project, cfg config.Config, args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("pack", flag.ContinueOnError)
	fs.SetOutput(errw)
	jsonOut := fs.Bool("json", false, "emit JSON")
	budget := fs.Int("budget", 0, "estimated-token budget (default pack_budget)")
	maxFiles := fs.Int("max-files", 0, "override scope file cap")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	task := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if task == "" {
		fmt.Fprintln(errw, "acp pack: task is required")
		return 2
	}
	r := packpkg.Build(p, cfg, task, *maxFiles, *budget)
	if *jsonOut {
		return writeJSON(out, r, errw)
	}
	packpkg.PrintText(out, r)
	return 0
}
