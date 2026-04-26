package modes

import (
	"fmt"
	"image/color"
	"math"
	"path/filepath"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/readers"
)

type ParallelismMetrics struct {
	TotalPeriods       int
	ParallelPeriods    int
	ParallelismIndex   float64
	PeakConcurrency    int
	AverageConcurrency float64
	DeveloperOverlaps  map[string]map[string]float64
	PeriodConcurrency  []int
	ActiveDevelopers   []string
}

// DevsParallel analyzes parallel development patterns and visualizes when developers work concurrently.
func DevsParallel(reader readers.Reader, output string, maxPeople int, allowSyntheticFallback bool) error {
	fmt.Println("Analyzing parallel development patterns...")

	// Get people burndown data to analyze temporal activity
	peopleBurndown, err := reader.GetPeopleBurndown()
	if err != nil {
		if allowSyntheticFallback {
			fmt.Printf("Warning: could not get people burndown data: %v\n", err)
			return generateSyntheticParallelAnalysis(reader, output)
		}
		return err
	}

	if len(peopleBurndown) == 0 {
		if allowSyntheticFallback {
			fmt.Println("No people burndown data available, using synthetic data")
			return generateSyntheticParallelAnalysis(reader, output)
		}
		return fmt.Errorf("%w: people burndown", readers.ErrAnalysisMissing)
	}
	peopleBurndown = filterPeopleBurndownByActivity(peopleBurndown, maxPeople)

	// Calculate parallelism metrics
	metrics := calculateParallelismMetrics(peopleBurndown)

	// Generate visualizations
	if err := plotParallelActivity(metrics, output); err != nil {
		return fmt.Errorf("failed to create parallel activity plot: %v", err)
	}

	if err := plotDeveloperConcurrency(metrics, output); err != nil {
		return fmt.Errorf("failed to create developer concurrency plot: %v", err)
	}

	// Print summary statistics
	printParallelismSummary(metrics)

	fmt.Println("Parallel development analysis completed successfully.")
	return nil
}

func filterPeopleBurndownByActivity(peopleBurndown []readers.PeopleBurndown, maxPeople int) []readers.PeopleBurndown {
	if maxPeople <= 0 || len(peopleBurndown) <= maxPeople {
		return peopleBurndown
	}

	filtered := append([]readers.PeopleBurndown(nil), peopleBurndown...)
	sort.SliceStable(filtered, func(i, j int) bool {
		left := peopleBurndownActivity(filtered[i])
		right := peopleBurndownActivity(filtered[j])
		if left == right {
			return filtered[i].Person < filtered[j].Person
		}
		return left > right
	})
	fmt.Printf("Warning: truncated people to the most active %d\n", maxPeople)
	return filtered[:maxPeople]
}

func peopleBurndownActivity(person readers.PeopleBurndown) int {
	total := 0
	for _, row := range person.Matrix {
		for _, value := range row {
			total += value
		}
	}
	return total
}

