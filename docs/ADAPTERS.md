# File adapters

ACP uses a shared extraction model rather than treating every language as identical.

## HTML and templates

Extracts form actions, IDs, class references, scripts, stylesheets, links, includes and inheritance. Strict checking can compare CSS class selectors against scanned template usage.

## CSS and preprocessors

Extracts selectors, custom properties, imports, keyframes and dependencies. It warns about `!important` in strict mode because it can bypass a canonical style path.

## Shell

Extracts functions, sourced files and shebang language. Built-in checks flag remote scripts piped to a shell, recursive deletion using variables and potentially unquoted variables. `shellcheck`, when installed, remains the recommended deeper validator.

## SQL

Extracts created or altered tables and foreign-key references. Migration semantics remain project-specific and should be guarded by the project test suite.

## General source code

Extracts common function, class, route and import forms for Go, Python, JavaScript/TypeScript, PHP, Ruby and brace-based languages. Unknown languages still receive generic extraction and ACP tags.

## Configuration and operations

Recognizes common configuration formats, Dockerfiles, Compose files, CI workflows, Makefiles, Terraform/HCL, Nginx and Apache files. These are included in task scope when task clauses or architectural closure indicate operational impact.

## Extending adapters

The current implementation keeps adapters inside `internal/scan`. New adapters should populate the same common fields:

```text
kind
tags
entries
symbols
dependencies
risks
```

Avoid adding mandatory runtime dependencies. Optional external analyzers should be detected through `acp doctor` and invoked only when explicitly configured.
