package graphics

import (
	"image/color"
	"testing"
)

func TestCustomPaletteAtHandlesFlatRange(t *testing.T) {
	palette := &CustomPalette{
		Colors: []color.Color{
			color.RGBA{R: 1, A: 255},
			color.RGBA{R: 2, A: 255},
			color.RGBA{R: 3, A: 255},
		},
		Min: 5,
		Max: 5,
	}

	if got := palette.At(5); got != palette.Colors[1] {
		t.Fatalf("At() = %#v, want middle color %#v", got, palette.Colors[1])
	}
}

func TestHeatMapDataRangeCentersIntegerTicks(t *testing.T) {
	heatmap := NewHeatMap(
		[][]float64{{1, 2, 3}, {4, 5, 6}},
		[]string{"row 0", "row 1"},
		[]string{"col 0", "col 1", "col 2"},
		nil,
	)

	xmin, xmax, ymin, ymax := heatmap.DataRange()

	if xmin != -0.5 || xmax != 2.5 || ymin != -0.5 || ymax != 1.5 {
		t.Fatalf("DataRange() = (%v, %v, %v, %v), want (-0.5, 2.5, -0.5, 1.5)", xmin, xmax, ymin, ymax)
	}
}
