.PHONY: check test vet fmt build tidy

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
