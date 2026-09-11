.PHONY: all build test bench lint fmt vet clean

all: lint test build

build:
	go build -o botcheck ./cmd/botcheck

test:
	go test -v -race ./...

bench:
	go test -run=NONE -bench=. -benchmem ./internal/...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2 run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f botcheck
