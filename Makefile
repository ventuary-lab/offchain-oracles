.PHONY: build

GOBIN = ./build/bin
GO ?= latest

build:
	go build -o build/server ./server/main.go
	go build -o build/signer ./signer/main.go
	@echo "Done building."