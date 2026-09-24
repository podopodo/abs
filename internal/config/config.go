package config

// @ACP O ACP.CONFIG

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Exclude        []string `json:"exclude"`
	MaxFileBytes   int64    `json:"max_file_bytes"`
	MaxScopeFiles  int      `json:"max_scope_files"`
	ContextFile    string   `json:"context_file"`
	StateFile      string   `json:"state_file"`
	StrictContexts bool     `json:"strict_contexts"`
	PackBudget     int      `json:"pack_budget"`
}

// tags: config
func Default() Config {
	return Config{
		Exclude:        []string{".git", ".hg", ".svn", "node_modules", "vendor", "dist", "build", "target", ".next", ".nuxt", ".cache", ".pytest_cache", "__pycache__", ".idea", ".vscode"},
		MaxFileBytes:   2 << 20,
		MaxScopeFiles:  18,
		ContextFile:    "CONTEXT.md",
		StateFile:      filepath.FromSlash(".agent/STATE.md"),
		StrictContexts: false,
		PackBudget:     8000,
	}
}

// tags: config
func Load(root string) (Config, error) {
	cfg := Default()
	p := filepath.Join(root, ".acp.json")
	b, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}
	if cfg.MaxFileBytes <= 0 {
		cfg.MaxFileBytes = Default().MaxFileBytes
	}
	if cfg.MaxScopeFiles <= 0 {
		cfg.MaxScopeFiles = Default().MaxScopeFiles
	}
	if cfg.PackBudget <= 0 {
		cfg.PackBudget = Default().PackBudget
	}
	if cfg.ContextFile == "" {
		cfg.ContextFile = Default().ContextFile
	}
	if cfg.StateFile == "" {
		cfg.StateFile = Default().StateFile
	}
	return cfg, nil
}
