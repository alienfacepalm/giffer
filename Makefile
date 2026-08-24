.PHONY: check test vet fmt build tidy release

check: tidy fmt vet test

tidy:
	go mod tidy

fmt:
	go run ./scripts/checkfmt

vet:
	go vet ./...

test:
	go test ./...

build:
	go build -o bin/giffer ./cmd/giffer

# Cross-platform binaries: Windows, Linux (amd64/arm64), macOS (Intel + Apple Silicon).
# Uses go run so env vars work the same on Windows, Linux, and macOS hosts.
release:
	go run ./scripts/release
