package modes

import (
	"fmt"
	"sort"

	"github.com/spf13/viper"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/progress"
	"labours-go/internal/readers"
)

// DevsEfforts generates plots for developers' effort analysis over time
func DevsEfforts(reader readers.Reader, output string, maxPeople int) error {
	quiet := viper.GetBool("quiet")
	progEstimator := progress.NewProgressEstimator(!quiet)

	totalPhases := 4 // data extraction, selection, analysis, plotting
	progEstimator.StartMultiOperation(totalPhases, "Developer Efforts Analysis")

	// Phase 1: Extract developer statistics
	progEstimator.NextOperation("Extracting developer statistics")
	developerStats, err := reader.GetDeveloperStats()
	if err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to get developer stats: %v", err)
	}

	// Phase 2: Select top developers
	progEstimator.NextOperation("Selecting top developers")
	if len(developerStats) > maxPeople {
		if !quiet {
			fmt.Printf("Picking top %d developers by commit count.\n", maxPeople)
		}
		developerStats = selectTopDevelopers(developerStats, maxPeople)
	}

	// Phase 3: Analyze developer efforts
	progEstimator.NextOperation("Analyzing developer efforts")
	effortMetrics := analyzeDevEfforts(developerStats)

	// Phase 4: Generate plots
	progEstimator.NextOperation("Generating visualization")
	if err := plotDevEfforts(effortMetrics, output); err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to generate developer efforts plots: %v", err)
	}

	progEstimator.FinishMultiOperation()
	if !quiet {
		fmt.Println("Developer efforts analysis completed successfully.")
	}
	return nil
}

// EffortMetric represents effort analysis for a developer
type EffortMetric struct {
	Name             string
	Commits          int
	LinesAdded       int
	LinesRemoved     int
	LinesModified    int
	FilesTouched     int
	ProductivityRank int
}

// analyzeDevEfforts performs effort analysis on developer statistics
func analyzeDevEfforts(stats []readers.DeveloperStat) []EffortMetric {
	metrics := make([]EffortMetric, 0, len(stats))

	// Calculate metrics for each developer
	for _, stat := range stats {
		metric := EffortMetric{
			Name:          stat.Name,
			Commits:       stat.Commits,
			LinesAdded:    stat.LinesAdded,
			LinesRemoved:  stat.LinesRemoved,
			LinesModified: stat.LinesModified,
			FilesTouched:  stat.FilesTouched,
		}
		metrics = append(metrics, metric)
	}

	// Sort by combined productivity score (commits + lines changed)
	sort.Slice(metrics, func(i, j int) bool {
		scoreI := float64(metrics[i].Commits) + float64(metrics[i].LinesAdded+metrics[i].LinesRemoved+metrics[i].LinesModified)*0.01
		scoreJ := float64(metrics[j].Commits) + float64(metrics[j].LinesAdded+metrics[j].LinesRemoved+metrics[j].LinesModified)*0.01
		return scoreI > scoreJ
	})

	// Assign productivity ranks
	for i := range metrics {
		metrics[i].ProductivityRank = i + 1
	}

	return metrics
}

// plotDevEfforts generates effort analysis plots
func plotDevEfforts(metrics []EffortMetric, output string) error {
	// Create commits vs lines changed scatter plot
	if err := plotCommitsVsLines(metrics, output); err != nil {
		return err
	}

	// Create productivity ranking bar chart
	if err := plotProductivityRanking(metrics, output); err != nil {
		return err
	}

	return nil
}

// plotCommitsVsLines creates scatter plot of commits vs total lines changed
func plotCommitsVsLines(metrics []EffortMetric, output string) error {
	p := plot.New()
	p.Title.Text = "Developer Efforts: Commits vs Lines Changed"
	p.X.Label.Text = "Total Commits"
	p.Y.Label.Text = "Total Lines Changed"

	// Prepare data points
	pts := make(plotter.XYs, len(metrics))
	for i, metric := range metrics {
		pts[i].X = float64(metric.Commits)
		pts[i].Y = float64(metric.LinesAdded + metric.LinesRemoved + metric.LinesModified)
	}

	// Create scatter plot
	scatter, err := plotter.NewScatter(pts)
	if err != nil {
		return fmt.Errorf("error creating scatter plot: %v", err)
	}

	scatter.Color = graphics.ColorPalette[0]
	p.Add(scatter)

	// Add developer names as labels (simplified)
	for i, metric := range metrics {
		if i < 10 { // Only label top 10 to avoid clutter
			label, err := plotter.NewLabels(plotter.XYLabels{
				XYs:    plotter.XYs{{X: float64(metric.Commits), Y: float64(metric.LinesAdded + metric.LinesRemoved + metric.LinesModified)}},
				Labels: []string{metric.Name},
			})
			if err == nil {
				p.Add(label)
			}
		}
	}

	pngFile, svgFile, err := savePlotPNGAndSVG(p, 16*vg.Inch, 8*vg.Inch, output, "devs_efforts_scatter")
	if err != nil {
		return fmt.Errorf("failed to save scatter plot: %v", err)
	}

	fmt.Printf("Saved developer efforts scatter plots to %s and %s\n", pngFile, svgFile)
	return nil
}

// plotProductivityRanking creates bar chart of developer productivity ranking
func plotProductivityRanking(metrics []EffortMetric, output string) error {
	p := plot.New()
	p.Title.Text = "Developer Productivity Ranking"
	p.X.Label.Text = "Developer Rank"
	p.Y.Label.Text = "Productivity Score (Commits + Lines/100)"

	// Prepare data for top developers only
	maxDev := len(metrics)
	if maxDev > 20 {
		maxDev = 20 // Show top 20 developers
	}

	values := make(plotter.Values, maxDev)
	for i := 0; i < maxDev; i++ {
		metric := metrics[i]
		values[i] = float64(metric.Commits) + float64(metric.LinesAdded+metric.LinesRemoved+metric.LinesModified)*0.01
	}

	// Create bar chart
	bars, err := plotter.NewBarChart(values, vg.Points(20))
	if err != nil {
		return fmt.Errorf("error creating bar chart: %v", err)
	}

	bars.Color = graphics.ColorPalette[1]
	p.Add(bars)

	pngFile, svgFile, err := savePlotPNGAndSVG(p, 16*vg.Inch, 8*vg.Inch, output, "devs_productivity_ranking")
	if err != nil {
		return fmt.Errorf("failed to save productivity ranking plot: %v", err)
	}

	fmt.Printf("Saved developer productivity ranking plots to %s and %s\n", pngFile, svgFile)
	return nil
}
