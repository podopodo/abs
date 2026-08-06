package textutil

import (
	"regexp"
	"sort"
	"strings"
)

var wordRE = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_-]*`)
var clauseRE = regexp.MustCompile(`(?i)[.;
]|(?:and|then|also|while|plus)`)

var stop = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "to": {}, "of": {}, "in": {}, "on": {}, "for": {}, "with": {}, "and": {}, "or": {}, "is": {}, "are": {}, "be": {}, "by": {}, "from": {}, "into": {}, "that": {}, "this": {}, "it": {}, "its": {}, "their": {}, "all": {}, "one": {}, "new": {}, "old": {}, "add": {}, "change": {}, "make": {}, "must": {}, "should": {}, "when": {}, "after": {}, "before": {}, "only": {}, "same": {}, "exact": {}, "per": {}, "via": {}, "using": {}, "use": {}, "support": {},
}

var synonyms = map[string][]string{
	"restart":     {"persist", "load", "restore", "repository", "serializer", "legacy"},
	"replay":      {"idempotent", "idempotency", "operation", "retry", "duplicate"},
	"idempotent":  {"replay", "operation", "retry", "duplicate"},
	"rollback":    {"transaction", "partial", "failure", "compensate"},
	"tenant":      {"auth", "authorization", "access", "isolation"},
	"worker":      {"retry", "queue", "background", "job", "scheduler"},
	"event":       {"contract", "publisher", "consumer", "notification"},
	"payment":     {"billing", "invoice", "capture", "refund", "settlement"},
	"report":      {"export", "csv", "stream", "analytics"},
	"state":       {"status", "transition", "workflow", "lifecycle"},
	"inventory":   {"stock", "reservation", "warehouse", "parts"},
	"deployment":  {"deploy", "release", "rollout"},
	"environment": {"env", "config", "configuration"},
	"guarded":     {"guard", "test", "verify"},
}

func Words(s string) map[string]int {
	out := map[string]int{}
	for _, w := range wordRE.FindAllString(strings.ToLower(s), -1) {
		if _, ok := stop[w]; ok || len(w) < 2 {
			continue
		}
		out[w]++
		for _, syn := range synonyms[w] {
			out[syn]++
		}
	}
	return out
}

func Clauses(s string) []string {
	parts := clauseRE.Split(strings.ToLower(s), -1)
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		p = strings.TrimSpace(strings.Trim(p, ",:-"))
		if len(p) < 5 || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	if len(out) == 0 && strings.TrimSpace(s) != "" {
		out = []string{strings.TrimSpace(strings.ToLower(s))}
	}
	return out
}

func Keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func Overlap(a, b map[string]int) int {
	n := 0
	for k, av := range a {
		if bv := b[k]; bv > 0 {
			if av < bv {
				n += av
			} else {
				n += bv
			}
		}
	}
	return n
}

func Unique(in []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
