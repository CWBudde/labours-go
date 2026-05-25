package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/spf13/viper"
	"labours-go/internal/readers"
)

// parseFlexibleDate parses a date string into a time.Time object.
// Returns an error if the date cannot be parsed.
func parseFlexibleDate(dateStr string) (time.Time, error) {
	parsedDate, err := dateparse.ParseAny(dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %v", err)
	}
	return parsedDate, nil
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func parseDates() (startTime *time.Time, endTime *time.Time) {
	if startTimeStr := viper.GetString("start-date"); startTimeStr != "" {
		parsedStartTime, err := parseFlexibleDate(startTimeStr)
		if err != nil {
			fmt.Printf("Error parsing start date: %v\n", err)
			os.Exit(1)
		}
		startTime = &parsedStartTime
	}

	if endTimeStr := viper.GetString("end-date"); endTimeStr != "" {
		parsedEndTime, err := parseFlexibleDate(endTimeStr)
		if err != nil {
			fmt.Printf("Error parsing end date: %v\n", err)
			os.Exit(1)
		}
		endTime = &parsedEndTime
	}

	return startTime, endTime
}

func validateDateRange(startTime, endTime *time.Time) {
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		fmt.Println("Error: end date must be after start date")
		os.Exit(1)
	}
}

func detectAndReadInput(input, inputFormat string) readers.Reader {
	reader, err := readers.DetectAndReadInput(input, inputFormat)
	if err != nil {
		fmt.Printf("Error detecting or reading input: %v\n", err)
		os.Exit(1)
	}
	return reader
}

var validModeNames = map[string]struct{}{
	"all":                     {},
	"burndown":                {},
	"burndown-file":           {},
	"burndown-person":         {},
	"burndown-project":        {},
	"burndown-repository":     {},
	"burndown-repos-combined": {},
	"bus-factor":              {},
	"couples":                 {},
	"couples-files":           {},
	"couples-people":          {},
	"couples-shotness":        {},
	"devs":                    {},
	"devs-efforts":            {},
	"devs-parallel":           {},
	"hotspot-risk":            {},
	"knowledge-diffusion":     {},
	"languages":               {},
	"old-vs-new":              {},
	"overwrites-matrix":       {},
	"ownership":               {},
	"ownership-concentration": {},
	"refactoring-proxy":       {},
	"run-times":               {},
	"sentiment":               {},
	"shotness":                {},
	"temporal-activity":       {},
}

var pythonAllModes = []string{
	"burndown-project",
	"overwrites-matrix",
	"ownership",
	"couples-files",
	"couples-people",
	"couples-shotness",
	"shotness",
	"devs",
	"devs-efforts",
}

func resolveModes() []string {
	rawModes := append([]string{}, viper.GetStringSlice("modes")...)
	rawModes = append(rawModes, viper.GetStringSlice("mode")...)
	modes, err := resolveModesFrom(rawModes)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return modes
}

func resolveModesFrom(rawModes []string) ([]string, error) {
	modes := splitModeValues(rawModes)
	if len(modes) == 0 {
		return nil, nil
	}

	// Handle mode aliases for Python compatibility
	var resolvedModes []string
	for _, mode := range modes {
		if !isValidMode(mode) {
			return nil, fmt.Errorf("unknown mode: %s", mode)
		}
		switch mode {
		case "burndown":
			// Python compatibility: burndown defaults to burndown-project
			resolvedModes = append(resolvedModes, "burndown-project")
		case "couples":
			// Python compatibility: couples runs all coupling analyses
			resolvedModes = append(resolvedModes, "couples-files", "couples-people", "couples-shotness")
		default:
			resolvedModes = append(resolvedModes, mode)
		}
	}
	modes = resolvedModes

	if contains(modes, "all") {
		// Match Python's "all" mode composition exactly
		modes = append([]string{}, pythonAllModes...)
	}
	return modes, nil
}

func splitModeValues(rawModes []string) []string {
	var modes []string
	for _, raw := range rawModes {
		for _, part := range strings.Split(raw, ",") {
			mode := strings.TrimSpace(part)
			if mode != "" {
				modes = append(modes, mode)
			}
		}
	}
	return modes
}

func isValidMode(mode string) bool {
	_, ok := validModeNames[mode]
	return ok
}

func normalizeInputFormat(inputFormat string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(inputFormat))
	if format == "" {
		format = "auto"
	}
	switch format {
	case "auto", "yaml", "pb":
		return format, nil
	default:
		return "", fmt.Errorf("unsupported input format %q: expected auto, yaml, or pb", inputFormat)
	}
}

// isExecutable checks if a file exists and is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode()&0o111 != 0
}

