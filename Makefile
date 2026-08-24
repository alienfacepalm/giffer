.PHONY: check test vet fmt build build-gui tidy release icon

check: tidy fmt vet test

tidy:
	go mod tidy

fmt:
	go run ./scripts/checkfmt

vet:
	go vet ./...

test:
	go test ./...

# Rebuild assets/giffer.ico and Windows .syso icon resources from assets/giffer-icon.png.
# Requires: go install github.com/akavel/rsrc@latest
icon:
	go run ./scripts/makeico

# Local dev binary (console on Windows so logs and wizard stay visible).
build:
	go build -tags desktop -o bin/giffer ./cmd/giffer

# Release-style GUI binary for double-click testing (windowsgui on Windows).
build-gui:
	go run ./scripts/build-gui

# Cross-platform binaries: Windows, Linux (amd64/arm64), macOS (Intel + Apple Silicon).
# Uses go run so env vars work the same on Windows, Linux, and macOS hosts.
release:
	go run ./scripts/release
