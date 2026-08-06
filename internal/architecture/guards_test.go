package architecture

import (
	"strings"
	"testing"

	"github.com/podopodo/abs/internal/assets"
	"github.com/podopodo/abs/internal/config"
)

// @ACP G ACP.CLI
// @ACP G ACP.SCAN
// @ACP G ACP.SCOPE
// @ACP G ACP.CHECK
// @ACP G ACP.INIT
// @ACP G ACP.DOCTOR
// @ACP G ACP.QUERY
// @ACP G ACP.PROTOCOL
// @ACP G ACP.CONFIG
// @ACP G ACP.RELEASE
// @ACP G ACP.BENCHMARK
func TestEmbeddedProtocolAndDefaults(t *testing.T) {
	if !strings.Contains(assets.Protocol, "ACP v30") {
		t.Fatal("embedded protocol missing version")
	}
	cfg := config.Default()
	if cfg.MaxScopeFiles < 10 || cfg.ContextFile != "CONTEXT.md" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}
