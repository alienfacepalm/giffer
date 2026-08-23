.PHONY: check test vet fmt build tidy

check: tidy fmt vet test

tidy:
	go mod tidy

fmt:
	@files=$$(gofmt -l .); \
	if [ -n "$$files" ]; then \
		echo "gofmt needed on:"; echo "$$files"; exit 1; \
	fi

vet:
	go vet ./...

test:
	go test ./...

build:
	go build -o bin/giffer ./cmd/giffer
