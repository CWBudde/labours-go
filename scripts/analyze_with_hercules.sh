#!/bin/bash

# analyze_with_hercules.sh - Complete Git Analytics Pipeline
# Uses Hercules for data analysis and Labours-Go for visualization

set -e

# Configuration
HERCULES_BINARY="${HERCULES_BINARY:-/home/christian/Code/hercules/hercules}"
LABOURS_GO_BINARY="${LABOURS_GO_BINARY:-./labours-go}"
OUTPUT_DIR="${OUTPUT_DIR:-./analysis_results/hercules_analysis}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_usage() {
	echo "Usage: $0 <repository-path> [options]"
	echo ""
	echo "Arguments:"
	echo "  repository-path     Path to the git repository to analyze"
	echo ""
	echo "Options:"
	echo "  -o, --output DIR    Output directory for results (default: ./analysis_results/hercules_analysis)"
	echo "  -m, --modes LIST    Comma-separated list of analysis modes to run"
	echo "                      Available: burndown,devs,couples,ownership,overwrites"
	echo "                      Default: burndown,devs"
	echo "  -t, --theme NAME    Visualization theme (default,dark,minimal,vibrant)"
	echo "  --hercules PATH     Path to hercules binary"
	echo "  --labours-go PATH   Path to labours-go binary"
	echo "  --pb                Use Protocol Buffer format (default: YAML)"
	echo "  -h, --help          Show this help message"
	echo ""
	echo "Examples:"
	echo "  $0 /path/to/repo"
	echo "  $0 /path/to/repo -m burndown,devs,couples -o results/ -t dark"
	echo "  $0 /path/to/repo --pb --hercules /custom/hercules"
}

log_info() {
	echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
	echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
	echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
	echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
	log_info "Checking dependencies..."

	if [[ ! -x $HERCULES_BINARY ]]; then
		log_error "Hercules binary not found at: $HERCULES_BINARY"
		log_info "Please set HERCULES_BINARY environment variable or use --hercules flag"
		exit 1
	fi

	if [[ ! -x $LABOURS_GO_BINARY ]]; then
		log_warning "Labours-Go binary not found at: $LABOURS_GO_BINARY"
		log_info "Attempting to build labours-go..."
		if ! go build -o labours-go; then
			log_error "Failed to build labours-go"
			exit 1
		fi
		LABOURS_GO_BINARY="./labours-go"
	fi

	log_success "All dependencies found"
}

run_hercules_analysis() {
	local repo_path="$1"
	local mode="$2"
	local output_file="$3"

	log_info "Running Hercules analysis: $mode"

	local hercules_flags=""
	case "$mode" in
	"burndown")
		hercules_flags="--burndown --burndown-files --burndown-people"
		;;
	"devs")
		hercules_flags="--devs"
		;;
	"couples")
		hercules_flags="--couples"
		;;
	"ownership")
		hercules_flags="--file-history"
		;;
	"overwrites")
		hercules_flags="--couples" # couples analysis provides overwrite matrix data
		;;
	*)
		log_warning "Unknown hercules mode: $mode, using default flags"
		hercules_flags="--$mode"
		;;
	esac

	if [[ $USE_PB == "true" ]]; then
		hercules_flags="$hercules_flags --pb"
	fi

	if ! "$HERCULES_BINARY" $hercules_flags "$repo_path" >"$output_file" 2>/dev/null; then
		log_error "Hercules analysis failed for mode: $mode"
		return 1
	fi

	log_success "Hercules analysis completed: $mode"
	return 0
}

