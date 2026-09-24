# Configuration

ACP reads optional project configuration from `.acp.json`.

```json
{
  "exclude": [".git", "node_modules", "vendor", "dist", "build"],
  "max_file_bytes": 2097152,
  "max_scope_files": 18,
  "context_file": "CONTEXT.md",
  "state_file": ".agent/STATE.md",
  "strict_contexts": false,
  "pack_budget": 8000
}
```

## Fields

- `exclude`: directory names skipped anywhere in the project tree.
- `max_file_bytes`: files larger than this are not loaded into the scanner.
- `max_scope_files`: default task capsule cap. Complex tasks may expand to the built-in safety cap.
- `context_file`: inherited local-context filename.
- `state_file`: unfinished handoff-state path.
- `pack_budget`: default estimated-token budget for `acp pack` (bytes/4 estimate). Override per call with `--budget`.
- `strict_contexts`: reserved for enforcing broader context-presence policies in future versions.

The defaults are embedded in the binary and written by `acp init`. ACP remains usable when `.acp.json` is absent.
