package compress

// @ACP D ACP.COMPRESS

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Tunables for the JSON crusher. Arrays at or below smallArray are kept whole.
const (
	smallArray   = 10
	maxKept      = 20
	longString   = 600
	outlierZ     = 3.0
	maxOutliers  = 3
	categoryCard = 6
)

// object is a JSON object that remembers key order.
type object struct {
	keys []string
	vals map[string]any
}

// crushJSON compresses a JSON document or JSON Lines stream. It hoists fields
// that are constant across an array of objects, keeps first/last, error,
// outlier, category and task-matching items, and drops the rest with a count.
// tags: compress, json_crush
func crushJSON(text string, opt Options) (string, bool) {
	words := queryWords(opt.Query)
	vals, err := decodeStream(text)
	if err != nil || len(vals) == 0 {
		return "", false
	}
	var v any = vals[0]
	lines := len(vals) > 1
	if lines {
		v = vals
	}
	var b bytes.Buffer
	if lines {
		b.WriteString("// JSON stream of " + strconv.Itoa(len(vals)) + " values as array\n")
	}
	writeJSON(&b, crushValue(v, words))
	return b.String(), true
}

// decodeStream decodes one JSON document, JSON Lines, or concatenated values.
// tags: compress, json_crush
func decodeStream(text string) ([]any, error) {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	out := []any{}
	for {
		v, err := decodeValue(dec)
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
}

// tags: compress, json_crush
func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := &object{vals: map[string]any{}}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				k, _ := kt.(string)
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				if _, dup := o.vals[k]; !dup {
					o.keys = append(o.keys, k)
				}
				o.vals[k] = v
			}
			_, err := dec.Token()
			return o, err
		case '[':
			arr := []any{}
			for dec.More() {
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			_, err := dec.Token()
			return arr, err
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	default:
		return tok, nil
	}
}

// tags: compress, json_crush
func crushValue(v any, words []string) any {
	switch x := v.(type) {
	case *object:
		o := &object{keys: x.keys, vals: map[string]any{}}
		for _, k := range x.keys {
			o.vals[k] = crushValue(x.vals[k], words)
		}
		return o
	case []any:
		return crushArray(x, words)
	case string:
		if len(x) > longString {
			return x[:300] + fmt.Sprintf(" …[%d chars elided]… ", len(x)-400) + x[len(x)-100:]
		}
		return x
	default:
		return v
	}
}

// tags: compress, json_crush
func crushArray(arr []any, words []string) any {
	if len(arr) <= smallArray {
		out := make([]any, len(arr))
		for i, v := range arr {
			out[i] = crushValue(v, words)
		}
		return out
	}
	objs := 0
	nums := 0
	for _, v := range arr {
		switch v.(type) {
		case *object:
			objs++
		case json.Number:
			nums++
		}
	}
	switch {
	case objs == len(arr):
		return crushObjects(arr, words)
	case nums == len(arr):
		return crushNumbers(arr)
	default:
		return crushMixed(arr, words)
	}
}

