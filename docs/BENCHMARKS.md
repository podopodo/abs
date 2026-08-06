# Benchmarks and evidence

ACP includes a reproducible retrieval benchmark:

```bash
go run ./cmd/acpbench \
  --root examples/mixed-stack \
  --tasks benchmarks/tasks.json \
  --out benchmarks/latest.json
```

It compares ACP scope with a plain lexical top-N control using expected-impact recall, selected files, selected bytes and runtime.

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
