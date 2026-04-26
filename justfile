# Justfile for labours-go development

# Default recipe - show available commands
default:
    @just --list

# === ESSENTIAL COMMANDS ===

# Build the project
build:
    @echo "Building ./labours"
    go build -o labours

# Run tests
test:
    @echo "Running tests"
    go test ./...

# Check code quality (format + lint)
check:
    @echo "Running code quality checks"
    @if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed, skipping lint"; fi
    @if command -v treefmt >/dev/null 2>&1; then treefmt --fail-on-change --allow-missing-formatter; else echo "treefmt not installed, skipping format check"; fi

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts"
    rm -f labours labours-go coverage.out coverage.html

# === DEVELOPMENT HELPERS ===

# Run with arguments (e.g., just run -i data.yaml -m burndown-project)
run *ARGS:
    @echo "Running labours-go {{ARGS}}"
    go run main.go {{ARGS}}

# Run the built binary
run-built *ARGS:
    just build
    ./labours {{ARGS}}

# === TESTING ===

# Run tests with coverage report
test-coverage:
    @echo "Running tests with coverage"
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report: coverage.html"

# Run integration tests
test-integration:
    @echo "Running integration tests"
    ./scripts/run_tests.sh

# Run visual regression tests
test-visual:
    @echo "Running visual regression tests"
    go test -v ./test/visual/...

# Run visual framework demo
test-visual-demo:
    @echo "Running visual framework demo"
    go test -v ./test/visual/ -run TestVisualFrameworkDemo

# Generate reference images for visual testing
visual-generate-refs:
    @echo "Generating visual reference images"
    GENERATE_REFERENCES=true go test -v ./test/visual/ -run TestReferenceGeneration

# Test Python compatibility (if reference images exist)
test-python-compat:
    @echo "Testing Python compatibility"
    go test -v ./test/visual/ -run TestPythonCompatibilityDemo

# === CHART GENERATION ===

# Generate example burndown chart
demo-burndown:
    @echo "Generating demo burndown chart"
    just build
    ./labours -i example_data/hercules_burndown.yaml -m burndown-project -o demo_burndown.png
    @echo "Chart saved as demo_burndown.png"

# Compare with Python reference
test-chart:
    @echo "Testing chart generation vs Python reference"
    just build
    ./labours -i example_data/hercules_burndown.yaml -m burndown-project -o analysis_results/test_chart.png
    @echo "Chart saved as analysis_results/test_chart.png"

# Generate all available charts for comprehensive testing
generate-all-charts INPUT="example_data/hercules_burndown.yaml" OUTPUT="visual_output":
    @echo "Generating complete chart suite"
    ./scripts/generate_all_charts.sh {{INPUT}} {{OUTPUT}}

# Generate charts quietly (minimal output)
generate-all-quiet INPUT="example_data/hercules_burndown.yaml" OUTPUT="visual_output":
    @echo "Generating charts quietly"
    QUIET=true ./scripts/generate_all_charts.sh {{INPUT}} {{OUTPUT}}
