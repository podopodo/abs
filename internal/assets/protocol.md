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
