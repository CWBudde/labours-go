package modes

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"gonum.org/v1/plot"
)

func TestProcessOverwritesMatrixUsesPythonColumnSemantics(t *testing.T) {
	people := []string{"low", "high"}
	matrix := [][]int{
		{10, -2, -5, -3},
		{20, -4, -1, -10},
	}

	rows, cols, normalized := processOverwritesMatrix(people, matrix, 1, true)
	if len(rows) != 1 || rows[0] != "high" {
		t.Fatalf("rows = %#v, want only high", rows)
	}
	wantCols := []string{"Unidentified", "high"}
	if len(cols) != len(wantCols) {
		t.Fatalf("cols = %#v, want %#v", cols, wantCols)
	}
	for i := range wantCols {
		if cols[i] != wantCols[i] {
			t.Fatalf("cols = %#v, want %#v", cols, wantCols)
		}
	}
	if len(normalized) != 1 || len(normalized[0]) != 2 {
		t.Fatalf("normalized shape = %#v, want 1x2", normalized)
	}
	if normalized[0][0] != 0.2 || normalized[0][1] != 0.5 {
		t.Fatalf("normalized = %#v, want [[0.2 0.5]]", normalized)
	}
}

func TestPlotOverwritesMatrixWritesOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "overwrites.png")
	err := plotOverwritesMatrix(
		[]string{"Alice", "Bob"},
		[]string{"Unidentified", "Alice", "Bob"},
		[][]float64{{-0.1, -0.2, -0.3}, {-0.4, -0.5, -0.6}},
		output,
	)
	if err != nil {
		t.Fatalf("plotOverwritesMatrix() error = %v", err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("expected output file: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty output file")
	}
}

func TestMatrixRangeUsesActualProcessedValues(t *testing.T) {
	minValue, maxValue := matrixRange([][]float64{
		{-0.2, 0.5},
		{0.1, -0.7},
	})

	if minValue != -0.7 || maxValue != 0.5 {
		t.Fatalf("matrixRange() = (%v, %v), want (-0.7, 0.5)", minValue, maxValue)
	}
}

func TestTopTickLabelsReserveTopSpace(t *testing.T) {
	labels := newTopTickLabels([]string{"Unidentified", "Alice"}, plot.New().X.Tick.Label)
	boxes := labels.GlyphBoxes(plot.New())

	if len(boxes) != 2 {
		t.Fatalf("GlyphBoxes() returned %d boxes, want 2", len(boxes))
	}
	for i, box := range boxes {
		if box.Y != 1 {
			t.Fatalf("box %d Y = %v, want 1", i, box.Y)
		}
		if box.Rectangle.Size().Y <= 0 || math.IsNaN(float64(box.Rectangle.Size().Y)) {
			t.Fatalf("box %d height = %v, want positive", i, box.Rectangle.Size().Y)
		}
	}
}
