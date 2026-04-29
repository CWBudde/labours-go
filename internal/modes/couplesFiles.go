package modes

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"strconv"

	"github.com/spf13/viper"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/font"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/text"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/progress"
	"labours-go/internal/readers"
)

// CouplesFiles generates file coupling analysis and visualization
func CouplesFiles(reader readers.Reader, output string) error {
	quiet := viper.GetBool("quiet")
	progEstimator := progress.NewProgressEstimator(!quiet)

	totalPhases := 3 // data extraction, analysis, plotting
	progEstimator.StartMultiOperation(totalPhases, "File Coupling Analysis")

	// Phase 1: Extract file coupling data
	progEstimator.NextOperation("Extracting file coupling data")
	fileNames, couplingMatrix, err := reader.GetFileCooccurrence()
	if err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to get file coupling data: %v", err)
	}

	if len(fileNames) == 0 {
		progEstimator.FinishMultiOperation()
		if !quiet {
			fmt.Println("No file coupling data available")
		}
		return nil
	}

	// Phase 2: Analyze coupling patterns
	progEstimator.NextOperation("Analyzing coupling patterns")
	couplingAnalysis := analyzeFileCoupling(fileNames, couplingMatrix)

	// Phase 3: Generate visualizations
	progEstimator.NextOperation("Generating visualization")
	if err := plotFileCoupling(couplingAnalysis, output); err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to generate file coupling plots: %v", err)
	}

	progEstimator.FinishMultiOperation()
	if !quiet {
		fmt.Println("File coupling analysis completed successfully.")
	}
	return nil
}

// FileCouplingPair represents a coupling relationship between two files
type FileCouplingPair struct {
	File1            string
	File2            string
	CouplingScore    float64
	CooccuranceCount int
}

// FileCouplingAnalysis represents the complete coupling analysis results
type FileCouplingAnalysis struct {
	FileNames      []string
	CouplingMatrix [][]int
	TopCoupling    []FileCouplingPair
	Statistics     CouplingStatistics
}

// CouplingStatistics provides summary statistics about file coupling
type CouplingStatistics struct {
	TotalFiles      int
	TotalCoupling   int
	AverageCoupling float64
	MaxCoupling     int
	MinCoupling     int
}

// analyzeFileCoupling performs analysis on file coupling data
func analyzeFileCoupling(fileNames []string, couplingMatrix [][]int) FileCouplingAnalysis {
	analysis := FileCouplingAnalysis{
		FileNames:      fileNames,
		CouplingMatrix: couplingMatrix,
	}

	// Calculate coupling pairs and statistics
	var pairs []FileCouplingPair
	totalCoupling := 0
	maxCoupling := 0
	minCoupling := int(^uint(0) >> 1) // Max int

	for i := 0; i < len(fileNames); i++ {
		for j := i + 1; j < len(fileNames); j++ {
			if i < len(couplingMatrix) && j < len(couplingMatrix[i]) {
				coupling := couplingMatrix[i][j]
				totalCoupling += coupling

				if coupling > maxCoupling {
					maxCoupling = coupling
				}
				if coupling < minCoupling && coupling > 0 {
					minCoupling = coupling
				}

				if coupling > 0 {
					pairs = append(pairs, FileCouplingPair{
						File1:            fileNames[i],
						File2:            fileNames[j],
						CouplingScore:    float64(coupling),
						CooccuranceCount: coupling,
					})
				}
			}
		}
	}

	// Sort pairs by coupling score (descending)
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].CouplingScore < pairs[j].CouplingScore {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Take top 20 couples for visualization
	if len(pairs) > 20 {
		analysis.TopCoupling = pairs[:20]
	} else {
		analysis.TopCoupling = pairs
	}

	// Calculate statistics
	avgCoupling := 0.0
	if len(pairs) > 0 {
		avgCoupling = float64(totalCoupling) / float64(len(pairs))
	}

	analysis.Statistics = CouplingStatistics{
		TotalFiles:      len(fileNames),
		TotalCoupling:   totalCoupling,
		AverageCoupling: avgCoupling,
		MaxCoupling:     maxCoupling,
		MinCoupling:     minCoupling,
	}

	return analysis
}

