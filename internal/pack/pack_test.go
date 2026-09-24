package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/podopodo/abs/internal/config"
	"github.com/podopodo/abs/internal/scan"
)

// @ACP G ACP.PACK

func TestPackInlinesCompressedEvidenceWithinBudget(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(root, filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	var body strings.Builder
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&body, "\tx%d := %d\n", i, i)
	}
	write("CONTEXT.md", "P: p\nR: r\nB: b\nX: x\n")
	write("orders/cancel.go", "package orders\n\n// @ACP O ORDER.CANCEL\nfunc Cancel() {\n"+body.String()+"}\n")
	write("tests/cancel_guard.go", "package tests\n// @ACP G ORDER.CANCEL\nfunc Guard(){}\n")
	cfg := config.Default()
	p, err := scan.Scan(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	r := Build(p, cfg, "cancel orders", 0, 0)
	if r.Budget != cfg.PackBudget || r.UsedTokens >= r.OriginalTokens || r.UsedTokens > r.Budget {
		t.Fatalf("budget=%d used=%d original=%d", r.Budget, r.UsedTokens, r.OriginalTokens)
	}
	var out strings.Builder
	PrintText(&out, r)
	for _, want := range []string{"== CFILE CONTEXT.md", "== F orders/cancel.go code", "func Cancel() {", "elided (L5-84)", "== F tests/cancel_guard.go"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in\n%s", want, out.String())
		}
	}
	tight := Build(p, cfg, "cancel orders", 0, 10)
	skipped := false
	for _, f := range tight.Files {
		skipped = skipped || f.Skipped
	}
	if !skipped {
		t.Fatalf("expected a skipped file under a tiny budget: %+v", tight.Files)
	}
}
