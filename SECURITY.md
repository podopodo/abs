# Security policy

## Reporting

Please report suspected vulnerabilities privately through GitHub Security Advisories rather than opening a public issue.

## Security properties

- The ACP binary requires no network access for scanning, scoping or checking.
- Files larger than the configured limit and binary files are skipped.
- Symlinks are not followed during project scanning.
- Generated release archives are checksummed and attested by GitHub Actions.
- Optional external tools are detected but not automatically downloaded or executed by core commands.

## Trust boundary

ACP analyzes untrusted repository text. It does not execute scanned project files. `acp check` may invoke `git` only for `--changed`; it does not run project build scripts or tests.

`acp run -- COMMAND` executes exactly the command you pass it, with your privileges, and nothing else. It exists so a caller can compress that command's output.

## Compression cache

`acp compress` and `acp run` keep the uncompressed original under `.acp/cache/` in the `--root` directory (default: current directory) so elisions can be reversed with `acp expand`. That cache is plain text on local disk and may contain whatever the input contained, including secrets printed by a command. `.acp/` writes its own `.gitignore` so it is not committed, the scanner never reads it, and it keeps at most 500 originals. Use `--no-store` to skip caching and `acp savings --reset` to delete the cache and ledger.
