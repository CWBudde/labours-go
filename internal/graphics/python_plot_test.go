package graphics

import (
	"image/color"
	"testing"
)

func TestGeneratePythonLaboursColorPaletteMatchesGgplotStackplot(t *testing.T) {
	colors := generatePythonLaboursColorPalette(2)
	if len(colors) != 2 {
		t.Fatalf("palette length = %d, want 2", len(colors))
	}

	want := []color.Color{
		color.RGBA{R: 226, G: 74, B: 51, A: 255},
		color.RGBA{R: 52, G: 138, B: 189, A: 255},
	}
	for i := range want {
		if colors[i] != want[i] {
			t.Fatalf("color %d = %#v, want %#v", i, colors[i], want[i])
		}
	}
}
