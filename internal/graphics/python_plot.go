package graphics

import (
	"fmt"
	"image/color"

	"github.com/spf13/viper"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"labours-go/internal/burndown"
)

// PlotBurndownPythonStyle creates a burndown plot that matches Python's pyplot.stackplot behavior
func PlotBurndownPythonStyle(data *burndown.ProcessedBurndown, output string, relative bool) error {
	if data == nil || len(data.Matrix) == 0 || len(data.DateRange) == 0 {
		return fmt.Errorf("empty burndown data")
	}

	p := plot.New()
	// Generate Python-compatible title: "repository 2 x 225 (granularity 30, sampling 30)"
	p.Title.Text = fmt.Sprintf("%s %d x %d (granularity %d, sampling %d)",
		data.Name, len(data.Matrix), len(data.DateRange), data.Granularity, data.Sampling)
	p.X.Label.Text = "Time"
	p.Y.Label.Text = "Lines of code"

	// Apply theme styling
	applyThemeToPlot(p)
	p.BackgroundColor = color.RGBA{R: 255, G: 255, B: 255, A: 0}

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

	// Create cumulative data for stacking (bottom to top like Python's stackplot)
	cumulative := make([][]float64, numSeries)
	for i := range cumulative {
		cumulative[i] = make([]float64, numPoints)
		for j := 0; j < numPoints && j < len(matrix[i]); j++ {
			cumulative[i][j] = matrix[i][j]
			if i > 0 {
				cumulative[i][j] += cumulative[i-1][j]
			}
		}
	}

	// Create stacked areas in data order so the legend matches matplotlib.
	for i := 0; i < numSeries; i++ {
		// Create data points for this layer
		var topPoints plotter.XYs
		var bottomPoints plotter.XYs

		for j := 0; j < numPoints; j++ {
			x := timeValues[j]
			topY := cumulative[i][j]

			var bottomY float64
			if i > 0 {
				bottomY = cumulative[i-1][j]
			} else {
				bottomY = 0
			}

			topPoints = append(topPoints, plotter.XY{X: x, Y: topY})
			bottomPoints = append(bottomPoints, plotter.XY{X: x, Y: bottomY})
		}

		// Use semantic label from Python processing
		label := fmt.Sprintf("Layer %d", i)
		if i < len(data.Labels) {
			label = data.Labels[i]
		}

		// Create polygon for this stacked area
		if err := addStackedLayer(p, topPoints, bottomPoints, colors[i], label); err != nil {
			return fmt.Errorf("error adding layer %s: %v", label, err)
		}
	}

	// Configure time axis with Python-style formatting
	configureBurndownTimeAxis(p, timeValues, data.ResampleMode)

	// Set Y-axis limits
	if relative {
		p.Y.Min = 0
		p.Y.Max = 1
	}

	// Configure legend position (matches Python behavior)
	p.Legend.Left = true
	p.Legend.Top = !relative

	width, height := GetPythonPlotSize(16, 12)
	if err := SavePNGWithBackground(p, width, height, output, color.Transparent); err != nil {
		return err
	}

	return nil
}

// configureBurndownTimeAxis sets up the time axis to match Python's matplotlib behavior
func configureBurndownTimeAxis(p *plot.Plot, timeValues []float64, resampleMode string) {
	if len(timeValues) == 0 {
		return
	}

	// Set basic time range
	p.X.Min = timeValues[0]
	p.X.Max = timeValues[len(timeValues)-1]

	// Configure time ticker based on resampling mode
	var format string
	switch resampleMode {
	case "A", "year":
		format = "2006"
		p.X.Tick.Marker = &TimeTicker{Format: format}
	case "M", "month":
		format = "2006-01"
		p.X.Tick.Marker = &TimeTicker{Format: format}
	case "D", "day":
		format = "2006-01-02"
		p.X.Tick.Marker = &TimeTicker{Format: format}
	default:
		format = "2006-01-02"
		p.X.Tick.Marker = &TimeTicker{Format: format}
	}
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
