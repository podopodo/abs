package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitAndProtocol(t *testing.T) {
	root := t.TempDir()
	var out, errb bytes.Buffer
	if code := Run([]string{"--root", root, "init"}, &out, &errb); code != 0 {
		t.Fatalf("code=%d err=%s", code, errb.String())
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
