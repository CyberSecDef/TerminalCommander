.PHONY: build clean test linux windows darwin all

# Build for current platform
build:
	go build -o terminalcommander main.go

# Build for Linux
linux:
	GOOS=linux GOARCH=amd64 go build -o terminalcommander-linux main.go

# Build for Windows
windows:
	GOOS=windows GOARCH=amd64 go build -o terminalcommander.exe main.go

# Build for macOS
darwin:
	GOOS=darwin GOARCH=amd64 go build -o terminalcommander-mac main.go

# Build for all platforms
all: linux windows darwin

# Clean build artifacts
clean:
	rm -f terminalcommander terminalcommander-linux terminalcommander.exe terminalcommander-mac

# Run tests
test:
	go test -v ./...

# Install dependencies
deps:
	go mod download
	go mod tidy
