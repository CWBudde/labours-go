package modes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/readers"
)

// Languages generates language statistics and visualization showing the distribution
// of programming languages used in the repository.
func Languages(reader readers.Reader, output string) error {
	// Step 1: Read language statistics
	languageStats, err := reader.GetLanguageStats()
	if err != nil {
		return fmt.Errorf("failed to get language stats: %v - ensure the input data contains language statistics", err)
	}

	if len(languageStats) == 0 {
		return fmt.Errorf("no language statistics found in the data - the input file may not contain language analysis results")
	}

	// Step 2: Sort languages by line count (descending)
	sort.Slice(languageStats, func(i, j int) bool {
		return languageStats[i].Lines > languageStats[j].Lines
	})

	// Step 3: Generate visualization
	if err := plotLanguages(languageStats, output); err != nil {
		return fmt.Errorf("failed to generate language plot: %v", err)
	}

	fmt.Printf("Language analysis completed. Found %d languages.\n", len(languageStats))
	return nil
}

// plotLanguages creates a bar chart showing language distribution by lines of code
func plotLanguages(languageStats []readers.LanguageStat, output string) error {
	// Create a new plot
	p := plot.New()
	p.Title.Text = "Programming Languages by Lines of Code"
	p.X.Label.Text = "Languages"
	p.Y.Label.Text = "Lines of Code"

	// Prepare data for the bar chart
	names := make([]string, len(languageStats))
	values := make(plotter.Values, len(languageStats))

	for i, stat := range languageStats {
		names[i] = stat.Language
		values[i] = float64(stat.Lines)
	}

	// Create bar chart
	bars, err := plotter.NewBarChart(values, vg.Points(50))
	if err != nil {
		return fmt.Errorf("failed to create bar chart: %v", err)
	}

	// Style the bars with different colors
	for i := range bars.Values {
		bars.Color = graphics.ColorPalette[i%len(graphics.ColorPalette)]
	}

	p.Add(bars)

	// Create custom labels for X axis
	p.NominalX(names...)

	// Rotate x-axis labels if there are many languages
	if len(languageStats) > 10 {
		p.X.Tick.Label.Rotation = 0.785398 // 45 degrees in radians
		p.X.Tick.Label.XAlign = -0.5
		p.X.Tick.Label.YAlign = -0.5
	}

	// Save the plot with dynamic sizing
	width, height := graphics.GetPlotSize(graphics.ChartTypeDefault)
	outputs, err := languageOutputPaths(output)
	if err != nil {
		return err
	}

	for _, outputPath := range outputs {
		if err := graphics.SavePlotWithFormat(p, width, height, outputPath); err != nil {
			return err
		}
		fmt.Printf("Language chart saved to %s\n", outputPath)
	}

	// Print text summary
	fmt.Println("\nLanguage Statistics:")
	fmt.Println("====================")
	totalLines := 0
	for _, stat := range languageStats {
		totalLines += stat.Lines
	}

	for i, stat := range languageStats {
		percentage := float64(stat.Lines) / float64(totalLines) * 100
		fmt.Printf("%2d. %-15s %8d lines (%5.1f%%)\n", i+1, stat.Language, stat.Lines, percentage)
	}

	fmt.Printf("\nTotal: %d lines across %d languages\n", totalLines, len(languageStats))

	return nil
}

func languageOutputPaths(output string) ([]string, error) {
	if output == "" {
		output = "."
	}

	ext := strings.ToLower(filepath.Ext(output))
	if ext != "" {
		if dir := filepath.Dir(output); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create output directory %s: %v", dir, err)
			}
		}
		return []string{output}, nil
	}

	if err := os.MkdirAll(output, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %v", output, err)
	}
	return []string{
		filepath.Join(output, "languages.png"),
		filepath.Join(output, "languages.svg"),
	}, nil
}