// calculateParallelismMetrics analyzes the temporal activity data to find parallel development patterns
func calculateParallelismMetrics(peopleBurndown []readers.PeopleBurndown) ParallelismMetrics {
	if len(peopleBurndown) == 0 {
		return ParallelismMetrics{}
	}

	// Find the maximum time period across all developers
	maxPeriods := 0
	for _, person := range peopleBurndown {
		if len(person.Matrix) > maxPeriods {
			maxPeriods = len(person.Matrix)
		}
	}

	if maxPeriods == 0 {
		return ParallelismMetrics{}
	}

	// Create a matrix of developer activity over time
	developers := make([]string, len(peopleBurndown))
	activityMatrix := make([][]bool, len(peopleBurndown))

	for i, person := range peopleBurndown {
		developers[i] = person.Person
		activityMatrix[i] = make([]bool, maxPeriods)

		// Mark periods where developer was active (had any commits/changes)
		for j, period := range person.Matrix {
			if j >= maxPeriods {
				break
			}
			// Consider active if there's any value in the period (sum > 0)
			active := false
			for _, val := range period {
				if val > 0 {
					active = true
					break
				}
			}
			activityMatrix[i][j] = active
		}
	}

	// Calculate concurrency for each time period
	periodConcurrency := make([]int, maxPeriods)
	parallelPeriods := 0
	totalConcurrency := 0
	peakConcurrency := 0

	for period := 0; period < maxPeriods; period++ {
		concurrent := 0
		for dev := 0; dev < len(developers); dev++ {
			if activityMatrix[dev][period] {
				concurrent++
			}
		}
		periodConcurrency[period] = concurrent
		totalConcurrency += concurrent

		if concurrent > 1 {
			parallelPeriods++
		}
		if concurrent > peakConcurrency {
			peakConcurrency = concurrent
		}
	}

	// Calculate developer overlap coefficients
	developerOverlaps := make(map[string]map[string]float64)
	for i, dev1 := range developers {
		developerOverlaps[dev1] = make(map[string]float64)
		for j, dev2 := range developers {
			if i == j {
				developerOverlaps[dev1][dev2] = 1.0
				continue
			}

			// Count periods where both developers were active
			overlap := 0
			dev1Active := 0
			dev2Active := 0

			for period := 0; period < maxPeriods; period++ {
				if activityMatrix[i][period] {
					dev1Active++
				}
				if activityMatrix[j][period] {
					dev2Active++
				}
				if activityMatrix[i][period] && activityMatrix[j][period] {
					overlap++
				}
			}

			// Calculate Jaccard similarity coefficient
			union := dev1Active + dev2Active - overlap
			if union > 0 {
				developerOverlaps[dev1][dev2] = float64(overlap) / float64(union)
			} else {
				developerOverlaps[dev1][dev2] = 0.0
			}
		}
	}

	parallelismIndex := 0.0
	if maxPeriods > 0 {
		parallelismIndex = float64(parallelPeriods) / float64(maxPeriods) * 100
	}

	averageConcurrency := 0.0
	if maxPeriods > 0 {
		averageConcurrency = float64(totalConcurrency) / float64(maxPeriods)
	}

	return ParallelismMetrics{
		TotalPeriods:       maxPeriods,
		ParallelPeriods:    parallelPeriods,
		ParallelismIndex:   parallelismIndex,
		PeakConcurrency:    peakConcurrency,
		AverageConcurrency: averageConcurrency,
		DeveloperOverlaps:  developerOverlaps,
		PeriodConcurrency:  periodConcurrency,
		ActiveDevelopers:   developers,
	}
}

// plotParallelActivity creates a timeline showing concurrent developer activity
func plotParallelActivity(metrics ParallelismMetrics, output string) error {
	if len(metrics.PeriodConcurrency) == 0 {
		metrics.PeriodConcurrency = []int{0}
	}

	p := plot.New()
	p.Title.Text = "Parallel Development Activity Over Time"
	p.X.Label.Text = "Time Period"
	p.Y.Label.Text = "Number of Concurrent Developers"

	// Create points for the concurrency timeline
	pts := make(plotter.XYs, len(metrics.PeriodConcurrency))
	for i, concurrency := range metrics.PeriodConcurrency {
		pts[i].X = float64(i)
		pts[i].Y = float64(concurrency)
	}

	// Create line plot
	line, err := plotter.NewLine(pts)
	if err != nil {
		return fmt.Errorf("error creating line plot: %v", err)
	}
	line.Color = graphics.ColorPalette[0]
	line.Width = vg.Points(2)

	// Add a filled polygon area under the line
	// Create polygon points for filled area
	areaPoints := make(plotter.XYs, len(pts)+2)
	areaPoints[0] = plotter.XY{X: pts[0].X, Y: 0}
	for i, pt := range pts {
		areaPoints[i+1] = pt
	}
	areaPoints[len(areaPoints)-1] = plotter.XY{X: pts[len(pts)-1].X, Y: 0}

	polygon, err := plotter.NewPolygon(areaPoints)
	if err == nil {
		// Create a semi-transparent version of the color
		baseColor := graphics.ColorPalette[0].(color.RGBA)
		polygon.Color = color.RGBA{R: baseColor.R, G: baseColor.G, B: baseColor.B, A: 100}
		p.Add(polygon)
	}

	p.Add(line)
	p.Legend.Add("Concurrent Developers", line)

	// Add horizontal line for average concurrency
	if metrics.AverageConcurrency > 0 {
		avgLine := plotter.NewFunction(func(x float64) float64 {
			return metrics.AverageConcurrency
		})
		avgLine.Color = graphics.ColorPalette[1]
		avgLine.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
		p.Add(avgLine)
		p.Legend.Add(fmt.Sprintf("Average (%.1f)", metrics.AverageConcurrency), avgLine)
	}

	// Save PNG with dynamic sizing
	width, height := graphics.GetPlotSize(graphics.ChartTypeDefault)
	outputFile := filepath.Join(output, "parallel_activity.png")
	if err := p.Save(width, height, outputFile); err != nil {
		return fmt.Errorf("failed to save parallel activity plot: %v", err)
	}

	// Save SVG
	outputFileSVG := filepath.Join(output, "parallel_activity.svg")
	if err := p.Save(width, height, outputFileSVG); err != nil {
		return fmt.Errorf("failed to save parallel activity SVG: %v", err)
	}

	fmt.Printf("Saved parallel activity plots to %s and %s\n", outputFile, outputFileSVG)
	return nil
}

