package modes

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"gonum.org/v1/plot/vg"
)

func TestCouplingPairGeometryMatchesMatplotlibDefaults(t *testing.T) {
	width := couplingFilePairBarWidth(15)
	wantWidth := vg.Points(800.0 / 15.0)
	if math.Abs(float64(width-wantWidth)) > 1e-9 {
		t.Fatalf("couplingFilePairBarWidth(15) = %v, want %v", width, wantWidth)
	}

	shotnessWidth := couplingBarWidth(20)
	wantShotnessWidth := vg.Points(800.0 / 20.0)
	if math.Abs(float64(shotnessWidth-wantShotnessWidth)) > 1e-9 {
		t.Fatalf("couplingBarWidth(20) = %v, want %v", shotnessWidth, wantShotnessWidth)
	}

	shotnessMinX, shotnessMaxX := shotnessCouplingPairXRange(20)
	if math.Abs(shotnessMinX-(-1.39)) > 1e-9 {
		t.Fatalf("shotnessCouplingPairXRange(20) min = %v, want -1.39", shotnessMinX)
	}
	if math.Abs(shotnessMaxX-20.39) > 1e-9 {
		t.Fatalf("shotnessCouplingPairXRange(20) max = %v, want 20.39", shotnessMaxX)
	}

	minX, maxX := couplingPairXRange(15)
	if math.Abs(minX-(-1.2)) > 1e-9 {
		t.Fatalf("couplingPairXRange(15) min = %v, want -1.2", minX)
	}
	if math.Abs(maxX-15.3) > 1e-9 {
		t.Fatalf("couplingPairXRange(15) max = %v, want 15.3", maxX)
	}
}

func TestPlotTopCouplingPairsWritesPNG(t *testing.T) {
	output := t.TempDir()
	if err := plotTopCouplingPairs(sampleFileCouplingAnalysis(), output); err != nil {
		t.Fatalf("plotTopCouplingPairs() failed: %v", err)
	}

	outputFile := filepath.Join(output, "top_file_coupling_pairs.png")
	if _, err := os.Stat(outputFile); err != nil {
		t.Fatalf("expected plot file %q: %v", outputFile, err)
	}
}

func sampleFileCouplingAnalysis() FileCouplingAnalysis {
	pairs := []FileCouplingPair{
		{File1: "pipeline.go", File2: "pipeline_test.go", CouplingScore: 18, CooccuranceCount: 18},
		{File1: "burndown.go", File2: "burndown_test.go", CouplingScore: 15, CooccuranceCount: 15},
		{File1: "identity.go", File2: "identity_test.go", CouplingScore: 14, CooccuranceCount: 14},
		{File1: "burndown.go", File2: "pipeline.go", CouplingScore: 12, CooccuranceCount: 12},
		{File1: "file.go", File2: "file_test.go", CouplingScore: 12, CooccuranceCount: 12},
		{File1: "README.md", File2: "labours.py", CouplingScore: 10, CooccuranceCount: 10},
		{File1: "burndown.go", File2: "couples.go", CouplingScore: 10, CooccuranceCount: 10},
		{File1: "blob_cache.go", File2: "burndown.go", CouplingScore: 9, CooccuranceCount: 9},
		{File1: "burndown.go", File2: "diff.go", CouplingScore: 9, CooccuranceCount: 9},
		{File1: "burndown.go", File2: "identity.go", CouplingScore: 9, CooccuranceCount: 9},
		{File1: "burndown.go", File2: "renames.go", CouplingScore: 9, CooccuranceCount: 9},
		{File1: "burndown.go", File2: "uast.go", CouplingScore: 9, CooccuranceCount: 9},
		{File1: "couples.go", File2: "identity.go", CouplingScore: 9, CooccuranceCount: 9},
		{File1: "couples.go", File2: "pipeline.go", CouplingScore: 9, CooccuranceCount: 9},
		{File1: "identity.go", File2: "renames.go", CouplingScore: 9, CooccuranceCount: 9},
	}
	return FileCouplingAnalysis{
		TopCoupling: pairs,
		Statistics: CouplingStatistics{
			TotalFiles:      18,
			TotalCoupling:   172,
			AverageCoupling: 11.47,
			MaxCoupling:     18,
			MinCoupling:     9,
		},
	}
}