// isGitRepository checks if a directory is a git repository
func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// detectOutputFormat determines the output format from file extension or backend flag
func detectOutputFormat(outputPath string) string {
	// Check if backend flag overrides format detection
	if backend := viper.GetString("backend"); backend != "" {
		switch strings.ToLower(backend) {
		case "pdf":
			return "pdf"
		case "png":
			return "png"
		case "svg":
			return "svg"
		case "auto":
			// Fall through to extension detection
		default:
			// For unknown backends, fall through to extension detection
		}
	}

	// Detect from file extension
	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".pdf":
		return "pdf"
	case ".svg":
		return "svg"
	case ".png", "":
		return "png" // Default to PNG
	default:
		return "png" // Default to PNG for unknown extensions
	}
}

// generateOutputPath generates the output path with the appropriate file extension
func generateOutputPath(basePath string, format string) string {
	ext := "." + format

	// If basePath already has the correct extension, use it as-is
	if strings.HasSuffix(strings.ToLower(basePath), ext) {
		return basePath
	}

	// Remove any existing extension and add the correct one
	nameWithoutExt := strings.TrimSuffix(basePath, filepath.Ext(basePath))
	return nameWithoutExt + ext
}

type outputConventionKind string

const (
	outputSingleFile outputConventionKind = "single-file"
	outputFileFanout outputConventionKind = "file-fanout"
	outputAssetDir   outputConventionKind = "asset-directory"
	outputCompanions outputConventionKind = "primary-file-with-companions"
)

type outputConvention struct {
	Kind        outputConventionKind
	Description string
	Assets      []string
}

