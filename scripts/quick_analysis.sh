#!/bin/bash

# quick_analysis.sh - Quick Git Repository Analysis
# Simple wrapper for common hercules + labours-go workflows

set -e

# Quick usage check
if [[ $# -eq 0 ]] || [[ $1 == "-h" ]] || [[ $1 == "--help" ]]; then
	echo "Quick Git Analysis with Hercules + Labours-Go"
	echo ""
	echo "Usage: $0 <repository-path> [output-dir]"
	echo ""
	echo "This script will:"
	echo "  1. Analyze the repository with Hercules (burndown & developer stats)"
	echo "  2. Generate visualizations with Labours-Go"
	echo "  3. Save results to output directory"
	echo ""
	echo "Examples:"
	echo "  $0 /path/to/my-repo"
	echo "  $0 /path/to/my-repo ./analysis-results"
	echo ""
	echo "Requirements:"
	echo "  - Hercules binary at /home/christian/Code/hercules/hercules"
	echo "  - Labours-Go built in current directory"
	exit 0
fi

REPO_PATH="$1"
OUTPUT_DIR="${2:-analysis_results/quick_analysis_$(date +%Y%m%d_%H%M%S)}"

echo "🚀 Starting Quick Git Analysis..."
echo "📁 Repository: $REPO_PATH"
echo "📊 Output: $OUTPUT_DIR"

# Check if repo exists
if [[ ! -d "$REPO_PATH/.git" ]]; then
	echo "❌ Error: $REPO_PATH is not a git repository"
	exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Check for hercules
HERCULES="/home/christian/Code/hercules/hercules"
if [[ ! -x $HERCULES ]]; then
	echo "❌ Error: Hercules not found at $HERCULES"
	exit 1
fi

# Build labours-go if needed
if [[ ! -x "./labours-go" ]]; then
	echo "🔨 Building labours-go..."
	go build -o labours-go
fi

echo "📈 Analyzing with Hercules..."

# Generate burndown analysis
echo "  → Burndown analysis..."
"$HERCULES" --burndown --burndown-files --burndown-people "$REPO_PATH" >"$OUTPUT_DIR/burndown.yaml" 2>/dev/null

# Generate developer stats
echo "  → Developer statistics..."
"$HERCULES" --devs "$REPO_PATH" >"$OUTPUT_DIR/devs.yaml" 2>/dev/null

echo "🎨 Creating visualizations with Labours-Go..."

# Create burndown charts
echo "  → Project burndown chart..."
./labours-go -i "$OUTPUT_DIR/burndown.yaml" -m burndown-project -o "$OUTPUT_DIR/burndown_project.png" 2>/dev/null

echo "  → File burndown chart..."
./labours-go -i "$OUTPUT_DIR/burndown.yaml" -m burndown-file -o "$OUTPUT_DIR/burndown_files.png" 2>/dev/null || echo "    (skipped - no file data)"

echo "  → Developer burndown chart..."
./labours-go -i "$OUTPUT_DIR/burndown.yaml" -m burndown-person -o "$OUTPUT_DIR/burndown_people.png" 2>/dev/null || echo "    (skipped - no people data)"

echo "  → Developer statistics..."
./labours-go -i "$OUTPUT_DIR/devs.yaml" -m devs -o "$OUTPUT_DIR/developer_stats.png" 2>/dev/null

echo "✅ Analysis Complete!"
echo ""
echo "📋 Results saved to: $OUTPUT_DIR"
echo "Generated files:"
ls -la "$OUTPUT_DIR"
echo ""
echo "🎯 Key files to check:"
echo "  • $OUTPUT_DIR/burndown_project.png - Project code evolution"
echo "  • $OUTPUT_DIR/developer_stats.png - Developer contributions"
