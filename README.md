# ACP — Architecture Continuity Protocol

Repository: <https://github.com/podopodo/abs>

`acp` is a single native CLI that answers the question a coding agent (or a human returning after two weeks) keeps getting wrong: **who already owns this rule, and what else breaks if I change it?**

It scans a project, resolves a task description into the files that actually own the affected duties, and reports whether the architecture still holds together. No Python, Node, Java or PHP runtime. One binary, standard library only.

```bash
acp scope "Change checkout validation, keep persistence compatible, and guard replay"
```

```text
TASK Change checkout validation, keep persistence compatible, and guard replay
C OK change checkout validation, keep persistence compatible, and guard replay -> UI.CHECKOUT.FORM,UI.CHECKOUT.STYLE
CFILE CONTEXT.md
CFILE web/CONTEXT.md
F web/index.html  # caller of web/checkout.js; caller of web/styles.css; symbol match; task terms
F web/styles.css  # dependency of web/index.html; symbol match; task terms
F tests/guards.go # guard UI.CHECKOUT.FORM; guard UI.CHECKOUT.STYLE
F web/checkout.js # dependency of web/index.html; dependent UI.CHECKOUT.FORM; path match
```

Four files and two context files instead of "read the repo".

## Why this exists

Long, multi-session work rarely fails because someone cannot write syntax. It fails because continuity is lost:

- the existing owner of a rule is never found, so the feature is rebuilt under a new name;
- one public entry gets the new rule while its sibling keeps the old one;
- persistence, workers, contracts or legacy loading are quietly skipped;
- a later session cannot recover *why* a boundary exists;
- documentation grows into a second, stale codebase.

ACP attacks that with five ideas and nothing else:

1. **SOT attribution** — every rule-bearing duty has one stable owner ID.
2. **Local inherited context** — a directory records only its architectural delta, not a prose summary.
3. **Protected paths** — sensitive work goes through declared gateways and contracts.
4. **Echo signals** — features and cross-owner joins have executable guards.
5. **Modification first** — extend the existing responsibility before creating a second one.

The CLI moves repetitive discovery and validation out of the model's context window. Code, contracts, tests and version control stay authoritative; ACP only stores the relationships needed to retrieve them.

## Install

### Released binary — Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/podopodo/abs/main/scripts/install.sh | sh
```

Installs to `~/.local/bin`. Pin a tag or change the directory:

```bash
curl -fsSL https://raw.githubusercontent.com/podopodo/abs/main/scripts/install.sh \
  | ACP_VERSION=v1.0.0 ACP_INSTALL_DIR=/usr/local/bin sh
```

A fork works too — pass it as an argument: `sh -s -- youruser/yourfork`.

The script verifies the SHA-256 from `checksums.txt` before installing.

### Released binary — Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/podopodo/abs/main/scripts/install.ps1 -OutFile install-acp.ps1
./install-acp.ps1
```

Installs to `~\bin`. Override with `-InstallDir`, `-Version` or `-Repository`.

Release assets: `acp_{linux,darwin}_{amd64,arm64}.tar.gz`, `acp_windows_{amd64,arm64}.zip`, `checksums.txt`, plus GitHub artifact provenance attestations.

### With Go

```bash
go install github.com/podopodo/abs/cmd/acp@latest
```

### From source

Go 1.23+ is needed only for building.

```bash
git clone https://github.com/podopodo/abs.git
cd abs
make build          # -> bin/acp
./bin/acp version
```

The binary is named `acp` regardless of the repository name.

## How to use it

### 1. Initialize

```bash
cd your-project
acp init
```

Creates:

```text
PROTOCOL.md      the embedded ACP V30 protocol — this is what an agent reads first
CONTEXT.md       root architectural context (edit this by hand)
.agent/STATE.md  unfinished-handoff state; empty means done
.acp.json        excludes, size caps, scope cap (all optional)
```

### 2. Tag the duties that matter

Tags are plain comments in whatever language the file is written in. Start with the high-risk duties — auth, tenancy, persistence and migrations, payments, public contracts, state machines, shared UI tokens, deploy scripts. Do **not** tag every private helper.

```go
// @ACP O ORDER.TOTAL
```

```html
<!-- @ACP D ORDER.TOTAL -->
```

```css
/* @ACP O UI.CHECKOUT.STYLE */
```

```bash
# @ACP O DEPLOY.RELEASE
```

```sql
-- @ACP G ORDER.SCHEMA
```

| Role | Meaning |
|---|---|
| `O` | authoritative owner of the duty — exactly one per duty |
| `D` | dependent, adapter, copy or consumer |
| `G` | guard test or executable proof; one guard can map several IDs with `+` |

