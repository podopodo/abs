# CLI architecture

ACP is built from one Go module and uses the standard library only.

```text
cmd/acp             CLI entry
cmd/releasepack     deterministic cross-platform archive builder
cmd/acpbench        retrieval benchmark harness
internal/app        command parsing and output
internal/assets     embedded protocol and templates
internal/scan       file discovery, classification, tags, entries, dependencies and risks
internal/scope      clause-to-SOT resolution and architectural closure
internal/check      structural and hygiene checks
internal/query      graph, entry and guard queries
internal/project    project initialization
internal/doctor     environment and optional-tool detection
```

## Runtime independence

The release binary is built with `CGO_ENABLED=0`. It requires no Python, Node, Java, PHP or shared ACP runtime.

Optional tools such as `rg`, `shellcheck`, `stylelint`, `eslint`, `htmlhint`, `sqlfluff` and `hadolint` are detected by `acp doctor`; the core scanner continues without them.

## Scanning model

Every file receives:

- a detected kind;
- normalized lexical tokens;
- ACP tags;
- public-entry candidates;
- symbol candidates;
- local dependency references;
- built-in risk findings.

Project-level indexes connect owners, dependents, guards, imports and reverse imports.

## Scope model

`acp scope`:

1. splits a task into independent clauses;
2. ranks lexical, path and symbol matches;
3. resolves up to two likely SOT owners per clause;
4. adds architectural closure for contracts, persistence, workers, security, payments, state, UI and operations;
5. follows one graph step through imports, callers, dependents and guards;
6. preserves clause owners and a relevant guard under the adaptive file cap;
7. returns paths and reasons, not source dumps.
