# Justfile for labours-go development

binary := "labours"
build_dir := "bin"
test_output_dir := "test_output"
coverage_file := "coverage.out"
build_flags := "-ldflags=\"-w -s\""
test_flags := "-timeout=10m -race"

# Default recipe - show available commands
default:
    @just --list

# === ESSENTIAL COMMANDS ===

# Build the project
build:
    @echo "Building ./{{binary}}"
    go build -o {{binary}}

# Build release binaries for supported platforms
build-all:
    @echo "Building release binaries"
    mkdir -p {{build_dir}}
    GOOS=linux GOARCH=amd64 go build {{build_flags}} -o {{build_dir}}/{{binary}}-linux-amd64 .
    GOOS=darwin GOARCH=amd64 go build {{build_flags}} -o {{build_dir}}/{{binary}}-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 go build {{build_flags}} -o {{build_dir}}/{{binary}}-darwin-arm64 .
    GOOS=windows GOARCH=amd64 go build {{build_flags}} -o {{build_dir}}/{{binary}}-windows-amd64.exe .

# Run tests
test:
    @echo "Running tests"
    go test ./...

# Run tests without coverage or benchmarks
test-quick:
    @echo "Running quick tests"
    COVERAGE=false BENCHMARKS=false ./scripts/run_tests.sh

# Run unit tests only
test-unit:
    @echo "Running unit tests"
    go test {{test_flags}} ./internal/... ./cmd/...

# Check code quality (format + lint)
check:
    @echo "Running code quality checks"
    @if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed, skipping lint"; fi
    @if command -v treefmt >/dev/null 2>&1; then treefmt --fail-on-change --allow-missing-formatter; else echo "treefmt not installed, skipping format check"; fi

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts"
    go clean
    rm -rf {{build_dir}} {{test_output_dir}}
    rm -f {{binary}} labours-go {{coverage_file}} coverage.html

# === DEVELOPMENT HELPERS ===

# Run with arguments (e.g., just run -i data.yaml -m burndown-project)
run *ARGS:
    @echo "Running labours-go {{ARGS}}"
    go run main.go {{ARGS}}

# Run the built binary
run-built *ARGS:
    just build
    ./{{binary}} {{ARGS}}

# === TESTING ===

# Run tests with coverage report
test-coverage:
    @echo "Running tests with coverage"
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Generate and view coverage report
coverage:
    @echo "Generating coverage report"
    mkdir -p {{test_output_dir}}
    go test {{test_flags}} -coverprofile={{coverage_file}} ./internal/... ./cmd/...
    go tool cover -html={{coverage_file}} -o {{test_output_dir}}/coverage.html
    @echo "Coverage report: {{test_output_dir}}/coverage.html"

# Show coverage by function
coverage-func:
    go test {{test_flags}} -coverprofile={{coverage_file}} ./internal/... ./cmd/...
    go tool cover -func={{coverage_file}}

# Run integration tests
test-integration:
    @echo "Running integration tests"
    go test {{test_flags}} ./test/integration/...

# Run visual regression tests
test-visual:
    @echo "Running visual regression tests"
    go test -v ./test/visual/...

# Generate Go-side PNGs used by the Python-vs-Go parity viewer
parity-update:
    @echo "Generating Go parity reference images"
    GOCACHE=/tmp/labours-parity-gocache go run . -i example_data/hercules_burndown.yaml -m burndown-project -o analysis_results/reference/go_burndown_absolute.png --quiet
    GOCACHE=/tmp/labours-parity-gocache go run . -i example_data/hercules_burndown.yaml -m burndown-project -o analysis_results/reference/go_burndown_relative.png --relative --quiet

# Start parity comparison viewer for Python labours references vs Go outputs
parity-viewer PORT="8090" FILTER="":
    PORT={{PORT}} GOCACHE=/tmp/labours-parity-gocache go run ./cmd/parityviewer --port {{PORT}} --name-filter "{{FILTER}}"

# Print parity comparison rows for filtered cases without starting a server
parity-viewer-print PORT="8090" FILTER="" PREFIX="":
    PORT={{PORT}} GOCACHE=/tmp/labours-parity-gocache go run ./cmd/parityviewer --port {{PORT}} --name-filter "{{FILTER}}" --name-prefix "{{PREFIX}}" --print