Untagged projects still work — ACP falls back to symbols, paths, dependencies and lexical matching — but ownership answers get sharper with every tag.

### 3. Work the loop

```text
1. Read PROTOCOL.md.
2. acp scope "<task>"
3. Read only the listed files and their context chain.
4. Modify the true owner, on the existing canonical path.
5. Add or extend the relevant guards.
6. Run tests, then acp check --changed.
7. Leave .agent/STATE.md empty when the task is complete.
```

Put steps 1–2 in your agent's system prompt or `CLAUDE.md`/`AGENTS.md` and the rest follows from `PROTOCOL.md`.

### 4. Keep it honest in CI

```bash
acp check --changed     # fast, PR-sized: file checks on changed files, ownership checks still global
acp check               # full run before release
acp check --strict      # adds advisory HTML/CSS, shell and duplicate-body checks
```

Exit code is nonzero when error-level findings exist. Output looks like:

```text
ACP CHECK PASS files=76 owners=16 guards=16 warnings=0
```

### Commands

```text
acp init [--force]                        write protocol, context, state, config
acp scope [--json] [--max-files N] "task"  task capsule: the files you may read
acp check [--changed] [--strict] [--json]  structural + hygiene validation
acp doctor [--json]                        platform, file kinds, optional tools
acp graph ID                               owner, dependents, guards for an ID
acp entries ID                             public entries reaching an ID
acp guards ID                              guards mapped to an ID
acp ids                                    all known IDs
acp protocol                               print the embedded protocol
acp version
```

Global `--root DIR` goes before the command. `--json` is available where it makes sense, so this scripts cleanly.

### Ask about a specific duty

```bash
acp graph ORDER.PAYMENT
acp entries ORDER.FULFILMENT
acp guards ORDER.FULFILMENT
```

## What it can read

Files are identified by extension, well-known filename, directory convention, shebang and content pattern — extensionless scripts included.

| Area | Examples |
|---|---|
| Source | Go, Python, JS, TS, Java, Kotlin, C#, PHP, Ruby, Rust, C/C++, Swift, Scala, Dart, Lua, Elixir |
| Web | HTML, Vue, Svelte, Astro, Twig, Handlebars, EJS, Jinja |
| Styles | CSS, SCSS, Sass, Less, Stylus |
| Shell | Bash, POSIX sh, Zsh, Fish, PowerShell, batch, shebang scripts |
| Data / contracts | SQL, GraphQL, Protobuf, JSON, YAML, TOML, XML, INI, env files |
| Operations | Dockerfile, Compose, Makefile, Jenkinsfile, GitHub Actions, Terraform/HCL, Nginx, Apache |
| Anything else | generic SOT, lexical, symbol and dependency extraction |

Specialized scanners raise precision; unknown formats get generic scanning instead of being ignored. Optional external tools (`rg`, `shellcheck`, `stylelint`, `eslint`, `htmlhint`, `sqlfluff`, `hadolint`) are detected by `acp doctor` and used when present — never required.

## Benchmarks

### Retrieval (reproducible, in this repo)

`cmd/acpbench` compares `acp scope` against a plain lexical top-N search on `examples/mixed-stack`, measuring expected-impact recall, files selected, bytes selected and scan time.

```bash
make benchmark
# go run ./cmd/acpbench --root examples/mixed-stack --tasks benchmarks/tasks.json --out benchmarks/latest.json
```

| Task | Engine | Files | Bytes | Recall | Time |
|---|---|---:|---:|---:|---:|
| checkout-validation | acp | 6 | 1,307 | 100% | 0.2 ms |
| checkout-validation | lexical | 10 | 2,072 | 100% | 0.03 ms |
| design-token | acp | 4 | 765 | 100% | 0.16 ms |
| design-token | lexical | 8 | 1,530 | 100% | 0.03 ms |
| deployment | acp | 3 | 388 | 100% | 0.15 ms |
| deployment | lexical | 4 | 742 | 67% | 0.03 ms |

Totals: **13 files / 2,460 bytes / 100% recall** for ACP against **22 files / 4,344 bytes / 89% recall** for lexical. Roughly half the retrieved bytes with no missed impact file — and the lexical control padded its results with `CONTEXT.md` files while still missing `scripts/env.sh` on the deployment task.

Caveat that matters: this is a small fixture repo and a deterministic *retrieval* benchmark. It says nothing about LLM output quality. Absolute times are sub-millisecond and not a meaningful axis of comparison.

### Protocol experiments (historical, weaker evidence)

