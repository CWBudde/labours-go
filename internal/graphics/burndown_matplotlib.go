package graphics

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"labours-go/internal/burndown"
	"matplotlib-go/backends"
	_ "matplotlib-go/backends/agg"
	_ "matplotlib-go/backends/svg"
	"matplotlib-go/core"
	"matplotlib-go/render"
	"matplotlib-go/style"
)

// PlotBurndownMatplotlib creates a burndown plot with matplotlib-go stackplot rendering.
func PlotBurndownMatplotlib(data *burndown.ProcessedBurndown, output string, relative bool) error {
	if data == nil || len(data.Matrix) == 0 || len(data.DateRange) == 0 {
		return fmt.Errorf("empty burndown data")
	}

	numSeries := len(data.Matrix)
	numPoints := len(data.DateRange)

	// Ensure matrix dimensions are consistent
	if numSeries == 0 {
		return fmt.Errorf("empty matrix")
	}

	// Convert dates to float64 for plotting (Unix timestamps)
	timeValues := make([]float64, numPoints)
	for i, date := range data.DateRange {
		timeValues[i] = float64(date.Unix())
	}

	// Python mutates the matrix after stackplot(), so the rendered reference
	// remains unnormalized and is only clipped by ylim(0, 1).
	matrix := data.Matrix

	if !viper.GetBool("quiet") {
		fmt.Printf("DEBUG MATRIX ANALYSIS:\n")
		fmt.Printf("  Matrix dimensions: %dx%d\n", len(matrix), len(matrix[0]))
		for i := 0; i < len(matrix); i++ {
			minVal, maxVal := matrix[i][0], matrix[i][0]
			negCount, posCount := 0, 0
			for j := 0; j < len(matrix[i]); j++ {
				if matrix[i][j] < minVal {
					minVal = matrix[i][j]
				}
				if matrix[i][j] > maxVal {
					maxVal = matrix[i][j]
				}
				if matrix[i][j] < 0 {
					negCount++
				}
				if matrix[i][j] > 0 {
					posCount++
				}
			}
			fmt.Printf("  Layer %d: min=%.2f, max=%.2f, negatives=%d, positives=%d\n", i, minVal, maxVal, negCount, posCount)
		}
	}

	colors := PythonLaboursColorPalette(numSeries)
	renderColors := make([]render.Color, len(colors))
	for i, c := range colors {
		renderColors[i] = renderColor(c)
	}
	labels := make([]string, numSeries)
	for i := range labels {
		labels[i] = fmt.Sprintf("Layer %d", i)
		if i < len(data.Labels) {
			labels[i] = data.Labels[i]
		}
	}

	width, height := pythonPlotPixelSize(16, 12)
	fig := core.NewFigure(
		width,
		height,
		style.WithTheme(style.ThemeGGPlot),
		style.WithFont("DejaVu Sans", 12),
	)
	ax := fig.AddSubplot(1, 1, 1)
	if ax == nil {
		return fmt.Errorf("failed to create burndown axes")
	}
	ax.SetTitle(fmt.Sprintf("%s %d x %d (granularity %d, sampling %d)",
		data.Name, len(data.Matrix), len(data.DateRange), data.Granularity, data.Sampling))
	ax.SetXLabel("Time")
	ax.SetYLabel("Lines of code")
	ax.SetXLim(timeValues[0], timeValues[len(timeValues)-1])
	if relative {
		ax.SetYLim(0, 1)
	}
	configureMatplotlibBurndownTimeAxis(ax, data.DateRange, data.ResampleMode)
	ax.AddXGrid()
	ax.AddYGrid()
	ax.StackPlot(timeValues, matrix, core.StackPlotOptions{
		Colors: renderColors,
		Labels: labels,
	})

	legend := ax.AddLegend()
	legend.Location = core.LegendUpperLeft
	if relative {
		legend.Location = core.LegendLowerLeft
	}

	return saveMatplotlibFigure(fig, output, width, height)
}

