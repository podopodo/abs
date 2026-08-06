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
