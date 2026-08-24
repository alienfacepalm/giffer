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

# Cross-platform binaries for download from the repo (committed under release/).
release:
	mkdir -p release
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o release/giffer-windows-amd64.exe ./cmd/giffer
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o release/giffer-linux-amd64 ./cmd/giffer
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o release/giffer-linux-arm64 ./cmd/giffer
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o release/giffer-darwin-amd64 ./cmd/giffer
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o release/giffer-darwin-arm64 ./cmd/giffer