var modeOutputConventions = map[string]outputConvention{
	"burndown-project": {
		Kind:        outputSingleFile,
		Description: "writes exactly the requested chart file",
		Assets:      []string{"<output>"},
	},
	"burndown-file": {
		Kind:        outputFileFanout,
		Description: "uses the requested file path as a basename and writes one chart per file",
		Assets:      []string{"<base>_<sanitized-file><ext>"},
	},
	"burndown-person": {
		Kind:        outputFileFanout,
		Description: "uses the requested file path as a basename and writes one chart per person",
		Assets:      []string{"<base>_<sanitized-person><ext>"},
	},
	"burndown-repository": {
		Kind:        outputAssetDir,
		Description: "writes one chart per repository into the requested asset directory",
		Assets:      []string{"burndown-repository_<sanitized-repository>.png", "burndown-repository_<sanitized-repository>.svg"},
	},
	"burndown-repos-combined": {
		Kind:        outputSingleFile,
		Description: "writes exactly the requested combined repository chart file",
		Assets:      []string{"<output>"},
	},
	"overwrites-matrix": {
		Kind:        outputSingleFile,
		Description: "writes exactly the requested matrix chart file",
		Assets:      []string{"<output>"},
	},
	"ownership": {
		Kind:        outputSingleFile,
		Description: "writes exactly the requested ownership chart file",
		Assets:      []string{"<output>"},
	},
	"couples-files": {
		Kind:        outputAssetDir,
		Description: "writes TensorBoard-style file coupling projector assets into the requested directory",
		Assets:      []string{"files_vocabulary.tsv", "files_vectors.tsv", "files_metadata.tsv"},
	},
	"couples-people": {
		Kind:        outputAssetDir,
		Description: "writes TensorBoard-style people coupling projector assets into the requested directory",
		Assets:      []string{"people_vocabulary.tsv", "people_vectors.tsv", "people_metadata.tsv"},
	},
	"couples-shotness": {
		Kind:        outputAssetDir,
		Description: "writes shotness coupling charts into the requested directory",
		Assets:      []string{"shotness_coupling_heatmap.png", "shotness_coupling_heatmap.svg", "top_shotness_coupling_pairs.png", "top_shotness_coupling_pairs.svg"},
	},
	"shotness": {
		Kind:        outputAssetDir,
		Description: "prints statistics and writes PNG/SVG charts into the requested directory",
		Assets:      []string{"shotness.png", "shotness.svg"},
	},
	"devs": {
		Kind:        outputSingleFile,
		Description: "writes exactly the requested developer chart file",
		Assets:      []string{"<output>"},
	},
	"devs-efforts": {
		Kind:        outputSingleFile,
		Description: "writes the requested developer efforts chart file; --devs-efforts-detail also writes a Go-only productivity ranking sibling",
		Assets:      []string{"<output>", "<base>_productivity_ranking<ext> (only with --devs-efforts-detail)"},
	},
	"old-vs-new": {
		Kind:        outputSingleFile,
		Description: "writes exactly the requested additions-vs-changes chart file",
		Assets:      []string{"<output>"},
	},
	"languages": {
		Kind:        outputSingleFile,
		Description: "writes the requested chart file; direct directory calls write languages.png and languages.svg",
		Assets:      []string{"<output>"},
	},
	"temporal-activity": {
		Kind:        outputCompanions,
		Description: "writes the Python labours sibling set: eight stacked bar charts (weekdays/hours/months/weeks × commits/lines) plus two weekday×hour heatmaps",
		Assets: []string{
			"<base>_weekdays_commits<ext>", "<base>_hours_commits<ext>", "<base>_months_commits<ext>", "<base>_weeks_commits<ext>",
			"<base>_weekdays_lines<ext>", "<base>_hours_lines<ext>", "<base>_months_lines<ext>", "<base>_weeks_lines<ext>",
			"<base>_heatmap_commits<ext>", "<base>_heatmap_lines<ext>",
		},
	},
	"devs-parallel": {
		Kind:        outputSingleFile,
		Description: "writes the parallel-coordinates developer chart and prints a parallel-development summary; --devs-parallel-detail also writes a Go-only concurrency timeline sibling",
		Assets:      []string{"<output>", "<base>_concurrency_timeline<ext> (only with --devs-parallel-detail)"},
	},
	"run-times": {
		Kind:        outputSingleFile,
		Description: "prints the runtime summary; the Go-only breakdown chart is written only with --run-times-detail (Python parity: text-only)",
		Assets:      []string{"<output> (only with --run-times-detail)"},
	},
	"bus-factor": {
		Kind:        outputCompanions,
		Description: "writes timeline, gauge, and subsystem bus-factor sibling charts",
		Assets:      []string{"<base>_timeline<ext>", "<base>_gauge<ext>", "<base>_subsystems<ext>"},
	},
	"ownership-concentration": {
		Kind:        outputCompanions,
		Description: "writes timeline and subsystem ownership concentration charts as sibling files",
		Assets:      []string{"<base>_timeline<ext>", "<base>_subsystems<ext>"},
	},
	"knowledge-diffusion": {
		Kind:        outputCompanions,
		Description: "writes distribution, silos, and Lorenz-curve sibling charts; --knowledge-diffusion-detail also writes a Go-only trend sibling",
		Assets:      []string{"<base>_distribution<ext>", "<base>_silos<ext>", "<base>_lorenz<ext>", "<base>_trend<ext> (only with --knowledge-diffusion-detail)"},
	},
	"hotspot-risk": {
		Kind:        outputCompanions,
		Description: "writes the requested risk chart plus a TSV table sibling",
		Assets:      []string{"<output>", "<base>_table.tsv"},
	},
	"sentiment": {
		Kind:        outputAssetDir,
		Description: "writes sentiment charts into the requested directory",
		Assets:      []string{"sentiment-overview.png", "sentiment-overview.svg", "sentiment-developers.png", "sentiment-developers.svg", "sentiment-languages.png", "sentiment-languages.svg"},
	},
	"refactoring-proxy": {
		Kind:        outputSingleFile,
		Description: "writes exactly the requested refactoring proxy chart file",
		Assets:      []string{"<output>"},
	},
}

func planModeOutput(baseOutput, mode string, modeCount int) string {
	if outputConventionFor(mode).Kind == outputAssetDir {
		return planMultiAssetModeOutput(baseOutput)
	}

	format := detectOutputFormat(baseOutput)
	if baseOutput == "" {
		return generateOutputPath(mode, format)
	}

	if isDirectoryPath(baseOutput) {
		return generateOutputPath(filepath.Join(baseOutput, mode), detectOutputFormat(""))
	}

	if modeCount > 1 {
		if filepath.Ext(baseOutput) == "" {
			return generateOutputPath(filepath.Join(baseOutput, mode), detectOutputFormat(""))
		}
		return generateOutputPath(filepath.Join(filepath.Dir(baseOutput), mode), format)
	}

	return generateOutputPath(baseOutput, format)
}

func planMultiAssetModeOutput(baseOutput string) string {
	if baseOutput == "" {
		return "."
	}
	if outputLooksLikeFile(baseOutput) {
		return filepath.Dir(baseOutput)
	}
	return baseOutput
}

func isMultiAssetMode(mode string) bool {
	return outputConventionFor(mode).Kind == outputAssetDir
}

func outputConventionFor(mode string) outputConvention {
	if convention, ok := modeOutputConventions[mode]; ok {
		return convention
	}
	return outputConvention{
		Kind:        outputSingleFile,
		Description: "writes exactly the requested chart file",
		Assets:      []string{"<output>"},
	}
}