// crushObjects is the table case: hoist constant fields, keep signal rows.
// tags: compress, json_crush
func crushObjects(arr []any, words []string) any {
	n := len(arr)
	rows := make([]*object, n)
	fields := []string{}
	seenField := map[string]bool{}
	for i, v := range arr {
		rows[i] = v.(*object)
		for _, k := range rows[i].keys {
			if !seenField[k] {
				seenField[k] = true
				fields = append(fields, k)
			}
		}
	}
	// Constant fields: present everywhere with one encoded value.
	common := &object{vals: map[string]any{}}
	isCommon := map[string]bool{}
	encoded := func(v any) string {
		var b bytes.Buffer
		writeJSON(&b, v)
		return b.String()
	}
	for _, f := range fields {
		first, ok := rows[0].vals[f]
		if !ok {
			continue
		}
		want := encoded(first)
		same := true
		for _, r := range rows[1:] {
			v, ok := r.vals[f]
			if !ok || encoded(v) != want {
				same = false
				break
			}
		}
		if same {
			isCommon[f] = true
			common.keys = append(common.keys, f)
			common.vals[f] = crushValue(first, words)
		}
	}
	keep := map[int]string{}
	order := []int{}
	mark := func(i int, why string) {
		if _, ok := keep[i]; ok {
			return
		}
		keep[i] = why
		order = append(order, i)
	}
	errs := 0
	for i, r := range rows {
		if rowIsError(r) {
			mark(i, "error")
			errs++
		}
	}
	for _, i := range []int{0, 1, 2, n - 1} {
		mark(i, "edge")
	}
	// Per-field summaries survive for omitted rows: value counts for
	// low-cardinality strings, range and mean for numbers.
	summary := &object{vals: map[string]any{}}
	for _, f := range fields {
		if isCommon[f] {
			continue
		}
		first := map[string]int{}
		counts := map[string]int{}
		values := []string{}
		for i, r := range rows {
			if s, ok := r.vals[f].(string); ok {
				if _, seen := first[s]; !seen {
					first[s] = i
					values = append(values, s)
				}
				counts[s]++
			}
			if len(first) > categoryCard {
				break
			}
		}
		if len(first) >= 2 && len(first) <= categoryCard && len(first)*4 <= n {
			c := &object{vals: map[string]any{}}
			for _, v := range values {
				mark(first[v], "category")
				c.keys = append(c.keys, v)
				c.vals[v] = json.Number(strconv.Itoa(counts[v]))
			}
			summary.keys = append(summary.keys, f)
			summary.vals[f] = c
		}
	}
	for _, f := range fields {
		if isCommon[f] {
			continue
		}
		vals := make([]float64, 0, n)
		idx := make([]int, 0, n)
		for i, r := range rows {
			if num, ok := r.vals[f].(json.Number); ok {
				if x, err := num.Float64(); err == nil {
					vals = append(vals, x)
					idx = append(idx, i)
				}
			}
		}
		for _, k := range outliers(vals) {
			mark(idx[k], "outlier")
		}
		if len(vals)*2 >= n {
			lo, hi, sum := vals[0], vals[0], 0.0
			for _, v := range vals {
				lo, hi, sum = math.Min(lo, v), math.Max(hi, v), sum+v
			}
			summary.keys = append(summary.keys, f)
			summary.vals[f] = &object{keys: []string{"min", "max", "mean"}, vals: map[string]any{"min": num(lo), "max": num(hi), "mean": num(sum / float64(len(vals)))}}
		}
	}
	if len(words) > 0 {
		hits := 0
		for i, r := range rows {
			if hits >= 5 {
				break
			}
			if queryHit(encoded(r), words) {
				mark(i, "match")
				hits++
			}
		}
	}
	limit := maxKept
	if errs > limit {
		limit = errs
	}
	if len(order) > limit {
		order = order[:limit]
	}
	sort.Ints(order)
	items := make([]any, 0, len(order))
	for _, i := range order {
		o := &object{keys: []string{"#"}, vals: map[string]any{"#": json.Number(strconv.Itoa(i))}}
		for _, k := range rows[i].keys {
			if isCommon[k] {
				continue
			}
			o.keys = append(o.keys, k)
			o.vals[k] = crushValue(rows[i].vals[k], words)
		}
		items = append(items, o)
	}
	omitted := n - len(order)
	if omitted == 0 && len(common.keys) == 0 {
		out := make([]any, n)
		for i, v := range arr {
			out[i] = crushValue(v, words)
		}
		return out
	}
	note := fmt.Sprintf("%d objects; %d shown (# = original index), %d omitted", n, len(order), omitted)
	if errs > 0 {
		note += fmt.Sprintf("; %d error rows kept", errs)
	}
	wrap := &object{keys: []string{"_acp"}, vals: map[string]any{"_acp": note}}
	if len(common.keys) > 0 {
		wrap.keys = append(wrap.keys, "common")
		wrap.vals["common"] = common
	}
	if omitted > 0 && len(summary.keys) > 0 {
		wrap.keys = append(wrap.keys, "summary")
		wrap.vals["summary"] = summary
	}
	wrap.keys = append(wrap.keys, "items")
	wrap.vals["items"] = items
	return wrap
}

// rowIsError flags rows that report a failure through common field shapes.
// tags: compress, json_crush
func rowIsError(r *object) bool {
	for _, k := range r.keys {
		lk := strings.ToLower(k)
		switch v := r.vals[k].(type) {
		case string:
			lv := strings.ToLower(v)
			if lk == "level" || lk == "severity" || lk == "status" || lk == "state" || lk == "result" || lk == "outcome" {
				if strings.Contains(lv, "err") || strings.Contains(lv, "fail") || lv == "fatal" || lv == "critical" || lv == "panic" {
					return true
				}
			}
			if (lk == "error" || lk == "err" || lk == "exception") && v != "" {
				return true
			}
		case json.Number:
			if lk == "status" || lk == "status_code" || lk == "statuscode" || lk == "code" && strings.Contains(strings.ToLower(strings.Join(r.keys, " ")), "http") {
				if x, err := v.Int64(); err == nil && x >= 400 && x < 600 {
					return true
				}
			}
			if lk == "exit_code" || lk == "exitcode" {
				if x, err := v.Int64(); err == nil && x != 0 {
					return true
				}
			}
		case bool:
			if !v && (lk == "ok" || lk == "success" || lk == "passed") {
				return true
			}
			if v && (lk == "failed" || lk == "error") {
				return true
			}
		case *object:
			if lk == "error" || lk == "exception" {
				return true
			}
		}
	}
	return false
}

