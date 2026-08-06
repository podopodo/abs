# Changelog

## v1.0.0 — 2026-08-06

First stable release.

- Initial standalone Go implementation of ACP V30.
- Mixed-stack scanning for source, templates, CSS, shell, SQL, configuration and operations files.
- Scope, check, graph, entries, guards, doctor and init commands.
- Cross-platform release packaging, installers, checksums and attestations.
- Module path `github.com/podopodo/abs`; installers default to that repository.
- CI covers Linux and macOS on Go 1.23 and stable; Windows is cross-compiled and smoke-tested only.
