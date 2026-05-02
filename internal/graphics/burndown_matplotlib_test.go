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

func TestPythonLaboursColorPaletteMatchesGGPlotCycle(t *testing.T) {
	// Python labours runs `pyplot.style.use("ggplot")` before plotting, so the
	// palette we produce must match `axes.prop_cycle` from
	// `matplotlib/mpl-data/stylelib/ggplot.mplstyle`.
	colors := PythonLaboursColorPalette(7)
	if len(colors) != 7 {
		t.Fatalf("palette length = %d, want 7", len(colors))
	}

	want := []color.Color{
		color.RGBA{R: 0xE2, G: 0x4A, B: 0x33, A: 255},
		color.RGBA{R: 0x34, G: 0x8A, B: 0xBD, A: 255},
		color.RGBA{R: 0x98, G: 0x8E, B: 0xD5, A: 255},
		color.RGBA{R: 0x77, G: 0x77, B: 0x77, A: 255},
		color.RGBA{R: 0xFB, G: 0xC1, B: 0x5E, A: 255},
		color.RGBA{R: 0x8E, G: 0xBA, B: 0x42, A: 255},
		color.RGBA{R: 0xFF, G: 0xB5, B: 0xB8, A: 255},
	}
	for i := range want {
		if colors[i] != want[i] {
			t.Fatalf("color %d = %#v, want %#v", i, colors[i], want[i])
		}
	}

	// More requested series than palette entries cycles, matching matplotlib.
	wrapped := PythonLaboursColorPalette(8)
	if wrapped[7] != want[0] {
		t.Fatalf("wrap color = %#v, want %#v", wrapped[7], want[0])
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
	oldBackground := viper.GetString("background")
	viper.Set("quiet", true)
	viper.Set("size", "2,1.5")
	viper.Set("background", "white")
	defer func() {
		viper.Set("quiet", oldQuiet)
		viper.Set("size", oldSize)
		viper.Set("background", oldBackground)
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
	img, err := png.Decode(pngFile)
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	if _, _, _, alpha := img.At(0, 0).RGBA(); alpha != 0 {
		t.Fatalf("corner alpha = %d, want transparent", alpha)
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

func TestBurndownYAxisTicksUseScientificScale(t *testing.T) {
	ticks, labels, offset := burndownYAxisTicks(25800)
	if offset != "1e4" {
		t.Fatalf("offset = %q, want 1e4", offset)
	}
	wantTicks := []float64{0, 5000, 10000, 15000, 20000, 25000}
	if len(ticks) != len(wantTicks) {
		t.Fatalf("ticks = %v, want %v", ticks, wantTicks)
	}
	for i := range wantTicks {
		if ticks[i] != wantTicks[i] {
			t.Fatalf("ticks = %v, want %v", ticks, wantTicks)
		}
	}
	wantLabels := []string{"0", "0.5", "1", "1.5", "2", "2.5"}
	if strings.Join(labels, ",") != strings.Join(wantLabels, ",") {
		t.Fatalf("labels = %v, want %v", labels, wantLabels)
	}

	ticks, labels, offset = burndownYAxisTicks(6800)
	if offset != "1e3" {
		t.Fatalf("offset = %q, want 1e3", offset)
	}
	if got, want := ticks[len(ticks)-1], 7000.0; got != want {
		t.Fatalf("last tick = %v, want %v", got, want)
	}
	if got, want := labels[len(labels)-1], "7"; got != want {
		t.Fatalf("last label = %q, want %q", got, want)
	}
}

func TestBurndownDateTicksUseEndpointDatesForShortYearlySpan(t *testing.T) {
	dates := []time.Time{
		time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	_, labels := burndownDateTicks(dates, "year")
	want := []string{"2017-01-01", "2018-01-01"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Fatalf("labels = %v, want %v", labels, want)
	}
}