func isDirectoryPath(path string) bool {
	if path == "" {
		return false
	}
	if strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\") {
		return true
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func outputLooksLikeFile(path string) bool {
	if isDirectoryPath(path) {
		return false
	}
	return filepath.Ext(path) != ""
}

// mapModesToHerculesAnalyses maps labours-go modes to hercules analysis types
func mapModesToHerculesAnalyses(modes []string) []string {
	analysisMap := make(map[string]bool)

	for _, mode := range modes {
		switch {
		case strings.HasPrefix(mode, "burndown"):
			analysisMap["burndown"] = true
		case mode == "devs" || mode == "devs-efforts":
			analysisMap["devs"] = true
		case strings.HasPrefix(mode, "couples"):
			analysisMap["couples"] = true
		case mode == "ownership":
			analysisMap["file-history"] = true
		case mode == "overwrites-matrix":
			analysisMap["couples"] = true // overwrites uses couples data
		}
	}

	result := make([]string, 0, len(analysisMap))
	for analysis := range analysisMap {
		result = append(result, analysis)
	}

	// Default to burndown if no specific analyses found
	if len(result) == 0 {
		result = []string{"burndown"}
	}

	return result
}

// runHerculesAndVisualize runs hercules analysis and then visualizes with labours-go
func runHerculesAndVisualize(herculesPath, repoPath, analysis string) error {
	// Generate temporary file for hercules output
	outputFile := fmt.Sprintf("/tmp/hercules_%s.yaml", analysis)

	// Build hercules command
	var herculesFlags []string
	switch analysis {
	case "burndown":
		herculesFlags = []string{"--burndown", "--burndown-files", "--burndown-people"}
	case "devs":
		herculesFlags = []string{"--devs"}
	case "couples":
		herculesFlags = []string{"--couples"}
	case "file-history":
		herculesFlags = []string{"--file-history"}
	default:
		herculesFlags = []string{"--" + analysis}
	}

	// Add any additional user-specified flags
	if userFlags := viper.GetString("hercules-flags"); userFlags != "" {
		herculesFlags = append(herculesFlags, strings.Fields(userFlags)...)
	}

	// Add repository path
	herculesFlags = append(herculesFlags, repoPath)

	fmt.Printf("Running hercules %s analysis...\n", analysis)

	// Execute hercules
	cmd := exec.Command(herculesPath, herculesFlags...) // #nosec G204 - user-configured Hercules executable is the purpose of this helper.
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("hercules command failed: %v", err)
	}

	// Write output to temporary file
	if err := os.WriteFile(outputFile, output, 0o600); err != nil {
		return fmt.Errorf("failed to write hercules output: %v", err)
	}

	fmt.Printf("Hercules analysis complete, creating visualizations...\n")

	// Determine labours-go modes for this analysis
	var laboursGoModes []string
	switch analysis {
	case "burndown":
		laboursGoModes = []string{"burndown-project"}
	case "devs":
		laboursGoModes = []string{"devs"}
	case "couples":
		laboursGoModes = []string{"couples-files"}
	case "file-history":
		laboursGoModes = []string{"ownership"}
	}

	// Run visualization for each mode
	for _, mode := range laboursGoModes {
		outputPath := viper.GetString("output")
		var format string

		if outputPath == "" {
			// Default to centralized analysis_results directory
			if err := os.MkdirAll("analysis_results", 0o750); err != nil {
				return fmt.Errorf("failed to create analysis_results directory: %v", err)
			}
			format = detectOutputFormat("") // Will use backend flag or default to PNG
			basePath := fmt.Sprintf("analysis_results/%s_%s", analysis, mode)
			outputPath = generateOutputPath(basePath, format)
		} else {
			// If output is a directory, create filename
			if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
				format = detectOutputFormat("") // Will use backend flag or default to PNG
				basePath := filepath.Join(outputPath, fmt.Sprintf("%s_%s", analysis, mode))
				outputPath = generateOutputPath(basePath, format)
			} else {
				// outputPath is a file, detect format from it
				format = detectOutputFormat(outputPath)
				outputPath = generateOutputPath(outputPath, format)
			}
		}

		fmt.Printf("Creating %s visualization...\n", mode)

		// Read the hercules output and create visualization
		reader := detectAndReadInput(outputFile, "yaml")
		startDate, endDate := parseDates()

		executeModes([]string{mode}, reader, outputPath, startDate, endDate)

		fmt.Printf("Saved: %s\n", outputPath)
	}

	// Clean up temporary file
	if err := os.Remove(outputFile); err != nil {
		return fmt.Errorf("failed to remove temporary output file: %v", err)
	}

	return nil
}
