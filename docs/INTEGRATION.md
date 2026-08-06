# Integrating ACP into a project

## 1. Install the binary

Use a release asset or copy a project-local binary into `tools/acp`.

## 2. Initialize

```bash
acp init
```

Edit the root `CONTEXT.md` to describe the real root purpose, rules, boundaries and forbidden actions.

## 3. Tag existing owners incrementally

Start with high-risk and frequently changed duties:

- authentication and authorization;
- tenant isolation;
- persistence and migrations;
- payments and billing;
- public contracts and event names;
- state transitions and workers;
- shared UI tokens and forms;
- deployment and rollback scripts.

Do not attempt to tag every private helper.

## 4. Map guards

Every owner should have at least one `@ACP G ID` in an executable test, architecture check or other verification file.

## 5. Add local contexts only at architecture boundaries

Use directory contexts for domains, adapters, gateways, infrastructure and test areas. Do not create contexts for arbitrary leaf folders with no local architectural difference.

## 6. CI integration

Add:

```bash
acp check
```

For pull requests, `acp check --changed` is fast, but a full `acp check` should still run before release.

## 7. Agent instructions

Tell agents to read `PROTOCOL.md`, run `acp scope`, read only listed evidence, and finish with tests plus `acp check --changed`.
