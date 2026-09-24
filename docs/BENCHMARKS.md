# Benchmarks and evidence

ACP includes a reproducible retrieval benchmark:

```bash
go run ./cmd/acpbench \
  --root examples/mixed-stack \
  --tasks benchmarks/tasks.json \
  --out benchmarks/latest.json
```

It compares ACP scope with a plain lexical top-N control using expected-impact recall, selected files, selected bytes and runtime.

## Compression benchmark

```bash
make benchmark-compress
# go run ./cmd/acpbench --compress --out benchmarks/compress.json
```

It compresses deterministic synthetic corpora (a 3,000-line service log with one error and one panic, a 500-row API response with a failed row and an outlier, 400 passing tests with one failure) plus two real source files from this repository. Each corpus lists facts that must survive; the benchmark reports estimated tokens before and after, and how many of those facts are still in the output.

| Corpus | Kind | Before | After | Saved | Facts kept |
|---|---|---:|---:|---:|---:|
| service-log | log | 77,295 | 564 | 99.3% | 4/4 |
| api-json | json | 19,123 | 170 | 99.1% | 4/4 |
| go-test-output | log | 5,241 | 80 | 98.5% | 3/3 |
| internal/scope/scope.go | code | 2,185 | 459 | 79.0% | 3/3 |
| internal/check/check.go | code | 2,261 | 379 | 83.2% | 3/3 |

Limitations:

- Tokens are bytes/4 estimates, not provider tokenizer counts.
- Synthetic logs and JSON are highly repetitive by construction; real output is often less compressible.
- "Facts kept" checks for specific strings. It does not prove a model answers equally well; the fact list is small and chosen by the author.
- Code outlines deliberately remove bodies. A task that needs a body must expand it, which costs a second read.

## Historical protocol experiments

The protocol was iterated through V30 using controlled A/B repositories and hidden checks. These figures are retained as design evidence, not universal performance guarantees.

| Experiment | Control working tokens | ACP working tokens | Outcome |
|---|---:|---:|---|
| Clean V9 rental project | 27,497 | 29,241 | V9 used 6.3% more; same defects |
| Large clean V20 field-service suite | 35,255 | 30,231 | V20 used 14.3% fewer; 28/32 modeled risks vs 12/32 |
| V30 cumulative known + unfamiliar domains | 54,923 | 48,271 | V30 used 12.1% fewer; 48/48 modeled risks |
| Final V30 implementation holdout | 4,941 | 5,888 | Both correct; ACP read more evidence and cost more on that single task |

## Critical limitations of those experiments

- No independent local coding-agent runtime was available.
- A/B implementations used isolated passes with the same coding capability.
- Token counts were text-size estimates, not API billing telemetry.
- Modeled hidden-risk prevention partly measured whether scope contained required evidence, not whether every model would use it correctly.
- Repeated use of one codebase can favor metadata already tuned to that architecture; later clean and unfamiliar-domain tests were added to reduce this bias.

The honest conclusion is cumulative, not absolute: ACP can reduce retrieval and repair cost in sufficiently connected projects, but a small or obvious task may be cheaper without it.

## Performance goal

ACP should be judged on a combined score:

```text
correctness
architectural consistency
non-fragmentation
continuity across sessions
retrieval size
repair sessions
runtime and output overhead
false-positive rate
```

A lower token count is not a win if the project becomes fragmented or misses a protected boundary.
