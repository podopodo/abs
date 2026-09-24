package compress

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// @ACP G ACP.COMPRESS+ACP.CCR

func TestDetect(t *testing.T) {
	cases := []struct{ path, text, want string }{
		{"", `{"a":1}`, KindJSON},
		{"", "{\"a\":1}\n{\"a\":2}\n", KindJSON},
		{"", "{\n \"a\": 1\n}\n{\n \"a\": 2\n}\n", KindJSON},
		{"", "diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n", KindDiff},
		{"", "2026-01-01 10:00:00 INFO a\n2026-01-01 10:00:01 INFO b\n2026-01-01 10:00:02 ERROR c\n2026-01-01 10:00:03 INFO d\n", KindLog},
		{"x.py", "def f():\n  pass\n", KindCode},
		{"", "plain words\nmore words\n", KindText},
		{"", "=== RUN   TestA\n--- PASS: TestA (0.00s)\n=== RUN   TestB\n--- PASS: TestB (0.00s)\nPASS\n", KindLog},
	}
	for _, c := range cases {
		if got := Detect(c.path, c.text); got != c.want {
			t.Errorf("Detect(%q, %q) = %s, want %s", c.path, c.text, got, c.want)
		}
	}
}

func TestJSONHoistsCommonFieldsAndKeepsErrorsAndOutliers(t *testing.T) {
	rows := []map[string]any{}
	for i := 0; i < 200; i++ {
		status := "ok"
		if i == 137 {
			status = "failed"
		}
		latency := 10 + i%5
		if i == 88 {
			latency = 9000
		}
		rows = append(rows, map[string]any{"region": "eu-west-1", "id": i, "status": status, "latency_ms": latency})
	}
	b, _ := json.MarshalIndent(rows, "", "  ")
	r := Compress(b, Options{})
	if r.Kind != KindJSON {
		t.Fatalf("kind=%s", r.Kind)
	}
	if r.Ratio() > 0.2 {
		t.Fatalf("ratio %.2f too high:\n%s", r.Ratio(), r.Output)
	}
	for _, want := range []string{`"common":{"region":"eu-west-1"}`, `"#":137`, `"failed"`, `"#":88`, `9000`, `"#":199`, "200 objects"} {
		if !strings.Contains(r.Output, want) {
			t.Errorf("missing %s in %s", want, r.Output)
		}
	}
	if !json.Valid([]byte(r.Output)) {
		t.Fatalf("output is not valid JSON: %s", r.Output)
	}
}

func TestJSONSmallDocumentOnlyCompacts(t *testing.T) {
	in := "{\n  \"b\": 1,\n  \"a\": [1, 2, 3]\n}\n"
	r := Compress([]byte(in), Options{})
	if r.Output != `{"b":1,"a":[1,2,3]}` {
		t.Fatalf("got %s", r.Output)
	}
}

func TestLogKeepsErrorsStackAndTailAndCollapsesRepeats(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&b, "2026-01-01T00:00:%02dZ INFO served request id=%d in %dms\n", i%60, i, i%97)
		if i == 500 {
			b.WriteString("2026-01-01T00:00:00Z ERROR db timeout on orders\n\tat orders.save(repo.go:42)\n\tat main(main.go:9)\n")
		}
	}
	b.WriteString("summary: 1 error\n")
	r := Compress([]byte(b.String()), Options{})
	if r.Kind != KindLog {
		t.Fatalf("kind=%s", r.Kind)
	}
	for _, want := range []string{"ERROR db timeout", "repo.go:42", "main.go:9", "summary: 1 error", "elided (L"} {
		if !strings.Contains(r.Output, want) {
			t.Errorf("missing %q", want)
		}
	}
	if r.Ratio() > 0.1 {
		t.Fatalf("ratio %.2f too high", r.Ratio())
	}
}

func TestGoCodeOutlineKeepsSignaturesTagsAndQueryLines(t *testing.T) {
	tag := "// @" + "ACP O ORDER.TOTAL" // split so the scanner does not index it
	src := `package orders

` + tag + `

// Total sums an order.
func Total(items []int) int {
	sum := 0
	for _, x := range items {
		sum += x
	}
	applyDiscount(&sum)
	return sum
}

func applyDiscount(sum *int) {
	a := 1
	b := 2
	c := 3
	*sum -= a + b + c
}

func short() int { return 1 }
`
	r := Compress([]byte(src), Options{Path: "orders.go", Query: "discount rule"})
	for _, want := range []string{tag, "// Total sums an order.", "func Total(items []int) int {", "applyDiscount(&sum)", "func short() int { return 1 }", "elided (L7-"} {
		if !strings.Contains(r.Output, want) {
			t.Errorf("missing %q in\n%s", want, r.Output)
		}
	}
	if strings.Contains(r.Output, "sum += x") {
		t.Errorf("body line survived:\n%s", r.Output)
	}
}

