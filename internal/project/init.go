package project

// @ACP O ACP.INIT

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/podopodo/abs/internal/assets"
)

type InitOptions struct{ Force bool }

func Init(root string, opt InitOptions) ([]string, error) {
	files := map[string]string{
		"PROTOCOL.md":                         assets.Protocol,
		"CONTEXT.md":                          assets.RootContext,
		filepath.FromSlash(".agent/STATE.md"): "",
		".acp.json":                           assets.ConfigJSON,
	}
	written := []string{}
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if _, err := os.Stat(path); err == nil && !opt.Force {
			continue
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return written, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return written, err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return written, err
		}
		written = append(written, filepath.ToSlash(rel))
	}
	return written, nil
}

func PrintInit(w []string) {
	if len(w) == 0 {
		fmt.Println("ACP already initialized; use --force to replace templates")
		return
	}
	for _, p := range w {
		fmt.Println("CREATE " + p)
	}
}