// outliers returns indexes whose z-score exceeds outlierZ, largest first.
// tags: compress, json_crush
func outliers(vals []float64) []int {
	if len(vals) < 8 {
		return nil
	}
	var sum, sq float64
	for _, v := range vals {
		sum += v
	}
	mean := sum / float64(len(vals))
	for _, v := range vals {
		sq += (v - mean) * (v - mean)
	}
	std := math.Sqrt(sq / float64(len(vals)))
	if std == 0 {
		return nil
	}
	type zi struct {
		i int
		z float64
	}
	hits := []zi{}
	for i, v := range vals {
		if z := math.Abs(v-mean) / std; z > outlierZ {
			hits = append(hits, zi{i, z})
		}
	}
	sort.Slice(hits, func(a, b int) bool { return hits[a].z > hits[b].z })
	out := []int{}
	for k, h := range hits {
		if k >= maxOutliers {
			break
		}
		out = append(out, h.i)
	}
	return out
}

// tags: compress, json_crush
func crushNumbers(arr []any) any {
	n := len(arr)
	vals := make([]float64, 0, n)
	min, max := math.Inf(1), math.Inf(-1)
	var sum float64
	for _, v := range arr {
		x, _ := v.(json.Number).Float64()
		vals = append(vals, x)
		sum += x
		min = math.Min(min, x)
		max = math.Max(max, x)
	}
	keep := []any{}
	keep = append(keep, arr[:5]...)
	keep = append(keep, fmt.Sprintf("…%d more…", n-7))
	keep = append(keep, arr[n-2:]...)
	stats := &object{keys: []string{"_acp", "min", "max", "mean", "sample"}, vals: map[string]any{
		"_acp":   fmt.Sprintf("%d numbers", n),
		"min":    num(min),
		"max":    num(max),
		"mean":   num(sum / float64(n)),
		"sample": keep,
	}}
	if idx := outliers(vals); len(idx) > 0 {
		o := []any{}
		for _, i := range idx {
			o = append(o, &object{keys: []string{"#", "v"}, vals: map[string]any{"#": json.Number(strconv.Itoa(i)), "v": arr[i]}})
		}
		stats.keys = append(stats.keys, "outliers")
		stats.vals["outliers"] = o
	}
	return stats
}

// tags: compress, json_crush
func num(x float64) json.Number {
	return json.Number(strconv.FormatFloat(x, 'g', 6, 64))
}

// crushMixed keeps edges, errors, task matches and unique scalars.
// tags: compress, json_crush
func crushMixed(arr []any, words []string) any {
	n := len(arr)
	keep := map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true, n - 2: true, n - 1: true}
	seen := map[string]bool{}
	for i, v := range arr {
		if len(keep) >= maxKept {
			break
		}
		var b bytes.Buffer
		writeJSON(&b, v)
		s := b.String()
		if important(s) || queryHit(s, words) {
			keep[i] = true
		} else if _, scalar := v.(string); scalar && !seen[s] && len(seen) < 8 {
			seen[s] = true
			keep[i] = true
		}
	}
	idx := make([]int, 0, len(keep))
	for i := range keep {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	out := []any{}
	prev := -1
	for _, i := range idx {
		if i-prev > 1 {
			out = append(out, fmt.Sprintf("…%d items omitted (#%d-#%d)…", i-prev-1, prev+1, i-1))
		}
		out = append(out, crushValue(arr[i], words))
		prev = i
	}
	return out
}

// writeJSON emits compact JSON, preserving object key order.
// tags: compress, json_crush
func writeJSON(b *bytes.Buffer, v any) {
	switch x := v.(type) {
	case *object:
		b.WriteByte('{')
		for i, k := range x.keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, k)
			b.WriteByte(':')
			writeJSON(b, x.vals[k])
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJSON(b, e)
		}
		b.WriteByte(']')
	case string:
		writeString(b, x)
	case json.Number:
		b.WriteString(x.String())
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case nil:
		b.WriteString("null")
	default:
		enc, _ := json.Marshal(x)
		b.Write(enc)
	}
}

// tags: compress, json_crush
func writeString(b *bytes.Buffer, s string) {
	enc := json.NewEncoder(b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	b.Truncate(b.Len() - 1) // drop Encoder's newline
}
