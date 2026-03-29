.DEFAULT_GOAL := test

# Build the dlv binary.
build:
	go build -o dlv ./dlv/

# Install dlv to GOPATH/bin.
install:
	go install ./dlv/

# Remove dlv from GOPATH/bin.
uninstall:
	go env GOPATH | xargs -I{} rm -f {}/bin/dlv

# Run go vet across all packages.
vet:
	go vet ./...

# Run all tests.
test: vet
	go test ./...

# Run only the DVAP package tests (fast).
test-dvap:
	go test -v ./service/dvap/

# Run only the terminal package tests.
test-terminal:
	go test ./pkg/terminal/...

.PHONY: build install uninstall vet test test-dvap test-terminal