func configureMatplotlibBurndownTimeAxis(ax *core.Axes, dates []time.Time, resampleMode string) {
	if ax == nil || len(dates) == 0 {
		return
	}

	ticks, labels := burndownDateTicks(dates, resampleMode)
	if len(ticks) == 0 {
		return
	}
	ax.XAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.XAxis.Formatter = core.FixedFormatter{Labels: labels}
	if len(labels) > 6 {
		ax.XAxis.MajorLabelStyle = core.TickLabelStyle{Rotation: 30, AutoAlign: true}
	}
}

func burndownDateTicks(dates []time.Time, resampleMode string) ([]float64, []string) {
	if len(dates) == 0 {
		return nil, nil
	}
	start := dates[0]
	end := dates[len(dates)-1]
	if end.Before(start) {
		start, end = end, start
	}

	mode := strings.ToLower(resampleMode)
	switch {
	case strings.Contains(mode, "m"):
		return buildMonthlyTicks(start, end)
	case strings.Contains(mode, "a"), strings.Contains(mode, "y"), strings.Contains(mode, "year"):
		return buildYearlyTicks(start, end)
	}

	days := end.Sub(start).Hours() / 24
	switch {
	case days <= 45:
		return buildDailyTicks(start, end)
	case days <= 730:
		return buildMonthlyTicks(start, end)
	default:
		return buildYearlyTicks(start, end)
	}
}

func buildDailyTicks(start, end time.Time) ([]float64, []string) {
	days := int(math.Ceil(end.Sub(start).Hours() / 24))
	step := max(1, int(math.Ceil(float64(max(1, days))/10)))
	ticks := []time.Time{start}
	for t := start.Truncate(24 * time.Hour); !t.After(end); t = t.AddDate(0, 0, step) {
		if !t.Before(start) {
			ticks = append(ticks, t)
		}
	}
	ticks = append(ticks, end)
	return formatDateTicks(ticks, "2006-01-02")
}

func buildMonthlyTicks(start, end time.Time) ([]float64, []string) {
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	step := max(1, int(math.Ceil(float64(max(1, months))/10)))
	ticks := []time.Time{start}
	for t := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location()); !t.After(end); t = t.AddDate(0, step, 0) {
		if !t.Before(start) {
			ticks = append(ticks, t)
		}
	}
	ticks = append(ticks, end)
	return formatDateTicks(ticks, "2006-01")
}

func buildYearlyTicks(start, end time.Time) ([]float64, []string) {
	years := max(1, end.Year()-start.Year()+1)
	step := max(1, int(math.Ceil(float64(years)/10)))
	ticks := []time.Time{start}
	for year := start.Year(); year <= end.Year(); year += step {
		t := time.Date(year, 1, 1, 0, 0, 0, 0, start.Location())
		if !t.Before(start) && !t.After(end) {
			ticks = append(ticks, t)
		}
	}
	ticks = append(ticks, end)
	return formatDateTicks(ticks, "2006")
}

func formatDateTicks(dates []time.Time, layout string) ([]float64, []string) {
	seen := map[int64]bool{}
	ticks := make([]float64, 0, len(dates))
	labels := make([]string, 0, len(dates))
	for _, date := range dates {
		unix := date.Unix()
		if seen[unix] {
			continue
		}
		seen[unix] = true
		ticks = append(ticks, float64(unix))
		labels = append(labels, date.Format(layout))
	}
	return ticks, labels
}

func saveMatplotlibFigure(fig *core.Figure, output string, width, height int) error {
	if output == "" {
		output = "burndown_project_python.png"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("failed to create output directory for %s: %v", output, err)
	}

	background := render.Color{R: 1, G: 1, B: 1, A: 0}
	config := backends.Config{Width: width, Height: height, Background: background, DPI: 100}
	switch strings.ToLower(filepath.Ext(output)) {
	case ".svg":
		renderer, _, err := backends.NewRenderer("svg", config, nil)
		if err != nil {
			return fmt.Errorf("failed to create SVG renderer: %v", err)
		}
		return core.SaveSVG(fig, renderer, output)
	default:
		renderer, _, err := backends.NewRenderer("agg", config, backends.TextCapabilities)
		if err != nil {
			return fmt.Errorf("failed to create AGG renderer: %v", err)
		}
		return core.SavePNG(fig, renderer, output)
	}
}

