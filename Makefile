GO ?= go
BINARY := md2wiki

.PHONY: build test vet lint clean

build:
	$(GO) build -o bin/$(BINARY) ./cmd/md2wiki

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

lint:
	golangci-lint run

clean:
	rm -rf bin