// plotFileCoupling generates coupling visualization plots
func plotFileCoupling(analysis FileCouplingAnalysis, output string) error {
	// Create heatmap for top coupled files
	if err := plotCouplingHeatmap(analysis, output); err != nil {
		return err
	}

	// Create bar chart of top coupling pairs
	if err := plotTopCouplingPairs(analysis, output); err != nil {
		return err
	}

	return nil
}

// plotCouplingHeatmap creates a heatmap of file coupling relationships
func plotCouplingHeatmap(analysis FileCouplingAnalysis, output string) error {
	if len(analysis.CouplingMatrix) == 0 {
		return fmt.Errorf("no coupling matrix data available")
	}

	// Create heatmap data
	heatmapData := make([][]float64, len(analysis.CouplingMatrix))
	maxVal := 0.0
	minVal := float64(analysis.Statistics.MaxCoupling)

	for i, row := range analysis.CouplingMatrix {
		heatmapData[i] = make([]float64, len(row))
		for j, val := range row {
			heatmapData[i][j] = float64(val)
			if float64(val) > maxVal {
				maxVal = float64(val)
			}
			if float64(val) < minVal && val > 0 {
				minVal = float64(val)
			}
		}
	}

	// Create custom palette for heatmap
	palette := &graphics.CustomPalette{
		Colors: []color.Color{
			color.RGBA{255, 255, 255, 255}, // White for low values
			color.RGBA{255, 200, 200, 255}, // Light red
			color.RGBA{255, 100, 100, 255}, // Medium red
			color.RGBA{200, 0, 0, 255},     // Dark red for high values
		},
		Min: minVal,
		Max: maxVal,
	}

	// Create plot
	p := plot.New()
	p.Title.Text = "File Coupling Heatmap"

	// Create heatmap
	heatmap := graphics.NewHeatMap(heatmapData, analysis.FileNames, analysis.FileNames, palette)
	p.Add(heatmap)

	// Save the plot
	outputFile := filepath.Join(output, "file_coupling_heatmap.png")
	widthHeat, heightHeat := graphics.GetPlotSize(graphics.ChartTypeSquare)
	if err := p.Save(widthHeat, heightHeat, outputFile); err != nil {
		return fmt.Errorf("failed to save heatmap: %v", err)
	}

	fmt.Printf("Saved file coupling heatmap to %s\n", outputFile)
	return nil
}

