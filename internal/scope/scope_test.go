package scope

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/scan"
)

func TestClauseOwnerAndGuardClosure(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(body), 0644)
	}
	write("orders/service.go", "package orders\n// @ACP O ORDER.CANCEL\nfunc Cancel(){}\n")
	write("storage/repository.go", "package storage\n// @ACP O ORDER.PERSISTENCE\nfunc Save(){}\n")
	write("workers/retry.go", "package workers\n// @ACP D ORDER.PERSISTENCE\nfunc Retry(){}\n")
	write("tests/order_guard.go", "package tests\n// @ACP G ORDER.CANCEL+ORDER.PERSISTENCE\nfunc Guard(){}\n")
	p, err := scan.Scan(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	r := Build(p, config.Default(), "cancel orders idempotently and survive restart", 18)
	seen := map[string]bool{}
	for _, f := range r.Files {
		seen[f.Path] = true
	}
	if !seen["orders/service.go"] || !seen["storage/repository.go"] || !seen["tests/order_guard.go"] {
		t.Fatalf("scope=%v", r.Files)
	}
}
