.PHONY: fmt vet test race build check benchmark release clean

fmt:
	gofmt -w .

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/acp ./cmd/acp

check:
	go run ./cmd/acp check

benchmark:
	go run ./cmd/acpbench --root examples/mixed-stack --tasks benchmarks/tasks.json --out benchmarks/latest.json

release:
	go run ./cmd/releasepack --version "$${VERSION:-dev}" --out dist

clean:
	rm -rf bin dist coverage.txt
