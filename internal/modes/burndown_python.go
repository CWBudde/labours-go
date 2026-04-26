package modes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"labours-go/internal/burndown"
	"labours-go/internal/graphics"
	"labours-go/internal/progress"
	"labours-go/internal/readers"
)

// GenerateBurndownProjectPython creates a Python-compatible burndown chart
func GenerateBurndownProjectPython(reader readers.Reader, output string, relative bool, resample string) error {
	fmt.Println("Running: burndown-project (Python-compatible)")

	// Initialize progress tracking
	quiet := viper.GetBool("quiet")
	progEstimator := progress.NewProgressEstimator(!quiet)

	totalPhases := 4 // validation, data loading, processing, plotting
	progEstimator.StartMultiOperation(totalPhases, "Python-Compatible Burndown Analysis")

	// Phase 1: Validation and setup
	progEstimator.NextOperation("Validating output path")
	if output == "" {
		output = "burndown_project_python.png"
		if !quiet {
			fmt.Printf("Output not provided, using default: %s\n", output)
		}
	}

	outputDir := filepath.Dir(output)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to create output directory %s: %v", outputDir, err)
	}

	// Phase 2: Load burndown data with Python-compatible header
	progEstimator.NextOperation("Loading burndown data")
	header, name, matrix, err := reader.GetProjectBurndownWithHeader()
	if err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to load burndown data: %v", err)
	}

	if !quiet {
		fmt.Printf("Processing %s with %d age bands and %d time points\n", name, len(matrix), len(matrix[0]))
		fmt.Printf("Header: start=%d, last=%d, sampling=%d, granularity=%d, tick_size=%.3f\n",
			header.Start, header.Last, header.Sampling, header.Granularity, header.TickSize)
	}

	// Phase 3: Process data using Python-compatible algorithms
	progEstimator.NextOperation("Processing data with Python algorithms")
	if resample == "" {
		resample = "year" // Default to yearly like Python
	}

	processedData, err := burndown.LoadBurndown(header, name, matrix, resample, true, true)
	if err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to process burndown data: %v", err)
	}

	if !quiet {
		fmt.Printf("Processed into %d layers: %v\n", len(processedData.Labels), processedData.Labels)
		fmt.Printf("Final matrix dimensions: %dx%d\n", len(processedData.Matrix), len(processedData.Matrix[0]))
	}

	// Print survival analysis (like Python does)
	if !quiet {
		graphics.PrintSurvivalFunction(processedData.Matrix)
	}

	// Phase 4: Generate visualization
	progEstimator.NextOperation("Generating Python-style visualization")
	if err := graphics.PlotBurndownPythonStyle(processedData, output, relative); err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("error creating Python-style burndown plot: %v", err)
	}

	progEstimator.FinishMultiOperation()
	if !quiet {
		fmt.Printf("Python-compatible chart saved to %s\n", output)
	}
	return nil
}

// GenerateBurndownFilePython creates Python-compatible file-level burndown charts
func GenerateBurndownFilePython(reader readers.Reader, output string, relative bool, resample string) error {
	fmt.Println("Running: burndown-file (Python-compatible)")

	// Get files burndown data
	files, err := reader.GetFilesBurndown()
	if err != nil {
		return fmt.Errorf("failed to get files burndown data: %v", err)
	}

	// Get header information
	header, _, _, err := reader.GetProjectBurndownWithHeader()
	if err != nil {
		return fmt.Errorf("failed to get burndown header: %v", err)
	}

	quiet := viper.GetBool("quiet")
	if !quiet {
		fmt.Printf("Processing %d files\n", len(files))
	}

	// Process each file
	for i, file := range files {
		if !quiet {
			fmt.Printf("Processing file %d/%d: %s\n", i+1, len(files), file.Filename)
		}

		if resample == "" {
			resample = "year"
		}

		processedData, err := burndown.LoadBurndown(header, file.Filename, file.Matrix, resample, false, false)
		if err != nil {
			if !quiet {
				fmt.Printf("Warning: failed to process %s: %v\n", file.Filename, err)
			}
			continue
		}

		// Generate output filename
		fileOutput := output
		if output == "" {
			fileOutput = fmt.Sprintf("burndown_file_%s.png", sanitizeFilename(file.Filename))
		} else {
			dir := filepath.Dir(output)
			ext := filepath.Ext(output)
			base := filepath.Base(output)
			base = base[:len(base)-len(ext)]
			fileOutput = filepath.Join(dir, fmt.Sprintf("%s_%s%s", base, sanitizeFilename(file.Filename), ext))
		}

		if err := graphics.PlotBurndownPythonStyle(processedData, fileOutput, relative); err != nil {
			if !quiet {
				fmt.Printf("Warning: failed to create plot for %s: %v\n", file.Filename, err)
			}
			continue
		}

		if !quiet {
			fmt.Printf("Chart saved: %s\n", fileOutput)
		}
	}

	return nil
}

