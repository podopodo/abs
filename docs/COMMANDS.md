# Commands

## `acp init`

Creates the embedded protocol, root context, empty state and default `.acp.json`.

Use `--force` only when intentionally replacing those templates.

## `acp scope "task"`

Generates a task capsule. Options:

- `--json` for machine-readable output.
- `--max-files N` to override the configured cap.
- `--root DIR` as a global option before the command.

## `acp pack "task"`

Runs `acp scope`, then inlines the evidence itself: context files verbatim and each scoped file, in rank order, as a task-aware compressed outline until the estimated-token budget is spent. Files that do not fit are listed as `SKIP` lines with their path.

- `--budget TOKENS` overrides `pack_budget` (default 8000).
- `--max-files N` overrides the scope cap.
- `--json` for machine-readable output.

Elision markers such as `// … 212 lines elided (L35-246)` cite original line numbers, so the agent reads exactly that range when a body matters.

## `acp compress [FILE...]`

Compresses files, or stdin when no file is given, for a model's context window. The original is cached so every elision can be reversed.

| Kind | Strategy |
|---|---|
| `json` | JSON, JSON Lines and concatenated values. Arrays of objects hoist constant fields into `common`, add `summary` value counts and numeric ranges, and keep first/last, error, outlier, per-category and task-matching rows with their original `#` index. Long scalar arrays and strings are sampled. |
| `log` | Keeps head, tail, every error/warning line with its stack, and the first two lines of each line shape (numbers, hex IDs and UUIDs normalized). Test-runner pass noise collapses. |
| `code` | Outline: package, imports, types, signatures, doc comments and `@ACP` tags stay; long function bodies become elision markers. Go uses `go/ast`; brace languages and Python/Ruby use structural heuristics. |
| `diff` | Keeps headers and changed lines with one line of context. |
| `text` | Collapses blank runs and repeated lines; long text keeps head, tail, headings and signal lines. |

- `--kind K` forces a kind; the default `auto` detects from file extension and content.
- `--budget TOKENS` caps the output; signal lines are reserved before head and tail fill.
- `--query TEXT` keeps lines and rows that mention task words.
- `--no-store` skips caching the original.
- `--json` emits stats and output.

Output ends with one footer line, for example:

```text
[acp] log 41443->588 est-tokens (-99%); expand: acp expand 13d5cb5932d0 --lines A:B
```

Output is never longer than the input. Token counts are bytes/4 estimates.

## `acp run [flags] -- COMMAND [ARGS...]`

Runs a command, captures stdout and stderr together, compresses the result like `acp compress`, and exits with the command's exit code (127 when it cannot start). The footer includes `exit=N`. Use it to wrap test runners, builds, linters and API calls:

```bash
acp run -- go test ./...
acp run --kind json -- gh api repos/OWNER/REPO/issues
```

## `acp expand ID`

Prints a cached original. `--lines A:B` prints only that 1-based inclusive range, numbered. A unique ID prefix is enough.

## `acp savings`

Totals the local ledger of compressions: runs, original and compressed estimated tokens, and savings per kind. `--json` for machine output; `--reset` deletes cached originals and the ledger.

## `acp check`

Validates ownership, guard mappings, contexts, state, syntax and selected hygiene rules.

- `--changed` focuses detailed file checks on Git-changed files while retaining global ownership checks.
- `--strict` enables advisory HTML/CSS, shell and duplicate-body checks.
- `--json` emits structured findings.

Exit code is nonzero when error-level findings exist.

## `acp doctor`

Shows platform, architecture, detected file kinds and optional external tools.

## Graph queries

```bash
acp graph ORDER.PAYMENT
acp entries ORDER.FULFILMENT
acp guards ORDER.FULFILMENT
acp ids
```

## Other commands

```bash
acp protocol
acp version
```
