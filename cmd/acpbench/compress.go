package main

// @ACP D ACP.BENCHMARK

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/podopodo/abs/internal/compress"
)

// corpus is one compression input plus the facts a model must still see.
type corpus struct {
	Name     string
	Path     string
	Text     string
	MustKeep []string
}

type compressRow struct {
	Name             string   `json:"name"`
	Kind             string   `json:"kind"`
	OriginalTokens   int      `json:"original_tokens"`
	CompressedTokens int      `json:"compressed_tokens"`
	SavedPct         float64  `json:"saved_pct"`
	Retained         int      `json:"retained"`
	Required         int      `json:"required"`
	Missing          []string `json:"missing,omitempty"`
	Micros           int64    `json:"micros"`
}

type compressReport struct {
	Note    string        `json:"note"`
	Results []compressRow `json:"results"`
}

// runCompressBench measures estimated-token savings and signal retention on
// deterministic synthetic corpora plus real source files from this repo.
// tags: benchmark, compress
func runCompressBench(out string) {
	rep := compressReport{Note: "tokens are estimates (bytes/4); retention = required facts still present"}
	for _, c := range corpora() {
		start := time.Now()
		r := compress.Compress([]byte(c.Text), compress.Options{Path: c.Path})
		row := compressRow{Name: c.Name, Kind: r.Kind, OriginalTokens: r.OriginalTokens, CompressedTokens: r.CompressedTokens, Required: len(c.MustKeep), Micros: time.Since(start).Microseconds()}
		row.SavedPct = float64(int((1-r.Ratio())*1000+0.5)) / 10
		for _, k := range c.MustKeep {
			if strings.Contains(r.Output, k) {
				row.Retained++
			} else {
				row.Missing = append(row.Missing, k)
			}
		}
		rep.Results = append(rep.Results, row)
		fmt.Printf("%-16s %-5s %7d -> %6d est-tokens  -%5.1f%%  retained %d/%d  %dµs\n", row.Name, row.Kind, row.OriginalTokens, row.CompressedTokens, row.SavedPct, row.Retained, row.Required, row.Micros)
	}
	if out != "" {
		b, _ := json.MarshalIndent(rep, "", "  ")
		fatalIf(os.WriteFile(out, append(b, '\n'), 0o644))
	}
}

// tags: benchmark, compress
func corpora() []corpus {
	rng := rand.New(rand.NewSource(42))
	var logb strings.Builder
	for i := 0; i < 3000; i++ {
		fmt.Fprintf(&logb, "2026-09-01T12:%02d:%02dZ INFO http request id=%08x method=GET path=/api/orders/%d status=200 dur=%dms\n", i/60%60, i%60, rng.Uint32(), rng.Intn(900), rng.Intn(120))
		switch i {
		case 1800:
			logb.WriteString("2026-09-01T12:30:00Z ERROR payment capture failed order=7731 reason=card_declined\n")
		case 2400:
			logb.WriteString("panic: runtime error: index out of range [3] with length 3\n\ngoroutine 41 [running]:\nmain.allocate(...)\n\t/srv/inventory/allocate.go:88 +0x1d4\n")
		}
	}
	logb.WriteString("2026-09-01T12:59:59Z INFO shutdown complete requests=3000 errors=2\n")

	type order struct {
		ID       int     `json:"id"`
		Tenant   string  `json:"tenant"`
		Currency string  `json:"currency"`
		Status   string  `json:"status"`
		Total    float64 `json:"total"`
		Region   string  `json:"region"`
	}
	orders := []order{}
	statuses := []string{"paid", "shipped", "pending"}
	for i := 0; i < 500; i++ {
		o := order{ID: 1000 + i, Tenant: "acme", Currency: "EUR", Status: statuses[rng.Intn(3)], Total: float64(20 + rng.Intn(80)), Region: "eu-west-1"}
		if i == 313 {
			o.Status, o.Total = "failed", 0
		}
		if i == 417 {
			o.Total = 98000
		}
		orders = append(orders, o)
	}
	jb, _ := json.MarshalIndent(map[string]any{"orders": orders, "page": 1}, "", "  ")

	var testb strings.Builder
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&testb, "=== RUN   TestCase%03d\n--- PASS: TestCase%03d (0.0%ds)\n", i, i, i%9)
		if i == 222 {
			testb.WriteString("=== RUN   TestRefundIdempotent\n    refund_test.go:57: second refund created a new ledger row\n--- FAIL: TestRefundIdempotent (0.01s)\n")
		}
	}
	testb.WriteString("FAIL\nFAIL\tshop/payments\t1.912s\n")

	out := []corpus{
		{"service-log", "", logb.String(), []string{"card_declined", "index out of range", "allocate.go:88", "errors=2"}},
		{"api-json", "", string(jb), []string{`"failed"`, "98000", `"tenant":"acme"`, "500 objects"}},
		{"go-test-output", "", testb.String(), []string{"TestRefundIdempotent", "refund_test.go:57", "FAIL\tshop/payments"}},
	}
	for _, src := range []struct {
		path string
		keep []string
	}{
		{"internal/scope/scope.go", []string{"func Build(", "func PrintText(", "@ACP O ACP.SCOPE"}},
		{"internal/check/check.go", []string{"func Run(", "type Options struct", "@ACP O ACP.CHECK"}},
	} {
		if b, err := os.ReadFile(src.path); err == nil {
			out = append(out, corpus{src.path, src.path, string(b), src.keep})
		}
	}
	return out
}
