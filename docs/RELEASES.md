# Release engineering

## Toolchain

- Minimum source compatibility: Go 1.23.
- CI tests the minimum line and the current stable Go toolchain.
- Release binaries are built with the current stable toolchain, `CGO_ENABLED=0`, `-trimpath`, and stripped debug symbols.

## Tagged release flow

A pushed `v*` tag triggers `.github/workflows/release.yml`:

1. Check out the complete tagged history.
2. Set up Go.
3. Run formatting, vet and tests.
4. Cross-compile six OS/architecture targets.
5. Package README, license, protocol and binary.
6. Generate `checksums.txt`.
7. Smoke-test the Linux AMD64 archive.
8. Generate GitHub artifact provenance attestations.
9. Upload a workflow artifact.
10. Create the GitHub Release and attach all archives.

GitHub automatically provides a repository-scoped `GITHUB_TOKEN`; the workflow requests only the permissions needed to read code, create the release and generate attestations.

## Reproducible local packaging

```bash
go run ./cmd/releasepack --version v1.0.0 --out dist
```

Archive timestamps are normalized and file names are stable, which supports predictable latest-release download URLs.
