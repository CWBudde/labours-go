package modes

import (
	"fmt"
	"image/color"
	"path/filepath"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
	"labours-go/internal/readers"
)

// CouplesShotness generates shotness-based coupling analysis and visualization
func CouplesShotness(reader readers.Reader, output string) error {
	return runCouplingMode(
		"Shotness Coupling Analysis",
		"shotness coupling",
		output,
		reader.GetShotnessCooccurrence,
		func(names []string, matrix [][]int, output string) error {
			return plotShotnessCoupling(analyzeShotnessCoupling(names, matrix), output)
		},
	)
}

// ShotnessCouplingPair represents a coupling relationship between two shotness entities
type ShotnessCouplingPair struct {
	Entity1          string
	Entity2          string
	CouplingScore    float64
	CooccuranceCount int
}

// ShotnessCouplingAnalysis represents the complete shotness coupling analysis results
type ShotnessCouplingAnalysis struct {
	EntityNames    []string
	CouplingMatrix [][]int
	TopCoupling    []ShotnessCouplingPair
	Statistics     ShotnessCouplingStatistics
}

// ShotnessCouplingStatistics provides summary statistics about shotness coupling
type ShotnessCouplingStatistics struct {
	TotalEntities   int
	TotalCouplings  int
	AverageCoupling float64
	MaxCoupling     int
	MinCoupling     int
}

// analyzeShotnessCoupling performs analysis on shotness coupling data
func analyzeShotnessCoupling(entityNames []string, couplingMatrix [][]int) ShotnessCouplingAnalysis {
	pairs, stats := analyzeCouplingPairs(entityNames, couplingMatrix, 25)
	shotnessPairs := make([]ShotnessCouplingPair, len(pairs))
	for i, pair := range pairs {
		shotnessPairs[i] = ShotnessCouplingPair{
			Entity1:          pair.Name1,
			Entity2:          pair.Name2,
			CouplingScore:    pair.Score,
			CooccuranceCount: pair.Count,
		}
	}

	analysis := ShotnessCouplingAnalysis{
		EntityNames:    entityNames,
		CouplingMatrix: couplingMatrix,
		TopCoupling:    shotnessPairs,
		Statistics: ShotnessCouplingStatistics{
			TotalEntities:   len(entityNames),
			TotalCouplings:  stats.Total,
			AverageCoupling: stats.Average,
			MaxCoupling:     stats.Max,
			MinCoupling:     stats.Min,
		},
	}

	return analysis
}

// plotShotnessCoupling generates coupling visualization plots
func plotShotnessCoupling(analysis ShotnessCouplingAnalysis, output string) error {
	// Create heatmap for shotness entities
	if err := plotShotnessCouplingHeatmap(analysis, output); err != nil {
		return err
	}

	// Create bar chart of top coupling pairs
	if err := plotTopShotnessCouplingPairs(analysis, output); err != nil {
		return err
	}

	return nil
}

// plotShotnessCouplingHeatmap creates a heatmap of shotness coupling relationships
func plotShotnessCouplingHeatmap(analysis ShotnessCouplingAnalysis, output string) error {
	if len(analysis.CouplingMatrix) == 0 {
		return fmt.Errorf("no coupling matrix data available")
	}

	pngFile := filepath.Join(output, "shotness_coupling_heatmap.png")
	if err := plotPythonCouplingHeatmap("Shotness Coupling Heatmap", pngFile, analysis.EntityNames, analysis.CouplingMatrix, "Greens"); err != nil {
		return fmt.Errorf("failed to save heatmap: %v", err)
	}

	svgFile := filepath.Join(output, "shotness_coupling_heatmap.svg")
	if err := plotPythonCouplingHeatmap("Shotness Coupling Heatmap", svgFile, analysis.EntityNames, analysis.CouplingMatrix, "Greens"); err != nil {
		return fmt.Errorf("failed to save heatmap: %v", err)
	}

	fmt.Printf("Saved shotness coupling heatmap to %s and %s\n", pngFile, svgFile)
	return nil
}

// plotTopShotnessCouplingPairs creates a bar chart of the most coupled shotness entities
func plotTopShotnessCouplingPairs(analysis ShotnessCouplingAnalysis, output string) error {
	if len(analysis.TopCoupling) == 0 {
		return fmt.Errorf("no coupling pairs data available")
	}

	p := plot.New()
	p.X.Label.Text = "Coupling Pair Rank"
	p.Y.Label.Text = "Coupling Score"

	// Prepare data for bar chart
	maxPairs := len(analysis.TopCoupling)
	if maxPairs > 20 {
		maxPairs = 20 // Show top 20 pairs
	}

	values := make(plotter.Values, maxPairs)
	for i := 0; i < maxPairs; i++ {
		values[i] = analysis.TopCoupling[i].CouplingScore
	}

	// Create bar chart
	bars, err := plotter.NewBarChart(values, couplingBarWidth(maxPairs))
	if err != nil {
		return fmt.Errorf("error creating bar chart: %v", err)
	}

	bars.Color = color.RGBA{R: 76, G: 120, B: 168, A: 255}
	bars.LineStyle = draw.LineStyle{Color: color.RGBA{}, Width: 0}
	p.Add(bars)

	labels := make([]string, maxPairs)
	for i := 0; i < maxPairs; i++ {
		pair := analysis.TopCoupling[i]
		labels[i] = compactCouplingPairLabel(filepath.Base(pair.Entity1)+"-"+filepath.Base(pair.Entity2), 28)
	}
	addTopCouplingPairLabels(p, labels, values, 10)

	// Create custom tick marks
	ticks := make([]plot.Tick, maxPairs)
	for i := range ticks {
		ticks[i] = plot.Tick{
			Value: float64(i),
			Label: strconv.Itoa(i + 1), // Just show rank numbers
		}
	}
	p.X.Tick.Marker = plot.ConstantTicks(ticks)
	p.X.Min, p.X.Max = shotnessCouplingPairXRange(maxPairs)
	p.Y.Min = 0
	p.Y.Max = maxCouplingValue(values) * 1.05
	p.Y.Tick.Marker = plot.ConstantTicks(couplingScoreTicks(p.Y.Max, 2, 0))
	addCouplingPairsTitle(p, "Top Shotness Coupling Pairs", float64(maxPairs-1)/2, p.Y.Max)
	p.Add(plotTopPadding{Height: vg.Points(84)})
	p.Add(plotAxesRectangle{})

	pngFile, svgFile, err := savePlotPNGAndSVG(p, 16*vg.Inch, 8*vg.Inch, output, "top_shotness_coupling_pairs")
	if err != nil {
		return fmt.Errorf("failed to save coupling pairs plot: %v", err)
	}

	fmt.Printf("Saved top shotness coupling pairs plots to %s and %s\n", pngFile, svgFile)

	// Print summary information
	fmt.Printf("Shotness Coupling Analysis Summary:\n")
	fmt.Printf("  Total entities: %d\n", analysis.Statistics.TotalEntities)
	fmt.Printf("  Total coupling relationships: %d\n", len(analysis.TopCoupling))
	fmt.Printf("  Average coupling score: %.2f\n", analysis.Statistics.AverageCoupling)
	fmt.Printf("  Max coupling score: %d\n", analysis.Statistics.MaxCoupling)

	return nil
}
