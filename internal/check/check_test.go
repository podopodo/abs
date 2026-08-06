package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/scan"
)

func TestCheckPassAndDetectMissingGuard(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(body), 0644)
	}
	write("CONTEXT.md", "P: x\nR: x\nB: x\nX: x\n")
	write("app.go", "package app\n// @ACP O APP.RUN\nfunc Run(){}\n")
	p, _ := scan.Scan(root, config.Default())
	r := Run(p, config.Default(), Options{})
	if r.Passed {
		t.Fatal("expected missing guard")
	}
	write("guard.go", "package app\n// @ACP G APP.RUN\nfunc Guard(){}\n")
	p, _ = scan.Scan(root, config.Default())
	r = Run(p, config.Default(), Options{})
	if !r.Passed {
		t.Fatalf("findings=%v", r.Findings)
	}
}