// plotDeveloperConcurrency creates a heatmap/bar chart showing developer overlap patterns
func plotDeveloperConcurrency(metrics ParallelismMetrics, output string) error {
	if len(metrics.ActiveDevelopers) == 0 {
		return fmt.Errorf("no active developers found")
	}

	// Create a bar chart showing average overlap per developer
	p := plot.New()
	p.Title.Text = "Developer Collaboration Patterns"
	p.X.Label.Text = "Developers"
	p.Y.Label.Text = "Average Overlap Coefficient"

	// Calculate average overlap for each developer
	devOverlapAvgs := make([]float64, len(metrics.ActiveDevelopers))
	for i, dev := range metrics.ActiveDevelopers {
		total := 0.0
		count := 0
		for otherDev, overlap := range metrics.DeveloperOverlaps[dev] {
			if dev != otherDev {
				total += overlap
				count++
			}
		}
		if count > 0 {
			devOverlapAvgs[i] = total / float64(count)
		}
		if math.IsNaN(devOverlapAvgs[i]) || math.IsInf(devOverlapAvgs[i], 0) {
			devOverlapAvgs[i] = 0
		}
	}

	// Create bar chart
	bars := make(plotter.Values, len(devOverlapAvgs))
	for i, avg := range devOverlapAvgs {
		bars[i] = avg
	}

	barChart, err := plotter.NewBarChart(bars, vg.Points(20))
	if err != nil {
		return fmt.Errorf("error creating bar chart: %v", err)
	}
	barChart.Color = graphics.ColorPalette[2]

	p.Add(barChart)

	// Set custom X-axis labels
	p.NominalX(metrics.ActiveDevelopers...)

	// Save PNG with dynamic sizing
	width2, height2 := graphics.GetPlotSize(graphics.ChartTypeDefault)
	outputFile := filepath.Join(output, "developer_concurrency.png")
	if err := p.Save(width2, height2, outputFile); err != nil {
		return fmt.Errorf("failed to save developer concurrency plot: %v", err)
	}

	// Save SVG
	outputFileSVG := filepath.Join(output, "developer_concurrency.svg")
	if err := p.Save(width2, height2, outputFileSVG); err != nil {
		return fmt.Errorf("failed to save developer concurrency SVG: %v", err)
	}

	fmt.Printf("Saved developer concurrency plots to %s and %s\n", outputFile, outputFileSVG)
	return nil
}

