package graphics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/viper"
	"gonum.org/v1/plot/vg"
)

// ChartType represents different types of charts with their default dimensions
type ChartType int

const (
	ChartTypeDefault ChartType = iota // Standard rectangular charts (burndown, devs, etc.)
	ChartTypeSquare                   // Square charts (heatmaps, coupling matrices)
	ChartTypeCompact                  // Compact charts (ownership, simple plots)
	ChartTypeWide                     // Wide charts (timeline-heavy charts)
)

// defaultSizes defines the default dimensions for each chart type
var defaultSizes = map[ChartType][2]float64{
	ChartTypeDefault: {16.0, 8.0},  // Python labours default (16, 12) adapted for Go's typical 16x8
	ChartTypeSquare:  {12.0, 12.0}, // Square for heatmaps and matrices
	ChartTypeCompact: {10.0, 6.0},  // Compact for simple charts
	ChartTypeWide:    {16.0, 10.0}, // Wide for timeline-heavy charts
}

// ParsePlotSize parses a size string in the format "width,height" and returns vg.Length values.
// If the size string is empty, returns the default size for the given chart type.
// The size string should be in inches, matching Python labours behavior.
func ParsePlotSize(sizeStr string, chartType ChartType) (width, height vg.Length, err error) {
	// If no size specified, use default for chart type
	if sizeStr == "" {
		defaultSize := defaultSizes[chartType]
		return vg.Length(defaultSize[0]) * vg.Inch, vg.Length(defaultSize[1]) * vg.Inch, nil
	}

	// Parse "width,height" format
	parts := strings.Split(strings.TrimSpace(sizeStr), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size format '%s': expected 'width,height' (e.g., '12,9')", sizeStr)
	}

	// Parse width
	widthFloat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width '%s': %w", parts[0], err)
	}

	// Parse height
	heightFloat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height '%s': %w", parts[1], err)
	}

	// Validate dimensions (reasonable bounds)
	if widthFloat <= 0 || heightFloat <= 0 {
		return 0, 0, fmt.Errorf("dimensions must be positive: got width=%.1f, height=%.1f", widthFloat, heightFloat)
	}
	if widthFloat > 50 || heightFloat > 50 {
		return 0, 0, fmt.Errorf("dimensions too large: got width=%.1f, height=%.1f (max 50 inches)", widthFloat, heightFloat)
	}

	return vg.Length(widthFloat) * vg.Inch, vg.Length(heightFloat) * vg.Inch, nil
}

// GetPlotSize returns the plot size based on the --size flag and chart type.
// This is the main function that modes should use to get dynamic plot dimensions.
func GetPlotSize(chartType ChartType) (width, height vg.Length) {
	sizeStr := viper.GetString("size")
	width, height, err := ParsePlotSize(sizeStr, chartType)
	if err != nil {
		// Log error and use default
		fmt.Printf("Warning: %v, using default size\n", err)
		defaultSize := defaultSizes[chartType]
		return vg.Length(defaultSize[0]) * vg.Inch, vg.Length(defaultSize[1]) * vg.Inch
	}
	return width, height
}

// GetPlotSizeInches returns the plot size in inches as floats, useful for debugging and logging
func GetPlotSizeInches(chartType ChartType) (width, height float64) {
	w, h := GetPlotSize(chartType)
	return float64(w / vg.Inch), float64(h / vg.Inch)
}

// GetPythonPlotSize returns dimensions that match matplotlib's default PNG
// export convention: figure size in inches rendered at 100 DPI.
func GetPythonPlotSize(defaultWidth, defaultHeight float64) (width, height vg.Length) {
	sizeStr := viper.GetString("size")
	if sizeStr == "" {
		return pythonInchesToVG(defaultWidth), pythonInchesToVG(defaultHeight)
	}

	parsedWidth, parsedHeight, err := parsePlotSizeFloats(sizeStr)
	if err != nil {
		fmt.Printf("Warning: %v, using default size\n", err)
		return pythonInchesToVG(defaultWidth), pythonInchesToVG(defaultHeight)
	}
	return pythonInchesToVG(parsedWidth), pythonInchesToVG(parsedHeight)
}

func pythonInchesToVG(inches float64) vg.Length {
	return vg.Length(inches * 100 * 72 / 96)
}

func parsePlotSizeFloats(sizeStr string) (width, height float64, err error) {
	parts := strings.Split(strings.TrimSpace(sizeStr), ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size format '%s': expected 'width,height' (e.g., '12,9')", sizeStr)
	}

	width, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width '%s': %w", parts[0], err)
	}

	height, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height '%s': %w", parts[1], err)
	}

	if width <= 0 || height <= 0 {
		return 0, 0, fmt.Errorf("dimensions must be positive: got width=%.1f, height=%.1f", width, height)
	}
	if width > 50 || height > 50 {
		return 0, 0, fmt.Errorf("dimensions too large: got width=%.1f, height=%.1f (max 50 inches)", width, height)
	}
	return width, height, nil
}