# Run benchmark tests
test-bench:
    @echo "Running benchmark tests"
    mkdir -p {{test_output_dir}}
    go test -bench=. -benchmem -run=^$ ./internal/... | tee {{test_output_dir}}/benchmarks.txt
    @echo "Benchmark results saved to {{test_output_dir}}/benchmarks.txt"

# Run comprehensive test suite
test-all: test-unit test-integration test-visual test-bench

# Run visual framework demo
test-visual-demo:
    @echo "Running visual framework demo"
    go test -v ./test/visual/ -run TestVisualFrameworkDemo

# Generate reference images for visual testing
visual-generate-refs:
    @echo "Generating visual reference images"
    GENERATE_REFERENCES=true go test -v ./test/visual/ -run TestReferenceGeneration

# Regenerate golden files for visual tests
golden-regen:
    @echo "Regenerating golden files"
    REGENERATE_GOLDEN=true go test -v ./test/visual/...

# Test Python compatibility (if reference images exist)
test-python-compat:
    @echo "Testing Python compatibility"
    go test -v ./test/visual/ -run TestPythonCompatibilityDemo

# === DEVELOPMENT TASKS ===

# Download and tidy dependencies
deps:
    @echo "Downloading dependencies"
    go mod download
    go mod tidy

# Format Go code
fmt:
    @echo "Formatting code"
    go fmt ./...

# Run go vet
vet:
    @echo "Running go vet"
    go vet ./...

# Run golangci-lint when available
lint:
    @if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run; else echo "golangci-lint not found, please install it"; fi

# Generate test data files
testdata:
    @echo "Generating test data"
    go run test/create_sample_data.go

# Run all code quality checks
quality: fmt vet lint

# Quick development test cycle
dev-test: fmt vet test-quick

# Set up development environment
dev-setup: deps testdata

# Continuous integration target
ci: quality test-all

# Pre-release checks
release-check: clean quality test-all build-all

# Install the binary to GOPATH/bin
install: build
    @echo "Installing {{binary}}"
    go install .

# Remove generated test data
clean-testdata:
    @echo "Cleaning test data"
    rm -rf test/testdata/*.pb test/golden/*.png

# Build Docker image
docker-build:
    @echo "Building Docker image"
    docker build -t labours-go:latest .

# Run tests in Docker
docker-test:
    @echo "Running tests in Docker"
    docker run --rm -v "$PWD":/app -w /app golang:1.22 go test ./...

# Generate documentation
docs:
    @echo "Generating documentation"
    mkdir -p docs
    go doc -all . > docs/api.md

# Show current configuration
config:
    @echo "Configuration:"
    @echo "  Binary name: {{binary}}"
    @echo "  Build directory: {{build_dir}}"
    @echo "  Test output directory: {{test_output_dir}}"
    @echo "  Go version: $(go version)"
    @echo "  Working directory: $PWD"

# Show project status
status:
    @echo "Project Status:"
    @echo "  Git branch: $(git branch --show-current 2>/dev/null || echo unknown)"
    @echo "  Git status: $(git status --porcelain | wc -l) files changed"
    @echo "  Go modules: $(go list -m all | wc -l) dependencies"
    @echo "  Test files: $(find . -name '*_test.go' | wc -l) test files"
    @echo "  Source files: $(find . -name '*.go' -not -name '*_test.go' | wc -l) source files"

# === CHART GENERATION ===

# Generate example burndown chart
demo-burndown:
    @echo "Generating demo burndown chart"
    just build
    ./{{binary}} -i example_data/hercules_burndown.yaml -m burndown-project -o demo_burndown.png
    @echo "Chart saved as demo_burndown.png"

# Compare with Python reference
test-chart:
    @echo "Testing chart generation vs Python reference"
    just build
    ./{{binary}} -i example_data/hercules_burndown.yaml -m burndown-project -o analysis_results/test_chart.png
    @echo "Chart saved as analysis_results/test_chart.png"

# Generate all available charts for comprehensive testing
generate-all-charts INPUT="example_data/hercules_burndown.yaml" OUTPUT="visual_output":
    @echo "Generating complete chart suite"
    ./scripts/generate_all_charts.sh {{INPUT}} {{OUTPUT}}

# Generate charts quietly (minimal output)
generate-all-quiet INPUT="example_data/hercules_burndown.yaml" OUTPUT="visual_output":
    @echo "Generating charts quietly"
    QUIET=true ./scripts/generate_all_charts.sh {{INPUT}} {{OUTPUT}}
