package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"labours-go/internal/modes"
	"labours-go/internal/progress"
	"labours-go/internal/readers"
)

// Map of mode names to their handlers
var modeHandlers = map[string]func(reader readers.Reader, output string, startTime, endTime *time.Time) error{
	"burndown-project":        burndownProject,
	"burndown-file":           burndownFile,
	"burndown-person":         burndownPerson,
	"overwrites-matrix":       overwritesMatrix,
	"ownership":               ownershipBurndown,
	"couples-files":           couplesFiles,
	"couples-people":          couplesPeople,
	"couples-shotness":        couplesShotness,
	"shotness":                shotness,
	"devs":                    devs,
	"devs-efforts":            devsEfforts,
	"old-vs-new":              oldVsNew,
	"languages":               languages,
	"temporal-activity":       temporalActivity,
	"devs-parallel":           devsParallel,
	"run-times":               runTimes,
	"bus-factor":              busFactor,
	"ownership-concentration": ownershipConcentration,
	"knowledge-diffusion":     knowledgeDiffusion,
	"hotspot-risk":            hotspotRisk,
	"sentiment":               sentiment,
	"all":                     runAllModes,
}

func executeModes(modes []string, reader readers.Reader, output string, startTime, endTime *time.Time) {
	// Check if JSON output is requested
	jsonOutput := strings.HasSuffix(strings.ToLower(output), ".json")

	// Initialize progress tracking for multiple modes
	quiet := viper.GetBool("quiet")
	progEstimator := progress.NewProgressEstimator(!quiet)

	// If JSON output, collect all results and save as JSON
	if jsonOutput {
		results := make(map[string]interface{})

		if len(modes) > 1 {
			progEstimator.StartMultiOperation(len(modes), "Analysis Modes")
		}

		for _, mode := range modes {
			if len(modes) > 1 {
				progEstimator.NextOperation(fmt.Sprintf("Running %s", mode))
			}

			if !quiet {
				fmt.Printf("Running mode: %s\n", mode)
			}

			// For JSON output, collect data instead of generating plots
			if modeFunc, ok := modeHandlers[mode]; ok {
				// Create temporary directory for this mode's data
				tempDir := filepath.Join(os.TempDir(), "labours-json-"+mode)
				os.MkdirAll(tempDir, 0755)
				defer os.RemoveAll(tempDir)

				if err := modeFunc(reader, tempDir, startTime, endTime); err != nil {
					fmt.Printf("Error in mode %s: %v\n", mode, err)
					results[mode] = map[string]interface{}{
						"error": err.Error(),
					}
				} else {
					// Extract data from the mode (this would need to be enhanced per mode)
					results[mode] = extractModeDataForJSON(reader, mode)
				}
			} else {
				printModeUnavailable(mode)
				results[mode] = map[string]interface{}{
					"error": "mode not implemented",
				}
			}
		}

		if len(modes) > 1 {
			progEstimator.FinishMultiOperation()
		}

		// Save results as JSON
		if err := saveJSONResults(results, output); err != nil {
			fmt.Printf("Error saving JSON results: %v\n", err)
		} else if !quiet {
			fmt.Printf("Results saved as JSON to: %s\n", output)
		}
	} else {
		// Regular image output
		if len(modes) > 1 {
			// Start multi-mode progress tracking
			progEstimator.StartMultiOperation(len(modes), "Analysis Modes")

			for _, mode := range modes {
				progEstimator.NextOperation(fmt.Sprintf("Running %s", mode))

				if !quiet {
					fmt.Printf("Running mode: %s\n", mode)
				}

				if modeFunc, ok := modeHandlers[mode]; ok {
					formattedOutput := planModeOutput(output, mode, len(modes))
					if err := modeFunc(reader, formattedOutput, startTime, endTime); err != nil {
						fmt.Printf("Error in mode %s: %v\n", mode, err)
					}
				} else {
					printModeUnavailable(mode)
				}
			}

			progEstimator.FinishMultiOperation()
		} else {
			// Single mode - let the individual mode handle its own progress
			for _, mode := range modes {
				if !quiet {
					fmt.Printf("Running mode: %s\n", mode)
				}

				if modeFunc, ok := modeHandlers[mode]; ok {
					formattedOutput := planModeOutput(output, mode, len(modes))
					if err := modeFunc(reader, formattedOutput, startTime, endTime); err != nil {
						fmt.Printf("Error in mode %s: %v\n", mode, err)
					}
				} else {
					printModeUnavailable(mode)
				}
			}
		}
	}
}

func printModeUnavailable(mode string) {
	if isValidMode(mode) {
		fmt.Printf("Mode not implemented yet: %s\n", mode)
		return
	}
	fmt.Printf("Unknown mode: %s\n", mode)
}

func burndownProject(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	relative := viper.GetBool("relative")
	resample := viper.GetString("resample")
	// Use Python-compatible implementation
	return modes.GenerateBurndownProjectPython(reader, output, relative, resample)
}

func burndownFile(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	relative := viper.GetBool("relative")
	resample := viper.GetString("resample")
	// Use Python-compatible implementation
	return modes.GenerateBurndownFilePython(reader, output, relative, resample)
}

