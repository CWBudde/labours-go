#!/bin/bash

# Generate All Charts Script for labours-go
# Creates comprehensive visual output for all supported modes

set -u # Exit on undefined variables (but not on command failures)

# Configuration
INPUT_FILE="${1:-example_data/hercules_burndown.yaml}"
OUTPUT_DIR="${2:-visual_output}"
QUIET="${QUIET:-false}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
	echo -e "${BLUE}===========================================${NC}"
	echo -e "${BLUE} labours-go: Complete Visual Chart Suite${NC}"
	echo -e "${BLUE}===========================================${NC}"
}

print_status() {
	echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
	echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
	echo -e "${RED}[ERROR]${NC} $1"
}

# Check if labours-go binary exists
check_binary() {
	if [ ! -f "./labours-go" ]; then
		print_status "Building labours-go..."
		go build -o labours-go
		if [ $? -ne 0 ]; then
			print_error "Failed to build labours-go"
			exit 1
		fi
	fi
}

# Create output directory
setup_output() {
	print_status "Setting up output directory: ${OUTPUT_DIR}"
	mkdir -p "${OUTPUT_DIR}"
	mkdir -p "${OUTPUT_DIR}/go"
	mkdir -p "${OUTPUT_DIR}/python" 2>/dev/null || true
	mkdir -p "${OUTPUT_DIR}/comparison"
}

# Check if input file exists
check_input() {
	if [ ! -f "${INPUT_FILE}" ]; then
		print_error "Input file not found: ${INPUT_FILE}"
		print_status "Available input files:"
		find example_data/ -name "*.yaml" -o -name "*.pb" 2>/dev/null || true
		exit 1
	fi
	print_status "Using input file: ${INPUT_FILE}"
}

# Generate Go charts
generate_go_charts() {
	print_status "Generating Go charts..."

	# List of modes most likely to work with basic hercules data
	local modes=("burndown-project") # Focus on modes that work with available data

	# Add other modes only if COMPREHENSIVE is set
	if [ "${COMPREHENSIVE:-false}" = "true" ]; then
		modes+=(
			"ownership"         # Usually requires --burndown-people
			"devs"              # Requires dev stats collection
			"couples-people"    # Requires --couples
			"burndown-file"     # Requires file-level data
			"burndown-person"   # Requires person-level data
			"couples-files"     # Requires coupling data
			"shotness"          # Requires --shotness
			"languages"         # Requires language collection
			"old-vs-new"        # May work with synthetic data
			"overwrites-matrix" # Requires interaction data
			"couples-shotness"  # Requires shotness + coupling
			"devs-efforts"      # Requires dev stats
			"devs-parallel"     # Requires dev stats
			"run-times"         # Rarely available
			"sentiment"         # Requires specific data
		)
		print_status "Running comprehensive mode - testing all ${#modes[@]} modes"
	else
		print_status "Running focused mode - testing ${#modes[@]} modes with available data"
		print_status "Set COMPREHENSIVE=true for full mode testing"
	fi

	local generated_count=0
	local failed_count=0

	for mode in "${modes[@]}"; do
		local output_file="${OUTPUT_DIR}/go/${mode}.png"
		print_status "  Generating ${mode}..."

		if [ "$QUIET" = "true" ]; then
			./labours-go -i "${INPUT_FILE}" -m "${mode}" -o "${output_file}" --quiet 2>/dev/null
			local exit_code=$?
		else
			./labours-go -i "${INPUT_FILE}" -m "${mode}" -o "${output_file}"
			local exit_code=$?
		fi

		if [ $exit_code -eq 0 ] && [ -f "${output_file}" ]; then
			print_status "    ✅ Generated: ${output_file}"
			((generated_count++))
		else
			print_warning "    ❌ Failed: ${mode} (data may not be available or mode not supported)"
			((failed_count++))
		fi
	done

	print_status "Go chart generation complete: ${generated_count} successful, ${failed_count} failed"
}