// plotTopCouplingPairs creates a bar chart of the most coupled file pairs
func plotTopCouplingPairs(analysis FileCouplingAnalysis, output string) error {
	if len(analysis.TopCoupling) == 0 {
		return fmt.Errorf("no coupling pairs data available")
	}

	p := plot.New()
	p.X.Label.Text = "Coupling Pair Rank"
	p.Y.Label.Text = "Coupling Score"

	// Prepare data for bar chart
	maxPairs := len(analysis.TopCoupling)
	if maxPairs > 15 {
		maxPairs = 15 // Show top 15 pairs
	}

	values := make(plotter.Values, maxPairs)
	for i := 0; i < maxPairs; i++ {
		values[i] = analysis.TopCoupling[i].CouplingScore
	}

	// Create bar chart
	bars, err := plotter.NewBarChart(values, vg.Points(55))
	if err != nil {
		return fmt.Errorf("error creating bar chart: %v", err)
	}

	bars.Color = color.RGBA{R: 76, G: 120, B: 168, A: 255}
	p.Add(bars)

	labelCount := maxPairs
	if labelCount > 10 {
		labelCount = 10
	}
	labels := make([]string, labelCount)
	labelPoints := make(plotter.XYs, labelCount)
	for i := 0; i < labelCount; i++ {
		pair := analysis.TopCoupling[i]
		labels[i] = compactCouplingPairLabel(filepath.Base(pair.File1)+"-"+filepath.Base(pair.File2), 28)
		labelPoints[i].X = float64(i)
		labelPoints[i].Y = values[i]
	}

	labelPlotter, err := plotter.NewLabels(plotter.XYLabels{
		XYs:    labelPoints,
		Labels: labels,
	})
	if err != nil {
		return fmt.Errorf("error creating coupling pair labels: %v", err)
	}
	labelStyle := text.Style{
		Color:    color.Black,
		Font:     font.From(plot.DefaultFont, vg.Points(7)),
		Rotation: 70 * math.Pi / 180,
		XAlign:   text.XLeft,
		YAlign:   text.YBottom,
		Handler:  plot.DefaultTextHandler,
	}
	for i := range labelPlotter.TextStyle {
		labelPlotter.TextStyle[i] = labelStyle
	}
	labelPlotter.Offset = vg.Point{Y: vg.Points(2)}
	p.Add(labelPlotter)

	// Create custom tick marks
	ticks := make([]plot.Tick, maxPairs)
	for i := range ticks {
		ticks[i] = plot.Tick{
			Value: float64(i),
			Label: strconv.Itoa(i + 1), // Just show rank numbers
		}
	}
	p.X.Tick.Marker = plot.ConstantTicks(ticks)
	p.X.Min = -0.5
	p.X.Max = float64(maxPairs) - 0.5
	p.Y.Min = 0
	p.Y.Max = maxCouplingValue(values) * 1.05
	p.Y.Tick.Marker = plot.ConstantTicks(couplingScoreTicks(p.Y.Max))
	addCouplingPairsTitle(p, "Top File Coupling Pairs", float64(maxPairs-1)/2, p.Y.Max)

	// Save the plot
	outputFile := filepath.Join(output, "top_file_coupling_pairs.png")
	widthBar, heightBar := graphics.GetPlotSize(graphics.ChartTypeDefault)
	if err := p.Save(widthBar, heightBar, outputFile); err != nil {
		return fmt.Errorf("failed to save coupling pairs plot: %v", err)
	}

	fmt.Printf("Saved top coupling pairs plot to %s\n", outputFile)

	// Print summary information
	fmt.Printf("File Coupling Analysis Summary:\n")
	fmt.Printf("  Total files: %d\n", analysis.Statistics.TotalFiles)
	fmt.Printf("  Total coupling relationships: %d\n", len(analysis.TopCoupling))
	fmt.Printf("  Average coupling score: %.2f\n", analysis.Statistics.AverageCoupling)
	fmt.Printf("  Max coupling score: %d\n", analysis.Statistics.MaxCoupling)

	return nil
}

func addCouplingPairsTitle(p *plot.Plot, title string, x, y float64) {
	titlePlotter, err := plotter.NewLabels(plotter.XYLabels{
		XYs: plotter.XYs{{X: x, Y: y}},
		Labels: []string{
			title,
		},
	})
	if err != nil {
		return
	}
	titlePlotter.TextStyle[0] = text.Style{
		Color:   color.Black,
		Font:    font.From(plot.DefaultFont, vg.Points(12)),
		XAlign:  text.XCenter,
		YAlign:  text.YBottom,
		Handler: plot.DefaultTextHandler,
	}
	titlePlotter.Offset = vg.Point{Y: vg.Points(8)}
	p.Add(titlePlotter)
}

func compactCouplingPairLabel(label string, limit int) string {
	if limit <= 0 || len(label) <= limit {
		return label
	}
	if limit <= 3 {
		return label[len(label)-limit:]
	}
	return "..." + label[len(label)-(limit-3):]
}

func maxCouplingValue(values plotter.Values) float64 {
	maxValue := 0.0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

func couplingScoreTicks(maxValue float64) []plot.Tick {
	if maxValue <= 0 {
		return []plot.Tick{{Value: 0, Label: "0.0"}}
	}
	ticks := []plot.Tick{}
	for value := 0.0; value <= maxValue; value += 2.5 {
		label := fmt.Sprintf("%.1f", value)
		if value == 0 {
			label = "0.0"
		}
		ticks = append(ticks, plot.Tick{Value: value, Label: label})
	}
	return ticks
}
