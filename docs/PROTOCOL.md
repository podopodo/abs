# Protocol design

The shipped `PROTOCOL.md` is intentionally compact and embedded in the binary. Run `acp protocol` to print the exact bundled version.

## Command guide

The protocol ends with a short "Commands: when -> how -> why" section. Each line names the situation, the exact command, and the reason — for example, run tests through `acp run -- <cmd>` so failures and stacks survive while repeated pass lines collapse. It also says when *not* to compress: short output, exact bytes, or a body being edited. `acp init` writes it into `PROTOCOL.md`, so every agent that reads the protocol learns the tools with it.

## Core invariants

```text
one duty      -> one stable ID
one rule      -> one owner
one behavior  -> one canonical path
one boundary  -> one contract
one rule/join -> one or more guards
```

## Local context

`CONTEXT.md` has exactly four non-empty lines:

```text
P: local purpose
R: local rules
B: boundaries and contracts
X: forbidden local actions
```

A child inherits parent contexts. It records only local differences. It is not a task log, file inventory or source-code summary.

## Temporary state

`.agent/STATE.md` is only for genuinely unfinished handoff state. Completed work leaves it empty or removes it. Durable facts belong in the owner, contract, guard or local context.

## Why tags use comments

Comments are already supported across programming, template, style, shell and data-definition languages. A file-level tag covers ordinary private helpers, avoiding repeated metadata on every function.

ACP recognizes tags only on comment lines. Examples inside strings and Markdown documentation are not treated as project ownership.
