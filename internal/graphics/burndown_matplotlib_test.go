package graphics

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/burndown"
)

func TestGeneratePythonLaboursColorPaletteMatchesPythonTab20Cycle(t *testing.T) {
	colors := PythonLaboursColorPalette(2)
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

func TestPlotBurndownMatplotlibUsesBackends(t *testing.T) {
	oldQuiet := viper.GetBool("quiet")
	oldSize := viper.GetString("size")
	viper.Set("quiet", true)
	viper.Set("size", "2,1.5")
	defer func() {
		viper.Set("quiet", oldQuiet)
		viper.Set("size", oldSize)
	}()

	data := &burndown.ProcessedBurndown{
		Name: "repo",
		Matrix: [][]float64{
			{4, 3, 2},
			{0, 1, 2},
		},
		DateRange: []time.Time{
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		},
		Labels:       []string{"old", "new"},
		Granularity:  30,
		Sampling:     30,
		ResampleMode: "month",
	}

	dir := t.TempDir()
	pngPath := filepath.Join(dir, "burndown.png")
	if err := PlotBurndownMatplotlib(data, pngPath, false); err != nil {
		t.Fatalf("plot png: %v", err)
	}
	pngFile, err := os.Open(pngPath)
	if err != nil {
		t.Fatalf("open png: %v", err)
	}
	defer pngFile.Close()
	if _, err := png.Decode(pngFile); err != nil {
		t.Fatalf("decode png: %v", err)
	}

	svgPath := filepath.Join(dir, "burndown.svg")
	if err := PlotBurndownMatplotlib(data, svgPath, true); err != nil {
		t.Fatalf("plot svg: %v", err)
	}
	svgBytes, err := os.ReadFile(svgPath)
	if err != nil {
		t.Fatalf("read svg: %v", err)
	}
	if !strings.Contains(string(svgBytes), "<svg") {
		t.Fatalf("svg output does not contain <svg")
	}
}
