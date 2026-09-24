# ACP v30 — Architecture Continuity Protocol
Goal: correct, clean, one-piece project; least context.
Start: `acp scope "<task>"`; read listed proof only. Read STATE only if unfinished.
Tags: `@ACP O ID` owner; `@ACP D ID` user/copy; `@ACP G ID[+ID]` guard. One owner/path/contract per duty.
Scope: split every requirement clause; resolve SOT by ID, symbol and path; close public entries/callers, gate, contract, data, worker and one guard. Unknown domain uses the same search. Expand until all clauses have proof.
Fix rule>owner, copy>dependent, input>caller, translate>adapter, wire>integration. Reuse; move callers; remove replaced/dead paths.
Gate/data change proves tenant, legacy load, idempotency, retry, rollback and exact boundary names when relevant.
Changed public entry or multi-owner/state join gets relevant success, forbidden, replay, failure and recovery guard.
Run affected tests, static/type, diff and `acp check`. Never weaken tests, bypass gates or hide failure.
`CONTEXT.md` is inherited local delta only: `P/R/B/X`. Durable fact lives in owner/contract/guard/context. DONE leaves STATE empty.
Done: acceptance proved; one owner/path/contract; entries/joins guarded; boundary safe; fragments removed; check passes; risk stated.

## Commands: when -> how -> why
- Before any edit -> `acp scope "<task>"` -> owner files, context chain, guard. Why: find the existing owner; never rebuild it under a new name.
- Need the contents too -> `acp pack "<task>" [--budget N]` -> scoped proof inlined as outlines within an estimated-token budget. Why: one bounded read instead of whole files.
- Question about one duty -> `acp graph|entries|guards ID`, `acp ids` -> owner, dependents, public entries, guards. Why: know the blast radius before changing it.
- Running tests, builds, linters or API calls -> `acp run -- <cmd>` -> errors and stacks kept, repeats collapsed, exit code kept. Why: the failure signal without the log flood.
- Large file, paste, log or JSON -> `acp compress [--query "<task>"] [FILE]` or `<cmd> | acp compress` -> errors, outliers, signatures kept. Why: read the signal, not the repetition.
- Marker `… N lines elided (LA-B)` and the body matters -> `acp expand ID --lines A:B` for output, or read lines A-B of the file for pack. Why: every cut is reversible; pay only for what the task needs.
- After edits -> affected tests, then `acp check --changed`; full `acp check` before release. Why: owners, guards and contexts stay valid.
- Do not compress -> short output, exact bytes needed (patches, hashes, secrets review), or a body you are editing: read it directly.
