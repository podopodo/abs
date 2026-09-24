package compress

// @ACP O ACP.CCR

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CacheDir is the project-relative directory holding originals and the
// savings ledger. It ignores itself in Git and is never scanned.
const CacheDir = ".acp"

const maxEntries = 500 // originals kept before the oldest are pruned

var idRE = regexp.MustCompile(`^[0-9a-f]{6,64}$`)

// Store keeps compressed-away originals so any elision can be reversed.
type Store struct{ dir string }

// LedgerEntry is one line of the savings ledger.
type LedgerEntry struct {
	Time             time.Time `json:"time"`
	ID               string    `json:"id"`
	Kind             string    `json:"kind"`
	Source           string    `json:"source"`
	OriginalTokens   int       `json:"original_tokens"`
	CompressedTokens int       `json:"compressed_tokens"`
}

// OpenStore returns the store under root without creating anything yet.
// tags: ccr
func OpenStore(root string) *Store {
	return &Store{dir: filepath.Join(root, CacheDir, "cache")}
}

// Put saves the original text and records the result in the ledger.
// It returns the content ID used by Expand.
// tags: ccr, savings
func (s *Store) Put(original []byte, r Result, source string) (string, error) {
	sum := sha256.Sum256(original)
	id := hex.EncodeToString(sum[:])[:12]
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return "", err
	}
	ignore := filepath.Join(filepath.Dir(s.dir), ".gitignore")
	if _, err := os.Stat(ignore); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile(ignore, []byte("*\n"), 0o644)
	}
	path := filepath.Join(s.dir, id+".txt")
	if _, err := os.Stat(path); err != nil {
		if err := os.WriteFile(path, original, 0o644); err != nil {
			return "", err
		}
	} else {
		now := time.Now()
		_ = os.Chtimes(path, now, now)
	}
	e := LedgerEntry{Time: time.Now().UTC(), ID: id, Kind: r.Kind, Source: source, OriginalTokens: r.OriginalTokens, CompressedTokens: r.CompressedTokens}
	if err := s.appendLedger(e); err != nil {
		return id, err
	}
	s.prune()
	return id, nil
}

// tags: savings
func (s *Store) appendLedger(e LedgerEntry) error {
	f, err := os.OpenFile(filepath.Join(s.dir, "ledger.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, _ := json.Marshal(e)
	_, err = f.Write(append(b, '\n'))
	return err
}

// prune removes the oldest originals beyond maxEntries.
// tags: ccr
func (s *Store) prune() {
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	type file struct {
		path string
		mod  time.Time
	}
	files := []file{}
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		if info, err := e.Info(); err == nil {
			files = append(files, file{filepath.Join(s.dir, e.Name()), info.ModTime()})
		}
	}
	if len(files) <= maxEntries {
		return
	}
	sort.Slice(files, func(i, j int) bool { return files[i].mod.Before(files[j].mod) })
	for _, f := range files[:len(files)-maxEntries] {
		_ = os.Remove(f.path)
	}
}

// Get returns the original text for an ID or a unique ID prefix.
// tags: ccr
func (s *Store) Get(id string) (string, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if !idRE.MatchString(id) {
		return "", fmt.Errorf("invalid id %q", id)
	}
	ents, err := os.ReadDir(s.dir)
	if err != nil {
		return "", fmt.Errorf("no cached original %s", id)
	}
	match := ""
	for _, e := range ents {
		name := strings.TrimSuffix(e.Name(), ".txt")
		if name == e.Name() || !strings.HasPrefix(name, id) {
			continue
		}
		if match != "" {
			return "", fmt.Errorf("id %s is ambiguous", id)
		}
		match = e.Name()
	}
	if match == "" {
		return "", fmt.Errorf("no cached original %s", id)
	}
	b, err := os.ReadFile(filepath.Join(s.dir, match))
	return string(b), err
}

// Lines returns original lines a..b (1-based, inclusive) from text, each
// prefixed with its line number. "A:B", "A:" and "A" are accepted.
// tags: ccr
func Lines(text, spec string) (string, error) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	a, b := 1, len(lines)
	parts := strings.SplitN(spec, ":", 2)
	var err error
	if parts[0] != "" {
		if a, err = strconv.Atoi(parts[0]); err != nil {
			return "", fmt.Errorf("bad line range %q", spec)
		}
	}
	if len(parts) == 2 && parts[1] != "" {
		if b, err = strconv.Atoi(parts[1]); err != nil {
			return "", fmt.Errorf("bad line range %q", spec)
		}
	} else if len(parts) == 1 {
		b = a
	}
	if a < 1 {
		a = 1
	}
	if b > len(lines) {
		b = len(lines)
	}
	if a > b {
		return "", fmt.Errorf("line range %q is outside 1:%d", spec, len(lines))
	}
	var sb strings.Builder
	for i := a; i <= b; i++ {
		fmt.Fprintf(&sb, "%d\t%s\n", i, lines[i-1])
	}
	return sb.String(), nil
}

// Savings aggregates the ledger.
type Savings struct {
	Runs             int                `json:"runs"`
	OriginalTokens   int                `json:"original_tokens"`
	CompressedTokens int                `json:"compressed_tokens"`
	SavedTokens      int                `json:"saved_tokens"`
	ByKind           map[string]Savings `json:"by_kind,omitempty"`
	Since            time.Time          `json:"since,omitempty"`
	Note             string             `json:"note"`
}

// Summary reads the ledger and totals estimated savings.
// tags: savings
func (s *Store) Summary() (Savings, error) {
	out := Savings{ByKind: map[string]Savings{}, Note: "estimated tokens (bytes/4), not provider telemetry"}
	f, err := os.Open(filepath.Join(s.dir, "ledger.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e LedgerEntry
		if json.Unmarshal(sc.Bytes(), &e) != nil {
			continue
		}
		if out.Runs == 0 || e.Time.Before(out.Since) {
			out.Since = e.Time
		}
		out.Runs++
		out.OriginalTokens += e.OriginalTokens
		out.CompressedTokens += e.CompressedTokens
		k := out.ByKind[e.Kind]
		k.Runs++
		k.OriginalTokens += e.OriginalTokens
		k.CompressedTokens += e.CompressedTokens
		k.SavedTokens = k.OriginalTokens - k.CompressedTokens
		out.ByKind[e.Kind] = k
	}
	out.SavedTokens = out.OriginalTokens - out.CompressedTokens
	return out, sc.Err()
}

// Clear deletes every cached original and the ledger.
// tags: ccr, savings
func (s *Store) Clear() error { return os.RemoveAll(s.dir) }