// GenerateBurndownRepositoryPython creates Python-compatible repository-level burndown charts.
func GenerateBurndownRepositoryPython(reader readers.Reader, output string, relative bool, resample string) error {
	fmt.Println("Running: burndown-repository (Python-compatible)")

	repoReader, ok := reader.(readers.RepositoryBurndownReader)
	if !ok {
		return fmt.Errorf("reader does not expose repository burndown data")
	}

	repositories, err := repoReader.GetRepositoriesBurndown()
	if err != nil {
		return fmt.Errorf("failed to get repositories burndown data: %v", err)
	}
	if len(repositories) == 0 {
		return fmt.Errorf("no repository burndown data found")
	}

	header, _, _, err := reader.GetProjectBurndownWithHeader()
	if err != nil {
		return fmt.Errorf("failed to get burndown header: %v", err)
	}

	if output == "" {
		output = "."
	}
	if err := os.MkdirAll(output, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory %s: %v", output, err)
	}
	if resample == "" {
		resample = "year"
	}

	quiet := viper.GetBool("quiet")
	for i, repository := range repositories {
		if !quiet {
			fmt.Printf("Processing repository %d/%d: %s\n", i+1, len(repositories), repository.Repository)
		}

		processedData, err := burndown.LoadBurndown(header, repository.Repository, repository.Matrix, resample, false, false)
		if err != nil {
			if !quiet {
				fmt.Printf("Warning: failed to process repository %s: %v\n", repository.Repository, err)
			}
			continue
		}

		repoBase := filepath.Join(output, fmt.Sprintf("burndown-repository_%s", sanitizeFilename(repository.Repository)))
		repoPNG := repoBase + ".png"
		if err := graphics.PlotBurndownPythonStyle(processedData, repoPNG, relative); err != nil {
			if !quiet {
				fmt.Printf("Warning: failed to create plot for repository %s: %v\n", repository.Repository, err)
			}
			continue
		}
		repoSVG := repoBase + ".svg"
		if err := graphics.PlotBurndownPythonStyle(processedData, repoSVG, relative); err != nil {
			if !quiet {
				fmt.Printf("Warning: failed to create SVG plot for repository %s: %v\n", repository.Repository, err)
			}
			continue
		}
		if !quiet {
			fmt.Printf("Charts saved: %s and %s\n", repoPNG, repoSVG)
		}
	}

	return nil
}

// GenerateBurndownReposCombinedPython creates one burndown chart from all repository matrices combined.
func GenerateBurndownReposCombinedPython(reader readers.Reader, output string, relative bool, resample string) error {
	fmt.Println("Running: burndown-repos-combined (Python-compatible)")

	repoReader, ok := reader.(readers.RepositoryBurndownReader)
	if !ok {
		return fmt.Errorf("reader does not expose repository burndown data")
	}

	repositories, err := repoReader.GetRepositoriesBurndown()
	if err != nil {
		return fmt.Errorf("failed to get repositories burndown data: %v", err)
	}
	if len(repositories) == 0 {
		return fmt.Errorf("no repository burndown data found")
	}

	header, _, _, err := reader.GetProjectBurndownWithHeader()
	if err != nil {
		return fmt.Errorf("failed to get burndown header: %v", err)
	}

	matrix := combineRepositoryBurndowns(repositories)
	if len(matrix) == 0 || len(matrix[0]) == 0 {
		return fmt.Errorf("empty combined repository burndown matrix")
	}
	if resample == "" {
		resample = "year"
	}
	if output == "" {
		output = "burndown-repos-combined.png"
	}
	if err := os.MkdirAll(filepath.Dir(output), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory %s: %v", filepath.Dir(output), err)
	}

	processedData, err := burndown.LoadBurndown(header, "repositories", matrix, resample, true, false)
	if err != nil {
		return fmt.Errorf("failed to process combined repository burndown data: %v", err)
	}
	if err := graphics.PlotBurndownPythonStyle(processedData, output, relative); err != nil {
		return fmt.Errorf("error creating combined repository burndown plot: %v", err)
	}

	if !viper.GetBool("quiet") {
		fmt.Printf("Combined repository burndown chart saved to %s\n", output)
	}
	return nil
}

func combineRepositoryBurndowns(repositories []readers.RepositoryBurndown) [][]int {
	rows, cols := 0, 0
	for _, repository := range repositories {
		if len(repository.Matrix) > rows {
			rows = len(repository.Matrix)
		}
		for _, row := range repository.Matrix {
			if len(row) > cols {
				cols = len(row)
			}
		}
	}
	if rows == 0 || cols == 0 {
		return nil
	}

	combined := make([][]int, rows)
	for i := range combined {
		combined[i] = make([]int, cols)
	}
	for _, repository := range repositories {
		for i, row := range repository.Matrix {
			for j, value := range row {
				combined[i][j] += value
			}
		}
	}
	return combined
}

// sanitizeFilename removes problematic characters from filenames
func sanitizeFilename(filename string) string {
	// Simple sanitization - replace path separators and problematic characters
	result := ""
	for _, r := range filename {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			result += "_"
		case '.':
			result += "_"
		default:
			result += string(r)
		}
	}
	if len(result) > 50 {
		result = result[:50]
	}
	return result
}
