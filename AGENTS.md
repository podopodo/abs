# AGENTS.md

Before editing, read `PROTOCOL.md` and run `acp scope "<task>"`.

## Behavior (ACP rules in PROTOCOL.md win on conflict)
- Think first: state assumptions, name competing readings, ask when unclear.
- Build least: no unrequested features, single-use abstractions or impossible-case handling.
- Touch least: each diff line traces to the task; report unrelated dead code, remove only what you orphaned.
- Prove: turn the task into a check first (failing test, invalid-input test, green before and after).
