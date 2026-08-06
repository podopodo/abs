# Contributing

1. Create a focused branch.
2. Run `gofmt -w .`.
3. Run `go vet ./...` and `go test -race ./...`.
4. Run `go run ./cmd/acp check`.
5. Add tests for scanner, scope or checker behavior changed by the contribution.
6. Update documentation when a command, release asset or protocol invariant changes.

New mandatory dependencies require a clear justification. ACP aims to keep the released binary self-contained and runtime-independent.