// generateSyntheticParallelAnalysis creates a fallback analysis when real data is not available
func generateSyntheticParallelAnalysis(reader readers.Reader, output string) error {
	fmt.Println("Generating synthetic parallel development analysis...")

	// Try to get basic developer stats for fallback
	developerStats, err := reader.GetDeveloperStats()
	if err != nil {
		fmt.Printf("Warning: could not get developer stats: %v\n", err)
		return fmt.Errorf("no data available for parallel analysis")
	}

	if len(developerStats) == 0 {
		return fmt.Errorf("no developer data available")
	}

	// Create synthetic parallel activity data
	numPeriods := 52 // 52 weeks
	metrics := ParallelismMetrics{
		TotalPeriods:       numPeriods,
		ParallelPeriods:    int(float64(numPeriods) * 0.6), // Assume 60% parallel activity
		ParallelismIndex:   60.0,
		PeakConcurrency:    min(len(developerStats), 4),
		AverageConcurrency: math.Min(float64(len(developerStats))*0.7, 3.0),
		ActiveDevelopers:   make([]string, 0, len(developerStats)),
		PeriodConcurrency:  make([]int, numPeriods),
		DeveloperOverlaps:  make(map[string]map[string]float64),
	}

	// Generate synthetic data based on developer stats
	for i, dev := range developerStats {
		metrics.ActiveDevelopers = append(metrics.ActiveDevelopers, dev.Name)

		// Initialize overlaps
		metrics.DeveloperOverlaps[dev.Name] = make(map[string]float64)
		for j, otherDev := range developerStats {
			if i == j {
				metrics.DeveloperOverlaps[dev.Name][otherDev.Name] = 1.0
			} else {
				// Synthetic overlap based on relative activity
				ratio := 0.0
				if max(dev.Commits, otherDev.Commits) > 0 {
					ratio = float64(min(dev.Commits, otherDev.Commits)) / float64(max(dev.Commits, otherDev.Commits))
				}
				overlap := ratio * (0.3 + 0.4*math.Sin(float64(i+j)*0.5)) // Add some variation
				metrics.DeveloperOverlaps[dev.Name][otherDev.Name] = math.Max(0, math.Min(1, overlap))
			}
		}
	}

	// Generate synthetic period concurrency
	for i := 0; i < numPeriods; i++ {
		// Simulate realistic parallel activity patterns
		baseActivity := 1 + int(metrics.AverageConcurrency*math.Sin(float64(i)*0.3)+0.5)
		variation := int(math.Sin(float64(i)*0.1) * 2)
		concurrency := max(1, min(len(developerStats), baseActivity+variation))
		metrics.PeriodConcurrency[i] = concurrency
	}

	// Generate plots with synthetic data
	if err := plotParallelActivity(metrics, output); err != nil {
		return fmt.Errorf("failed to create parallel activity plot: %v", err)
	}

	if err := plotDeveloperConcurrency(metrics, output); err != nil {
		return fmt.Errorf("failed to create developer concurrency plot: %v", err)
	}

	printParallelismSummary(metrics)

	fmt.Println("Synthetic parallel development analysis completed.")
	return nil
}

// printParallelismSummary displays key metrics about parallel development
func printParallelismSummary(metrics ParallelismMetrics) {
	fmt.Println("\n=== Parallel Development Summary ===")
	fmt.Printf("Total Time Periods: %d\n", metrics.TotalPeriods)
	fmt.Printf("Periods with Parallel Activity: %d (%.1f%%)\n",
		metrics.ParallelPeriods, metrics.ParallelismIndex)
	fmt.Printf("Peak Concurrent Developers: %d\n", metrics.PeakConcurrency)
	fmt.Printf("Average Concurrent Developers: %.2f\n", metrics.AverageConcurrency)
	fmt.Printf("Active Developers: %d\n", len(metrics.ActiveDevelopers))

	if len(metrics.ActiveDevelopers) > 1 {
		fmt.Println("\nTop Developer Collaborations:")

		type overlap struct {
			pair    string
			overlap float64
		}

		var overlaps []overlap
		processed := make(map[string]bool)

		for dev1, others := range metrics.DeveloperOverlaps {
			for dev2, ovr := range others {
				if dev1 != dev2 {
					pairKey := dev1 + "-" + dev2
					reversePairKey := dev2 + "-" + dev1

					if !processed[pairKey] && !processed[reversePairKey] {
						overlaps = append(overlaps, overlap{
							pair:    fmt.Sprintf("%s ↔ %s", dev1, dev2),
							overlap: ovr,
						})
						processed[pairKey] = true
						processed[reversePairKey] = true
					}
				}
			}
		}

		sort.Slice(overlaps, func(i, j int) bool {
			return overlaps[i].overlap > overlaps[j].overlap
		})

		maxDisplay := min(5, len(overlaps))
		for i := 0; i < maxDisplay; i++ {
			fmt.Printf("  %s: %.3f\n", overlaps[i].pair, overlaps[i].overlap)
		}
	}
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