func burndownPerson(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	relative := viper.GetBool("relative")
	resample := viper.GetString("resample")
	return modes.BurndownPerson(reader, output, relative, startTime, endTime, resample)
}

func overwritesMatrix(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.OverwritesMatrix(reader, output)
}

func ownershipBurndown(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.OwnershipBurndown(reader, output)
}

func couplesFiles(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	// Note: --disable-projector flag is supported for Python compatibility but not used
	// Our Go implementation focuses on core coupling analysis without TensorFlow embeddings
	return modes.CouplesFiles(reader, output)
}

func couplesPeople(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.CouplesPeople(reader, output)
}

func couplesShotness(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.CouplesShotness(reader, output)
}

func shotness(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.Shotness(reader, output)
}

func devs(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	maxPeople := viper.GetInt("max-people")
	return modes.Devs(reader, output, maxPeople)
}

func devsEfforts(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	maxPeople := viper.GetInt("max-people")
	return modes.DevsEfforts(reader, output, maxPeople)
}

func oldVsNew(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	resample := viper.GetString("resample")
	return modes.OldVsNew(reader, output, startTime, endTime, resample)
}

func languages(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.Languages(reader, output)
}

func temporalActivity(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	legendThreshold := viper.GetInt("temporal-legend-threshold")
	singleColumnThreshold := viper.GetInt("temporal-legend-single-col-threshold")
	return modes.TemporalActivity(reader, output, legendThreshold, singleColumnThreshold)
}

func devsParallel(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.DevsParallel(reader, output)
}

func runTimes(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.RunTimes(reader, output)
}

func busFactor(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.BusFactor(reader, output)
}

func ownershipConcentration(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.OwnershipConcentration(reader, output)
}

func knowledgeDiffusion(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.KnowledgeDiffusion(reader, output)
}

func hotspotRisk(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.HotspotRisk(reader, output)
}

func sentiment(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	return modes.Sentiment(reader, output)
}

func runAllModes(reader readers.Reader, output string, startTime, endTime *time.Time) error {
	// 'all' mode runs the most commonly used analysis modes
	// This matches the Python labours behavior for the 'all' meta-mode
	allModes := []func(readers.Reader, string, *time.Time, *time.Time) error{
		burndownProject,
		devs,
		ownershipBurndown,
		couplesFiles,
		devsEfforts,
		languages,
	}

	modeNames := []string{
		"burndown-project",
		"devs",
		"ownership",
		"couples-files",
		"devs-efforts",
		"languages",
	}

	if !viper.GetBool("quiet") {
		fmt.Printf("Running 'all' mode: executing %d analysis modes\n", len(allModes))
	}

	for i, modeFunc := range allModes {
		if !viper.GetBool("quiet") {
			fmt.Printf("  Running %s...\n", modeNames[i])
		}

		if err := modeFunc(reader, output, startTime, endTime); err != nil {
			fmt.Printf("  Error in mode %s: %v\n", modeNames[i], err)
			// Continue with other modes even if one fails
		}
	}

	return nil
}

// extractModeDataForJSON extracts raw data from the reader for JSON output
func extractModeDataForJSON(reader readers.Reader, mode string) interface{} {
	switch mode {
	case "devs":
		if stats, err := reader.GetDeveloperStats(); err == nil {
			return map[string]interface{}{
				"developer_stats": stats,
			}
		}
	case "burndown-project":
		if header, name, matrix, err := reader.GetProjectBurndownWithHeader(); err == nil {
			return map[string]interface{}{
				"header": header,
				"name":   name,
				"matrix": matrix,
			}
		}
	case "ownership":
		if names, matrices, err := reader.GetOwnershipBurndown(); err == nil {
			return map[string]interface{}{
				"file_names": names,
				"matrices":   matrices,
			}
		}
	case "couples-files":
		if names, matrix, err := reader.GetFileCooccurrence(); err == nil {
			return map[string]interface{}{
				"file_names":      names,
				"coupling_matrix": matrix,
			}
		}
	case "couples-people":
		if names, matrix, err := reader.GetPeopleCooccurrence(); err == nil {
			return map[string]interface{}{
				"people_names":    names,
				"coupling_matrix": matrix,
			}
		}
	case "couples-shotness":
		if names, matrix, err := reader.GetShotnessCooccurrence(); err == nil {
			return map[string]interface{}{
				"entity_names":    names,
				"coupling_matrix": matrix,
			}
		}
	case "run-times":
		if stats, err := reader.GetRuntimeStats(); err == nil {
			return map[string]interface{}{
				"runtime_stats": stats,
			}
		}
	case "languages":
		if stats, err := reader.GetLanguageStats(); err == nil {
			return map[string]interface{}{
				"language_stats": stats,
			}
		}
	}

	return map[string]interface{}{
		"message": "No data available for JSON export",
	}
}

// saveJSONResults saves the analysis results as JSON
func saveJSONResults(results map[string]interface{}, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print

	// Add metadata
	output := map[string]interface{}{
		"meta": map[string]interface{}{
			"generated_by":   "labours-go",
			"generated_at":   time.Now().Format(time.RFC3339),
			"modes_executed": len(results),
		},
		"results": results,
	}

	return encoder.Encode(output)
}
