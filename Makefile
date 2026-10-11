.PHONY: run test lint build

run:
	go run .

test:
	go test -race ./...

lint:
	golangci-lint run

build:
	go build -o bin/app .
