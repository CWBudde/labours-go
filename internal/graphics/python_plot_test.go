package graphics

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

func TestGeneratePythonLaboursColorPaletteMatchesPythonTab20Cycle(t *testing.T) {
	colors := generatePythonLaboursColorPalette(2)
	if len(colors) != 2 {
		t.Fatalf("palette length = %d, want 2", len(colors))
	}

	want := []color.Color{
		color.RGBA{R: 31, G: 119, B: 180, A: 255},
		color.RGBA{R: 174, G: 199, B: 232, A: 255},
	}
	for i := range want {
		if colors[i] != want[i] {
			t.Fatalf("color %d = %#v, want %#v", i, colors[i], want[i])
		}
	}
}

func TestSavePNGWithBackgroundPreservesTransparency(t *testing.T) {
	p := plot.New()
	output := filepath.Join(t.TempDir(), "transparent.png")

	if err := SavePNGWithBackground(p, 2*vg.Inch, 2*vg.Inch, output, color.Transparent); err != nil {
		t.Fatalf("save transparent png: %v", err)
	}

	file, err := os.Open(output)
	if err != nil {
		t.Fatalf("open transparent png: %v", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode transparent png: %v", err)
	}
	_, _, _, a := img.At(0, 0).RGBA()
	if a != 0 {
		t.Fatalf("corner alpha = %d, want 0", a)
	}
}
