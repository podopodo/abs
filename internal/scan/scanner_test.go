package scan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/podopodo/abs/internal/config"
)

func TestMixedKindsTagsAndEntries(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		p := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("app/main.go", "package app\n// @ACP O APP.RUN\nfunc Run() {}\n")
	write("web/index.html", `<!-- @ACP O UI.FORM --><form id="signup" class="panel" action="/signup"></form><link href="style.css">`)
	write("web/style.css", `/* @ACP O UI.STYLE */ .panel{display:grid} :root{--gap:1rem}`)
	write("scripts/deploy", "#!/usr/bin/env bash\n# @ACP O DEPLOY.RUN\ndeploy() { echo ok; }\n")
	write("db/001.sql", "-- @ACP O DB.USER\nCREATE TABLE users(id INT PRIMARY KEY);\n")
	write("tests/guard.go", "package tests\n// @ACP G APP.RUN+UI.FORM+UI.STYLE+DEPLOY.RUN+DB.USER\nfunc Guard(){}\n")
	p, err := Scan(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	if p.Files["web/index.html"].Kind != "html" {
		t.Fatalf("html kind=%s", p.Files["web/index.html"].Kind)
	}
	if p.Files["scripts/deploy"].Kind != "shell" {
		t.Fatalf("shebang kind=%s", p.Files["scripts/deploy"].Kind)
	}
	if got := p.Owners["APP.RUN"]; got != "app/main.go" {
		t.Fatalf("owner=%s", got)
	}
	if len(p.Files["web/index.html"].Entries) < 2 {
		t.Fatalf("html entries=%v", p.Files["web/index.html"].Entries)
	}
	if len(p.Guards["DB.USER"]) != 1 {
		t.Fatalf("guard=%v", p.Guards)
	}
}
