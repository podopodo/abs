# Retrieval benchmark

This benchmark compares `acp scope` with a plain lexical top-N file search. It measures:

- expected-impact recall;
- number of files selected;
- selected source bytes;
- scanner execution time.

Run:

```bash
go run ./cmd/acpbench \
  --root examples/mixed-stack \
  --tasks benchmarks/tasks.json \
  --out benchmarks/latest.json
```

This is a deterministic retrieval benchmark, not an LLM quality benchmark. Token and coding-agent claims require separate model telemetry and hidden behavioral tests.