# Generate relative/absolute variants for burndown modes
generate_go_variants() {
	print_status "Generating Go chart variants..."

	local burndown_modes=(
		"burndown-project"
		"burndown-file"
		"burndown-person"
	)

	for mode in "${burndown_modes[@]}"; do
		# Relative version
		local relative_file="${OUTPUT_DIR}/go/${mode}_relative.png"
		print_status "  Generating ${mode} (relative)..."

		if [ "$QUIET" = "true" ]; then
			./labours-go -i "${INPUT_FILE}" -m "${mode}" --relative -o "${relative_file}" --quiet 2>/dev/null
		else
			./labours-go -i "${INPUT_FILE}" -m "${mode}" --relative -o "${relative_file}"
		fi

		if [ $? -eq 0 ] && [ -f "${relative_file}" ]; then
			print_status "    ✅ Generated: ${relative_file}"
		else
			print_warning "    ❌ Failed: ${mode} relative"
		fi
	done
}

# Generate Python charts (if available)
generate_python_charts() {
	print_status "Checking for Python labours availability..."

	# Check if hercules Python is available
	local hercules_python=""

	# Common locations for hercules
	local possible_paths=(
		"/home/christian/Code/hercules/python"
		"../hercules/python"
		"../../hercules/python"
		"$HOME/Code/hercules/python"
		"/usr/local/src/hercules/python"
	)

	for path in "${possible_paths[@]}"; do
		if [ -d "$path" ] && [ -f "$path/labours/__main__.py" ]; then
			hercules_python="$path"
			break
		fi
	done

	if [ -z "$hercules_python" ]; then
		print_warning "Python labours not found. Skipping Python chart generation."
		print_status "To enable Python comparison:"
		print_status "  1. Clone hercules: git clone https://github.com/src-d/hercules"
		print_status "  2. Place it adjacent to labours-go directory"
		return
	fi

	print_status "Found Python labours at: ${hercules_python}"

	# Generate Python charts
	local modes=("burndown-project" "ownership") # Start with modes most likely to work

	for mode in "${modes[@]}"; do
		local output_file="${OUTPUT_DIR}/python/${mode}.png"
		print_status "  Generating Python ${mode}..."

		# Get absolute paths before entering subshell
		local current_dir="$(pwd)"
		local abs_input_file="${current_dir}/${INPUT_FILE}"
		local abs_output_file="${current_dir}/${output_file}"

		(
			cd "$hercules_python"
			# Use the hercules Python environment
			export PYTHONPATH="${hercules_python}:${PYTHONPATH:-}"
			python -m labours -i "${abs_input_file}" -m "${mode}" -o "${abs_output_file}" 2>/dev/null
		)

		# Python creates subdirectories, so check for both direct file and subdirectory pattern
		local python_subdir="${OUTPUT_DIR}/python/${mode}"
		local python_project_file="${python_subdir}/project.png"

		if [ -f "${output_file}" ]; then
			print_status "    ✅ Generated: ${output_file}"
		elif [ -f "${python_project_file}" ]; then
			# Python created a subdirectory - move to expected location
			mv "${python_project_file}" "${output_file}" 2>/dev/null
			rmdir "${python_subdir}" 2>/dev/null || true
			print_status "    ✅ Generated: ${output_file} (moved from subdirectory)"
		else
			print_warning "    ❌ Failed: Python ${mode}"
		fi

		# Generate relative version for burndown
		if [[ $mode == burndown* ]]; then
			local relative_file="${OUTPUT_DIR}/python/${mode}_relative.png"
			print_status "  Generating Python ${mode} (relative)..."

			# Get absolute paths for relative version
			local abs_relative_file="${current_dir}/${relative_file}"

			(
				cd "$hercules_python"
				# Use the hercules Python environment
				export PYTHONPATH="${hercules_python}:${PYTHONPATH:-}"
				python -m labours -i "${abs_input_file}" -m "${mode}" --relative -o "${abs_relative_file}" 2>/dev/null
			)

			# Handle Python subdirectory creation for relative version too
			local python_relative_subdir="${OUTPUT_DIR}/python/${mode}_relative"
			local python_relative_project_file="${python_relative_subdir}/project.png"

			if [ -f "${relative_file}" ]; then
				print_status "    ✅ Generated: ${relative_file}"
			elif [ -f "${python_relative_project_file}" ]; then
				mv "${python_relative_project_file}" "${relative_file}" 2>/dev/null
				rmdir "${python_relative_subdir}" 2>/dev/null || true
				print_status "    ✅ Generated: ${relative_file} (moved from subdirectory)"
			fi
		fi
	done
}

