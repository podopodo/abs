# Changelog

## v1.1.0 — 2026-09-24

Context compression, following the concept of [Headroom](https://github.com/headroomlabs-ai/headroom), built as deterministic standard-library Go.

- `acp compress` — content-aware compression of files or stdin: JSON (common-field hoisting, per-field summaries, error/outlier/category rows kept), logs and test output (shape dedupe, errors and stacks kept), code outlines (Go via `go/ast`; brace languages, Python and Ruby by structure), diffs and text. Output is never longer than the input.
- `acp run -- CMD` — runs a command, compresses its combined output and keeps its exit code.
- `acp expand ID [--lines A:B]` — reversible compression: originals are cached in `.acp/cache/` and every elision marker cites original line numbers.
- `acp pack "task"` — `acp scope` plus the evidence itself, inlined as task-aware outlines under an estimated-token budget (`pack_budget`, default 8000).
- `acp savings` — local ledger of estimated tokens saved; `--reset` clears cache and ledger.
- `--budget` reserves error and warning lines before filling head and tail; `--query` keeps task-matching lines and rows.
- The embedded `PROTOCOL.md` (written by `acp init`, printed by `acp protocol`) gains a "Commands: when -> how -> why" guide covering scope, pack, graph queries, run, compress, expand and check, plus when not to compress. `acp init` now ends with a `NEXT` line pointing at it.
- Scanner never reads `.acp/`; the directory ignores itself in Git.
- `acpbench --compress` measures savings and required-fact retention (`make benchmark-compress`).

## v1.0.0 — 2026-08-06

First stable release.

- Initial standalone Go implementation of ACP V30.
- Mixed-stack scanning for source, templates, CSS, shell, SQL, configuration and operations files.
- Scope, check, graph, entries, guards, doctor and init commands.
- Cross-platform release packaging, installers, checksums and attestations.
- Module path `github.com/podopodo/abs`; installers default to that repository.
- CI covers Linux and macOS on Go 1.23 and stable; Windows is cross-compiled and smoke-tested only.
