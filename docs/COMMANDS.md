# Commands

## `acp init`

Creates the embedded protocol, root context, empty state and default `.acp.json`.

Use `--force` only when intentionally replacing those templates.

## `acp scope "task"`

Generates a task capsule. Options:

- `--json` for machine-readable output.
- `--max-files N` to override the configured cap.
- `--root DIR` as a global option before the command.

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