The protocol was iterated to V30 with controlled A/B repositories. Kept as design evidence, not as a performance promise:

| Experiment | Control tokens | ACP tokens | Outcome |
|---|---:|---:|---|
| Clean V9 rental project | 27,497 | 29,241 | ACP cost 6.3% more; same defects |
| Large V20 field-service suite | 35,255 | 30,231 | 14.3% fewer; 28/32 modeled risks vs 12/32 |
| V30 known + unfamiliar domains | 54,923 | 48,271 | 12.1% fewer; 48/48 modeled risks |
| V30 implementation holdout | 4,941 | 5,888 | Both correct; ACP read more and cost more |

Those token counts are text-size estimates, not billing telemetry, and no independent agent runtime was used. The honest reading: ACP pays off in sufficiently connected projects and on tasks with hidden coupling; a small, obvious task is cheaper without it. Full method and caveats in [docs/BENCHMARKS.md](docs/BENCHMARKS.md).

## Limitations

ACP is deterministic project intelligence — not a compiler, not a language server.

- AST and call-graph precision varies by language.
- Dynamic imports, reflection, codegen and DI can hide real relationships.
- `@ACP G` proves a guard mapping exists; it cannot prove the assertion is strong.
- `--strict` HTML/CSS and duplicate-body checks produce advisory false positives by design.
- Context savings only materialize if the agent actually reads *only* the scope capsule.
- Windows binaries are cross-compiled and smoke-tested, but the test suite runs only on Linux and macOS — see [Platform coverage](#platform-coverage).

## Development

```bash
make fmt      # gofmt -w .
make vet      # go vet ./...
make test     # go test ./...
make race     # go test -race ./...
make build    # bin/acp
make check    # dogfood: run acp check on this repo
make benchmark
make release  # cross-platform archives + checksums into dist/
```

CI runs gofmt, a `go mod tidy` drift check, vet, race tests, a build, and dogfoods `acp check` on this repository.

### Platform coverage

| Platform | Build | Test suite | Notes |
|---|:--:|:--:|---|
| Linux amd64/arm64 | ✅ | ✅ | full CI matrix, Go 1.23 and stable |
| macOS amd64/arm64 | ✅ | ✅ | full CI matrix, Go 1.23 and stable |
| Windows amd64/arm64 | ✅ | ❌ | cross-compiled every PR; release archives smoke-tested |

Windows is a **build-verified, not test-verified** target. Every PR cross-compiles the Windows binaries and every release smoke-tests an archive, but `go test` never runs there.

The known portability hazards are handled deliberately rather than by luck:

- All reported paths are normalized to `/` with `filepath.ToSlash`, so `scope`, `check` and `--json` output is separator-identical on every platform and safe to match on.
- Shebang classification reads the first line and matches by prefix and substring, so a trailing `\r` from a CRLF file does not defeat it.
- `.gitattributes` normalizes the working tree to LF, except `.ps1`, `.bat` and `.cmd` which are checked out CRLF on purpose.
- `check --changed` shells out to `git`, and `doctor` probes optional tools through `exec.LookPath` — both resolve `.exe` normally on Windows.

What is missing is coverage, not correctness by design: no test run has ever executed on Windows, so a regression there would not be caught by CI. If you hit a path, line-ending or tool-detection problem on Windows, please open an issue — a Windows matrix job is wanted, and a concrete report is what will justify it.

### Releasing

```bash
git tag v1.0.0
git push origin v1.0.0
```

The release workflow tests, cross-compiles six binaries, builds archives, writes `checksums.txt`, attests provenance and publishes the GitHub Release. See [docs/RELEASES.md](docs/RELEASES.md).

## Documentation

| Doc | Contents |
|---|---|
| [docs/WHY.md](docs/WHY.md) | motivation and the five pillars |
| [docs/PROTOCOL.md](docs/PROTOCOL.md) | protocol design |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | CLI internals |
| [docs/COMMANDS.md](docs/COMMANDS.md) | command reference |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | `.acp.json` fields |
| [docs/ADAPTERS.md](docs/ADAPTERS.md) | language and file adapters |
| [docs/INTEGRATION.md](docs/INTEGRATION.md) | rolling ACP into an existing project |
| [docs/BENCHMARKS.md](docs/BENCHMARKS.md) | evidence and its limits |
| [docs/RELEASES.md](docs/RELEASES.md) | release engineering |
| [SECURITY.md](SECURITY.md) | security model |
| [CONTRIBUTING.md](CONTRIBUTING.md) | contribution workflow |

## License

MIT — see [LICENSE](LICENSE).
