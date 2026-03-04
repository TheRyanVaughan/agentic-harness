# Build and install agentic-harness

default: build

# Build the binary to ./harness
build:
    go build -o harness ./cmd/harness

# Install to $GOPATH/bin (on PATH)
install:
    go install ./cmd/harness

# Run all tests
test:
    go test ./...

# Build + test
check: build test

# Remove build artifacts
clean:
    rm -f harness