run_labours_visualization() {
	local input_file="$1"
	local mode="$2"
	local output_prefix="$3"

	log_info "Running Labours-Go visualization: $mode"

	# Map hercules analysis to labours-go modes
	local labours_mode=""
	case "$mode" in
	"burndown")
		labours_mode="burndown-project,burndown-file,burndown-person"
		;;
	"devs")
		labours_mode="devs"
		;;
	"couples")
		labours_mode="couples-files"
		;;
	"ownership")
		labours_mode="ownership"
		;;
	"overwrites")
		labours_mode="overwrites-matrix"
		;;
	*)
		labours_mode="$mode"
		;;
	esac

	# Split comma-separated modes and run each
	IFS=',' read -ra MODES <<<"$labours_mode"
	for labours_single_mode in "${MODES[@]}"; do
		local output_file="${output_prefix}_${labours_single_mode}"

		local labours_flags="-i $input_file -m $labours_single_mode"

		if [[ -n $THEME ]]; then
			labours_flags="$labours_flags --theme $THEME"
		fi

		# Try PNG first, fallback to SVG
		if "$LABOURS_GO_BINARY" $labours_flags -o "${output_file}.png" 2>/dev/null; then
			log_success "Generated: ${output_file}.png"
		elif "$LABOURS_GO_BINARY" $labours_flags -o "${output_file}.svg" 2>/dev/null; then
			log_success "Generated: ${output_file}.svg"
		else
			log_warning "Failed to generate visualization for: $labours_single_mode"
		fi
	done
}

main() {
	local repo_path=""
	local modes="burndown,devs"
	local output_dir="./hercules_analysis"
	local theme=""
	local use_pb="false"

	# Parse command line arguments
	while [[ $# -gt 0 ]]; do
		case $1 in
		-h | --help)
			print_usage
			exit 0
			;;
		-o | --output)
			output_dir="$2"
			shift 2
			;;
		-m | --modes)
			modes="$2"
			shift 2
			;;
		-t | --theme)
			theme="$2"
			shift 2
			;;
		--hercules)
			HERCULES_BINARY="$2"
			shift 2
			;;
		--labours-go)
			LABOURS_GO_BINARY="$2"
			shift 2
			;;
		--pb)
			use_pb="true"
			shift
			;;
		-*)
			log_error "Unknown option: $1"
			print_usage
			exit 1
			;;
		*)
			if [[ -z $repo_path ]]; then
				repo_path="$1"
			else
				log_error "Multiple repository paths provided"
				print_usage
				exit 1
			fi
			shift
			;;
		esac
	done

	if [[ -z $repo_path ]]; then
		log_error "Repository path is required"
		print_usage
		exit 1
	fi

	if [[ ! -d $repo_path ]]; then
		log_error "Repository path does not exist: $repo_path"
		exit 1
	fi

	if [[ ! -d "$repo_path/.git" ]]; then
		log_error "Not a git repository: $repo_path"
		exit 1
	fi

	# Set global variables
	OUTPUT_DIR="$output_dir"
	THEME="$theme"
	USE_PB="$use_pb"

	# Create output directory
	mkdir -p "$OUTPUT_DIR"

	# Check dependencies
	check_dependencies

	log_info "Starting Git Analytics Pipeline"
	log_info "Repository: $repo_path"
	log_info "Output Directory: $OUTPUT_DIR"
	log_info "Analysis Modes: $modes"
	log_info "Format: $([ "$USE_PB" == "true" ] && echo "Protocol Buffer" || echo "YAML")"

	# Process each analysis mode
	IFS=',' read -ra MODE_LIST <<<"$modes"
	for mode in "${MODE_LIST[@]}"; do
		mode=$(echo "$mode" | xargs) # trim whitespace

		local data_file="${OUTPUT_DIR}/hercules_${mode}"
		if [[ $USE_PB == "true" ]]; then
			data_file="${data_file}.pb"
		else
			data_file="${data_file}.yaml"
		fi

		# Run Hercules analysis
		if run_hercules_analysis "$repo_path" "$mode" "$data_file"; then
			# Run Labours-Go visualization
			run_labours_visualization "$data_file" "$mode" "${OUTPUT_DIR}/${mode}"
		else
			log_warning "Skipping visualization for failed analysis: $mode"
		fi
	done

	log_success "Git Analytics Pipeline completed!"
	log_info "Results saved to: $OUTPUT_DIR"
	log_info ""
	log_info "Generated files:"
	find "$OUTPUT_DIR" -type f | sort
}

# Run main function with all arguments
main "$@"
