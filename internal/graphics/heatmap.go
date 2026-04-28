package graphics

import (
	"fmt"
	"image/color"
	"math"
	"sync"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	matcolor "matplotlib-go/color"
	"matplotlib-go/render"
)

var registerHeatmapColormapsOnce sync.Once

// RegisterPythonLaboursHeatmapColormaps registers colormaps used by Python labours
// but not provided by matplotlib-go's default registry.
func RegisterPythonLaboursHeatmapColormaps() {
	registerHeatmapColormapsOnce.Do(func() {
		matcolor.RegisterColormap("OrRd", matcolor.NewColormap("OrRd", []matcolor.ColorStop{
			{Pos: 0.000, Color: render.Color{R: 1.000, G: 0.969, B: 0.925, A: 1}},
			{Pos: 0.125, Color: render.Color{R: 0.996, G: 0.910, B: 0.784, A: 1}},
			{Pos: 0.250, Color: render.Color{R: 0.992, G: 0.831, B: 0.620, A: 1}},
			{Pos: 0.375, Color: render.Color{R: 0.992, G: 0.733, B: 0.518, A: 1}},
			{Pos: 0.500, Color: render.Color{R: 0.988, G: 0.553, B: 0.349, A: 1}},
			{Pos: 0.625, Color: render.Color{R: 0.937, G: 0.396, B: 0.282, A: 1}},
			{Pos: 0.750, Color: render.Color{R: 0.843, G: 0.188, B: 0.122, A: 1}},
			{Pos: 0.875, Color: render.Color{R: 0.702, G: 0.000, B: 0.000, A: 1}},
			{Pos: 1.000, Color: render.Color{R: 0.498, G: 0.000, B: 0.000, A: 1}},
		}))
	})
}

// CustomPalette represents a mapping of values to a predefined set of colors.
type CustomPalette struct {
	Colors []color.Color
	Min    float64
	Max    float64
}

// At maps a value to a corresponding color in the palette.
func (p *CustomPalette) At(value float64) color.Color {
	if len(p.Colors) == 0 {
		return color.Black
	}
	if p.Max == p.Min {
		return p.Colors[len(p.Colors)/2]
	}

	// Normalize the value to the range [0, 1].
	normalized := (value - p.Min) / (p.Max - p.Min)
	if math.IsNaN(normalized) {
		normalized = 0
	}
	if normalized < 0 {
		normalized = 0
	} else if normalized > 1 {
		normalized = 1
	}

	// Scale the normalized value to the palette size.
	index := int(math.Round(normalized * float64(len(p.Colors)-1)))
	return p.Colors[index]
}

// HeatMap represents a heatmap plotter for a 2D matrix.
type HeatMap struct {
	Matrix  [][]float64
	Rows    []string
	Cols    []string
	Palette *CustomPalette
}

// NewHeatMap creates a new HeatMap with a custom palette.
func NewHeatMap(matrix [][]float64, rows, cols []string, palette *CustomPalette) *HeatMap {
	return &HeatMap{
		Matrix:  matrix,
		Rows:    rows,
		Cols:    cols,
		Palette: palette,
	}
}

func ValidateHeatMap(matrix [][]float64, rows, cols []string) error {
	if len(matrix) != len(rows) {
		return fmt.Errorf("heatmap row count mismatch: matrix has %d rows, labels have %d", len(matrix), len(rows))
	}
	for i, row := range matrix {
		if len(row) != len(cols) {
			return fmt.Errorf("heatmap column count mismatch on row %d: matrix has %d columns, labels have %d", i, len(row), len(cols))
		}
	}
	return nil
}

// Plot draws the heatmap onto the plot canvas.
func (hm *HeatMap) Plot(c draw.Canvas, p *plot.Plot) {
	r := c.Rectangle.Size()
	cellWidth := r.X / vg.Length(len(hm.Cols))
	cellHeight := r.Y / vg.Length(len(hm.Rows))

	for rowIdx, row := range hm.Matrix {
		for colIdx, value := range row {
			x := vg.Length(colIdx) * cellWidth
			y := vg.Length(len(hm.Rows)-1-rowIdx) * cellHeight // Invert rows for correct orientation

			// Map value to a color using the custom palette.
			clr := hm.Palette.At(value)

			// Define the coordinates for the cell.
			xMin := c.Rectangle.Min.X + x
			xMax := xMin + cellWidth
			yMin := c.Rectangle.Min.Y + y
			yMax := yMin + cellHeight

			// Create a path for the rectangle.
			path := vg.Path{
				{Type: vg.MoveComp, Pos: vg.Point{X: xMin, Y: yMin}},
				{Type: vg.LineComp, Pos: vg.Point{X: xMax, Y: yMin}},
				{Type: vg.LineComp, Pos: vg.Point{X: xMax, Y: yMax}},
				{Type: vg.LineComp, Pos: vg.Point{X: xMin, Y: yMax}},
				{Type: vg.CloseComp},
			}

			// Set the fill color and fill the rectangle.
			c.SetColor(clr)
			c.Fill(path)
		}
	}
}

// DataRange returns the minimum and maximum data range of the heatmap.
func (hm *HeatMap) DataRange() (xmin, xmax, ymin, ymax float64) {
	xmin, ymin = -0.5, -0.5
	xmax = float64(len(hm.Cols)) - 0.5
	ymax = float64(len(hm.Rows)) - 0.5
	return
}