# Run visual comparison tests
run_visual_tests() {
	print_status "Running visual similarity tests..."

	# Check if Go test framework is available
	if [ -f "test/visual/similarity.go" ]; then
		print_status "  Running visual framework demo..."
		if just test-visual-demo >"${OUTPUT_DIR}/visual_test_results.log" 2>&1; then
			print_status "    ✅ Visual tests passed"
		else
			print_warning "    ⚠️  Visual tests had issues (check ${OUTPUT_DIR}/visual_test_results.log)"
		fi

		# Try Python compatibility test
		print_status "  Running Python compatibility test..."
		if just test-python-compat >"${OUTPUT_DIR}/python_compat_results.log" 2>&1; then
			print_status "    ✅ Python compatibility test completed"
		else
			print_warning "    ⚠️  Python compatibility test had issues"
		fi
	fi
}

# Generate summary report
generate_report() {
	local report_file="${OUTPUT_DIR}/generation_report.md"

	cat >"$report_file" <<EOF
# Chart Generation Report

**Generated**: $(date)
**Input File**: ${INPUT_FILE}
**Output Directory**: ${OUTPUT_DIR}

## Go Charts Generated

$(ls -la "${OUTPUT_DIR}/go/" 2>/dev/null | grep '\.png$' | wc -l) files created:

\`\`\`
$(ls "${OUTPUT_DIR}/go/"*.png 2>/dev/null | sort || echo "No files generated")
\`\`\`

## Python Charts Generated

$(ls -la "${OUTPUT_DIR}/python/" 2>/dev/null | grep '\.png$' | wc -l) files created:

\`\`\`
$(ls "${OUTPUT_DIR}/python/"*.png 2>/dev/null | sort || echo "No files generated")  
\`\`\`

## Visual Comparison

Run these commands to compare charts:

\`\`\`bash
# View Go charts
open ${OUTPUT_DIR}/go/burndown-project.png

# Compare Go vs Python (if available)
open ${OUTPUT_DIR}/go/burndown-project.png ${OUTPUT_DIR}/python/burndown-project.png

# Run visual similarity tests
just test-visual-demo
\`\`\`

## File Sizes

$(find "${OUTPUT_DIR}" -name "*.png" -exec ls -lh {} \; 2>/dev/null | sort -k5 -h || echo "No PNG files found")

---
Generated by labours-go chart generation script
EOF

	print_status "Report saved: ${report_file}"
}

# Main execution
main() {
	print_header

	# Validate environment
	check_binary
	check_input
	setup_output

	# Generate charts
	generate_go_charts
	generate_go_variants
	generate_python_charts

	# Run tests
	run_visual_tests

	# Create report
	generate_report

	print_header
	print_status "Chart generation complete!"
	print_status "Output directory: ${OUTPUT_DIR}"
	print_status "View report: ${OUTPUT_DIR}/generation_report.md"

	# Show quick summary
	local go_count=$(ls "${OUTPUT_DIR}/go/"*.png 2>/dev/null | wc -l)
	local python_count=$(ls "${OUTPUT_DIR}/python/"*.png 2>/dev/null | wc -l)

	echo
	echo -e "${GREEN}📊 Summary:${NC}"
	echo -e "  Go charts: ${go_count}"
	echo -e "  Python charts: ${python_count}"
	echo -e "  Total files: $((go_count + python_count))"
	echo
}

# Handle command line arguments
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
	echo "Usage: $0 [INPUT_FILE] [OUTPUT_DIR]"
	echo
	echo "Generate comprehensive chart suite for visual comparison"
	echo
	echo "Arguments:"
	echo "  INPUT_FILE    Input data file (default: example_data/hercules_burndown.yaml)"
	echo "  OUTPUT_DIR    Output directory (default: visual_output)"
	echo
	echo "Environment variables:"
	echo "  QUIET=true    Suppress verbose output"
	echo
	echo "Examples:"
	echo "  $0"
	echo "  $0 data/my_data.yaml my_charts/"
	echo "  QUIET=true $0"
	exit 0
fi

# Run main function
main "$@"