func TestBraceAndPythonOutlines(t *testing.T) {
	ts := "export class Cart {\n  total(): number {\n    let s = 0;\n    for (const x of this.items) {\n      s += x;\n    }\n    return s;\n  }\n}\n"
	r := Compress([]byte(ts), Options{Path: "cart.ts"})
	if !strings.Contains(r.Output, "total(): number {") || strings.Contains(r.Output, "s += x") {
		t.Fatalf("ts outline:\n%s", r.Output)
	}
	py := "class Cart:\n    def total(self):\n        \"\"\"Sum items.\"\"\"\n        s = 0\n        for x in self.items:\n            s += x\n        return s\n\n    def empty(self):\n        return not self.items\n"
	r = Compress([]byte(py), Options{Path: "cart.py"})
	for _, want := range []string{"def total(self):", `"""Sum items."""`, "def empty(self):", "return not self.items"} {
		if !strings.Contains(r.Output, want) {
			t.Errorf("py outline missing %q:\n%s", want, r.Output)
		}
	}
	if strings.Contains(r.Output, "s += x") {
		t.Errorf("py body survived:\n%s", r.Output)
	}
}

func TestDiffTrimsContext(t *testing.T) {
	var b strings.Builder
	b.WriteString("diff --git a/x.go b/x.go\n--- a/x.go\n+++ b/x.go\n@@ -1,40 +1,40 @@\n")
	for i := 0; i < 40; i++ {
		if i == 20 {
			b.WriteString("-old line\n+new line\n")
			continue
		}
		fmt.Fprintf(&b, " context %d\n", i)
	}
	r := Compress([]byte(b.String()), Options{})
	if r.Kind != KindDiff || !strings.Contains(r.Output, "-old line") || !strings.Contains(r.Output, " context 19") || strings.Contains(r.Output, " context 5\n") {
		t.Fatalf("diff:\n%s", r.Output)
	}
}

func TestBudgetKeepsSignalLines(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 400; i++ {
		fmt.Fprintf(&b, "line %d unique payload %x\n", i, i*7919)
		if i == 250 {
			b.WriteString("FATAL: disk full\n")
		}
	}
	r := Compress([]byte(b.String()), Options{Kind: KindText, Budget: 150})
	if r.CompressedTokens > 150 || !strings.Contains(r.Output, "FATAL: disk full") {
		t.Fatalf("tokens=%d\n%s", r.CompressedTokens, r.Output)
	}
}

func TestNeverGrows(t *testing.T) {
	for _, in := range []string{"a", "x\n", "[1,2]", "\n\n\n"} {
		if r := Compress([]byte(in), Options{}); r.CompressedBytes > r.OriginalBytes {
			t.Fatalf("%q grew to %q", in, r.Output)
		}
	}
}

func TestStoreRoundTripLinesAndSavings(t *testing.T) {
	root := t.TempDir()
	st := OpenStore(root)
	orig := []byte("one\ntwo\nthree\nfour\n")
	r := Result{Kind: KindText, OriginalTokens: 10, CompressedTokens: 4}
	id, err := st.Put(orig, r, "test")
	if err != nil {
		t.Fatal(err)
	}
	got, err := st.Get(id[:8])
	if err != nil || got != string(orig) {
		t.Fatalf("get=%q err=%v", got, err)
	}
	lines, err := Lines(got, "2:3")
	if err != nil || lines != "2\ttwo\n3\tthree\n" {
		t.Fatalf("lines=%q err=%v", lines, err)
	}
	if _, err := st.Get("../../etc"); err == nil {
		t.Fatal("path-like id accepted")
	}
	s, err := st.Summary()
	if err != nil || s.Runs != 1 || s.SavedTokens != 6 || s.ByKind[KindText].Runs != 1 {
		t.Fatalf("summary=%+v err=%v", s, err)
	}
	if err := st.Clear(); err != nil {
		t.Fatal(err)
	}
	if s, _ := st.Summary(); s.Runs != 0 {
		t.Fatalf("not cleared: %+v", s)
	}
}
