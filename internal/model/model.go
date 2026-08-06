package model

import "time"

type TagRole string

const (
	OwnerTag     TagRole = "O"
	DependentTag TagRole = "D"
	GuardTag     TagRole = "G"
)

type Tag struct {
	Role TagRole  `json:"role"`
	IDs  []string `json:"ids"`
	Line int      `json:"line"`
}

type Entry struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Line int    `json:"line"`
}

type Finding struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	Message  string `json:"message"`
}

type File struct {
	Path         string         `json:"path"`
	Kind         string         `json:"kind"`
	Size         int64          `json:"size"`
	Modified     time.Time      `json:"modified"`
	Text         string         `json:"-"`
	Tokens       map[string]int `json:"-"`
	Tags         []Tag          `json:"tags,omitempty"`
	Entries      []Entry        `json:"entries,omitempty"`
	Symbols      []string       `json:"symbols,omitempty"`
	Dependencies []string       `json:"dependencies,omitempty"`
	References   []string       `json:"references,omitempty"`
	Risks        []Finding      `json:"risks,omitempty"`
}

type Project struct {
	Root       string              `json:"root"`
	Files      map[string]*File    `json:"files"`
	Owners     map[string]string   `json:"owners"`
	OwnerLists map[string][]string `json:"owner_lists"`
	Dependents map[string][]string `json:"dependents"`
	Guards     map[string][]string `json:"guards"`
	ImportedBy map[string][]string `json:"imported_by"`
}

type ScopeItem struct {
	Path    string   `json:"path"`
	Score   float64  `json:"score"`
	Reasons []string `json:"reasons"`
}

type ClauseEvidence struct {
	Clause  string   `json:"clause"`
	IDs     []string `json:"ids,omitempty"`
	Paths   []string `json:"paths,omitempty"`
	Covered bool     `json:"covered"`
}

type ScopeResult struct {
	Task     string           `json:"task"`
	Clauses  []ClauseEvidence `json:"clauses"`
	Files    []ScopeItem      `json:"files"`
	Contexts []string         `json:"contexts,omitempty"`
	Warnings []string         `json:"warnings,omitempty"`
}

type CheckResult struct {
	Passed   bool           `json:"passed"`
	Findings []Finding      `json:"findings"`
	Stats    map[string]int `json:"stats"`
}