func renderColor(c color.Color) render.Color {
	r, g, b, a := c.RGBA()
	return render.Color{
		R: float64(r) / 0xffff,
		G: float64(g) / 0xffff,
		B: float64(b) / 0xffff,
		A: float64(a) / 0xffff,
	}
}

func pythonPlotPixelSize(defaultWidth, defaultHeight float64) (int, int) {
	width := defaultWidth
	height := defaultHeight
	if sizeStr := viper.GetString("size"); sizeStr != "" {
		parsedWidth, parsedHeight, err := parsePlotSizeFloats(sizeStr)
		if err == nil {
			width, height = parsedWidth, parsedHeight
		} else {
			fmt.Printf("Warning: %v, using default size\n", err)
		}
	}
	return max(1, int(math.Round(width*100))), max(1, int(math.Round(height*100)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// PrintSurvivalFunction prints survival ratios to match Python output (placeholder)
func PrintSurvivalFunction(matrix [][]float64) {
	fmt.Println("           Ratio of survived lines")
	// TODO: Implement Kaplan-Meier survival analysis like Python
	// For now, just print a placeholder that shows we're processing survival data

	if len(matrix) > 0 && len(matrix[0]) > 0 {
		total := 0.0
		for i := range matrix {
			for j := range matrix[i] {
				total += matrix[i][j]
			}
		}

		for i := 0; i < len(matrix[0]); i++ {
			alive := 0.0
			for j := range matrix {
				if i < len(matrix[j]) {
					alive += matrix[j][i]
				}
			}
			if total > 0 {
				ratio := alive / total
				fmt.Printf("%d days\t\t%.6f\n", i, ratio)
			}
		}
	}
}

// generatePythonLaboursColorPalette matches Python labours' tab20 color cycle.
func PythonLaboursColorPalette(n int) []color.Color {
	tab20Colors := []color.Color{
		color.RGBA{R: 31, G: 119, B: 180, A: 255},
		color.RGBA{R: 174, G: 199, B: 232, A: 255},
		color.RGBA{R: 255, G: 127, B: 14, A: 255},
		color.RGBA{R: 255, G: 187, B: 120, A: 255},
		color.RGBA{R: 44, G: 160, B: 44, A: 255},
		color.RGBA{R: 152, G: 223, B: 138, A: 255},
		color.RGBA{R: 214, G: 39, B: 40, A: 255},
		color.RGBA{R: 255, G: 152, B: 150, A: 255},
		color.RGBA{R: 148, G: 103, B: 189, A: 255},
		color.RGBA{R: 197, G: 176, B: 213, A: 255},
		color.RGBA{R: 140, G: 86, B: 75, A: 255},
		color.RGBA{R: 196, G: 156, B: 148, A: 255},
		color.RGBA{R: 227, G: 119, B: 194, A: 255},
		color.RGBA{R: 247, G: 182, B: 210, A: 255},
		color.RGBA{R: 127, G: 127, B: 127, A: 255},
		color.RGBA{R: 199, G: 199, B: 199, A: 255},
		color.RGBA{R: 188, G: 189, B: 34, A: 255},
		color.RGBA{R: 219, G: 219, B: 141, A: 255},
		color.RGBA{R: 23, G: 190, B: 207, A: 255},
		color.RGBA{R: 158, G: 218, B: 229, A: 255},
	}

	colors := make([]color.Color, n)
	for i := 0; i < n; i++ {
		if i < len(tab20Colors) {
			colors[i] = tab20Colors[i]
		} else {
			colors[i] = generateHSVColorWithOpacity(i, n, 255)
		}
	}

	return colors
}
