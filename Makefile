.PHONY: build test vet check

build:
	mkdir -p bin
	go build -trimpath -o bin/config-api ./cmd/config-api

test:
	go test ./...

vet:
	go vet ./...

check: test vet build
