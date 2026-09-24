package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitAndProtocol(t *testing.T) {
	root := t.TempDir()
	var out, errb bytes.Buffer
	if code := Run([]string{"--root", root, "init"}, &out, &errb); code != 0 {
		t.Fatalf("code=%d err=%s", code, errb.String())
	}
	for _, want := range []string{"CREATE PROTOCOL.md", "NEXT read PROTOCOL.md"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("init output missing %q:\n%s", want, out.String())
		}
	}
	proto, err := os.ReadFile(filepath.Join(root, "PROTOCOL.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"when -> how -> why", "acp scope", "acp pack", "acp run --", "acp compress", "acp expand", "acp check --changed", "Do not compress"} {
		if !strings.Contains(string(proto), want) {
			t.Errorf("initialized PROTOCOL.md missing %q", want)
		}
	}
	for _, name := range []string{"PROTOCOL.md", "CONTEXT.md", ".acp.json", filepath.FromSlash(".agent/STATE.md")} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	out.Reset()
	if code := Run([]string{"protocol"}, &out, &errb); code != 0 || !bytes.Contains(out.Bytes(), []byte("ACP v30")) {
		t.Fatalf("code=%d out=%s", code, out.String())
	}
}

func TestCompressExpandRunAndSavings(t *testing.T) {
	root := t.TempDir()
	var lines bytes.Buffer
	for i := 0; i < 300; i++ {
		lines.WriteString("2026-01-01T00:00:00Z INFO tick 1\n")
	}
	lines.WriteString("2026-01-01T00:00:00Z ERROR boom\n")
	stdin = bytes.NewReader(lines.Bytes())
	defer func() { stdin = os.Stdin }()
	var out, errb bytes.Buffer
	if code := Run([]string{"--root", root, "compress"}, &out, &errb); code != 0 {
		t.Fatalf("code=%d err=%s", code, errb.String())
	}
	s := out.String()
	if !strings.Contains(s, "ERROR boom") || !strings.Contains(s, "[acp] log") || !strings.Contains(s, "acp expand ") {
		t.Fatalf("compress output:\n%s", s)
	}
	id := strings.Fields(s[strings.Index(s, "acp expand ")+len("acp expand "):])[0]
	out.Reset()
	if code := Run([]string{"--root", root, "expand", id, "--lines", "301:301"}, &out, &errb); code != 0 || out.String() != "301\t2026-01-01T00:00:00Z ERROR boom\n" {
		t.Fatalf("expand code=%d out=%q err=%s", code, out.String(), errb.String())
	}
	out.Reset()
	if code := Run([]string{"--root", root, "savings", "--json"}, &out, &errb); code != 0 || !strings.Contains(out.String(), `"runs": 1`) {
		t.Fatalf("savings code=%d out=%s", code, out.String())
	}
	out.Reset()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// Re-run this test binary with an unknown flag: it prints usage and exits 2.
	code := Run([]string{"--root", root, "run", "--no-store", "--", exe, "-test.bogus"}, &out, &errb)
	if code != 2 || !strings.Contains(out.String(), "exit=2") {
		t.Fatalf("run code=%d out=%s", code, out.String())
	}
	if code := Run([]string{"--root", root, "run", "--", "acp-no-such-binary"}, &out, &errb); code != 127 {
		t.Fatalf("missing binary code=%d", code)
	}
}
