package modes

import (
	"fmt"
	"image/color"
	"sort"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"labours-go/internal/graphics"
	"labours-go/internal/progress"
	"labours-go/internal/readers"
)

// Devs generates plots for individual developers' contributions over time.
func Devs(reader readers.Reader, output string, maxPeople int) error {
	// Initialize progress tracking
	quiet := viper.GetBool("quiet")
	progEstimator := progress.NewProgressEstimator(!quiet)

	// Start multi-phase operation for developer analysis
	totalPhases := 5 // data extraction, selection, time series generation, clustering, plotting
	progEstimator.StartMultiOperation(totalPhases, "Developer Analysis")

	// Phase 1: Extract developer statistics
	progEstimator.NextOperation("Extracting developer statistics")
	if timeSeries, err := reader.GetDeveloperTimeSeriesData(); err == nil && len(timeSeries.Days) > 0 {
		startUnix, endUnix := reader.GetHeader()
		if startUnix > 0 && endUnix > startUnix {
			if err := plotDevsPythonStyle(timeSeries, startUnix, endUnix, output, maxPeople); err != nil {
				progEstimator.FinishMultiOperation()
				return fmt.Errorf("failed to generate Python-style developer plot: %v", err)
			}
			progEstimator.FinishMultiOperation()
			if !quiet {
				fmt.Println("Developer plots generated successfully.")
			}
			return nil
		}
	}

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

	// Phase 3: Generate time series data for each developer
	progEstimator.NextOperation("Generating time series data")
	devSeries := generateTimeSeriesWithProgress(developerStats, progEstimator)

	// Phase 4: Cluster developers by contribution patterns
	progEstimator.NextOperation("Clustering developers")
	clusters := clusterDevelopers(devSeries)

	// Phase 5: Plot the developer contributions
	progEstimator.NextOperation("Generating visualization")
	if err := plotDevs(developerStats, devSeries, clusters, output); err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to generate developer plots: %v", err)
	}

	progEstimator.FinishMultiOperation()
	if !quiet {
		fmt.Println("Developer plots generated successfully.")
	}
	return nil
}

type devSeriesRow struct {
	Index       int
	Name        string
	Series      []float64
	Commits     int
	LinesAdded  int
	LinesRemove int
	LinesChange int
}

func plotDevsPythonStyle(timeSeries *readers.DeveloperTimeSeriesData, startUnix, endUnix int64, output string, maxPeople int) error {
	rows, dates := buildDeveloperSeriesRows(timeSeries, startUnix, endUnix, maxPeople)
	if len(rows) == 0 {
		return fmt.Errorf("no developer time series to plot")
	}

	p := plot.New()
	p.HideY()
	p.X.Tick.Marker = monthlyTicks(dates[0], dates[len(dates)-1])
	p.X.Min = float64(dates[0].Unix())
	p.X.Max = float64(dates[len(dates)-1].Unix())

	maxY := 0.0
	for _, row := range rows {
		for _, v := range row.Series {
			if v > maxY {
				maxY = v
			}
		}
	}
	if maxY <= 0 {
		maxY = 1
	}
	p.Y.Min = 0
	p.Y.Max = maxY * 7

	colors := graphics.PythonLaboursColorPalette(len(rows))
	for i, row := range rows {
		poly, err := areaPolygon(dates, row.Series)
		if err != nil {
			return err
		}
		rgba := colorToRGBA(colors[i%len(colors)], 255)
		poly.Color = rgba
		poly.LineStyle.Width = 0
		p.Add(poly)
	}

	labelY := p.Y.Max * 0.08
	var leftLabels plotter.XYLabels
	var rightLabels plotter.XYLabels
	for _, row := range rows {
		leftLabels.XYs = append(leftLabels.XYs, plotter.XY{X: p.X.Min, Y: labelY})
		leftLabels.Labels = append(leftLabels.Labels, shortenDeveloperName(row.Name))

		rightLabels.XYs = append(rightLabels.XYs, plotter.XY{X: p.X.Max, Y: labelY})
		rightLabels.Labels = append(rightLabels.Labels, fmt.Sprintf("%5d %8s %8s", row.Commits, formatNumber(row.LinesAdded-row.LinesRemove), formatNumber(row.LinesChange)))
	}
	rightLabels.XYs = append(rightLabels.XYs, plotter.XY{X: p.X.Max, Y: p.Y.Max * 0.55})
	rightLabels.Labels = append(rightLabels.Labels, " cmts    delta  changed")

	if labels, err := plotter.NewLabels(leftLabels); err == nil {
		p.Add(labels)
	}
	if labels, err := plotter.NewLabels(rightLabels); err == nil {
		p.Add(labels)
	}

	width, height := graphics.GetPythonPlotSize(32, 16)
	if err := graphics.SavePNGWithBackground(p, width, height, output, color.Transparent); err != nil {
		return err
	}

	fmt.Printf("Saved developer plot to %s\n", output)
	return nil
}

func buildDeveloperSeriesRows(timeSeries *readers.DeveloperTimeSeriesData, startUnix, endUnix int64, maxPeople int) ([]devSeriesRow, []time.Time) {
	start, _, size := timeSeriesCalendarRange(timeSeries, startUnix, endUnix, 0)

	dates := make([]time.Time, size)
	for i := range dates {
		dates[i] = start.AddDate(0, 0, i)
	}

	rowsByDev := make(map[int]*devSeriesRow)
	for day, devs := range timeSeries.Days {
		if day < 0 || day >= size {
			continue
		}
		for dev, stats := range devs {
			row := rowsByDev[dev]
			if row == nil {
				name := fmt.Sprintf("Developer %d", dev)
				if dev >= 0 && dev < len(timeSeries.People) {
					name = timeSeries.People[dev]
				}
				row = &devSeriesRow{
					Index:  dev,
					Name:   name,
					Series: make([]float64, size),
				}
				rowsByDev[dev] = row
			}
			row.Series[day] = float64(stats.Commits)
			row.Commits += stats.Commits
			row.LinesAdded += stats.LinesAdded
			row.LinesRemove += stats.LinesRemoved
			row.LinesChange += stats.LinesModified
		}
	}

	rows := make([]devSeriesRow, 0, len(rowsByDev))
	window := oldVsNewSmoothingWindow(max(size/64, 1))
	for _, row := range rowsByDev {
		row.Series = convolveSame(row.Series, window)
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Commits == rows[j].Commits {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Commits > rows[j].Commits
	})
	if maxPeople > 0 && len(rows) > maxPeople {
		rows = rows[:maxPeople]
	}
	return rows, dates
}

func monthlyTicks(start, end time.Time) plot.ConstantTicks {
	var ticks []plot.Tick
	current := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
	if current.Before(start) {
		current = current.AddDate(0, 1, 0)
	}
	for !current.After(end) {
		ticks = append(ticks, plot.Tick{Value: float64(current.Unix()), Label: current.Format("2006-01")})
		current = current.AddDate(0, 1, 0)
	}
	return plot.ConstantTicks(ticks)
}

func areaPolygon(dates []time.Time, values []float64) (*plotter.Polygon, error) {
	if len(dates) != len(values) {
		return nil, fmt.Errorf("date and value length mismatch")
	}
	points := make(plotter.XYs, len(values)*2)
	for i := range values {
		points[i] = plotter.XY{X: float64(dates[i].Unix()), Y: values[i]}
	}
	for i := range values {
		points[len(values)+i] = plotter.XY{X: float64(dates[len(values)-1-i].Unix()), Y: 0}
	}
	return plotter.NewPolygon(points)
}

func shortenDeveloperName(name string) string {
	const maxLen = 36
	if len(name) <= maxLen {
		return name
	}
	return strings.TrimSpace(name[:maxLen]) + "..."
}

func formatNumber(n int) string {
	sign := ""
	value := float64(n)
	if n < 0 {
		sign = "-"
		value = -value
	}
	switch {
	case value >= 1_000_000:
		return fmt.Sprintf("%s%.1fM", sign, value/1_000_000)
	case value >= 10_000:
		return fmt.Sprintf("%s%dK", sign, int(value/1_000))
	case value >= 1_000:
		return fmt.Sprintf("%s%.1fK", sign, value/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func colorToRGBA(c color.Color, alpha uint8) color.RGBA {
	r, g, b, _ := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: alpha}
}

// selectTopDevelopers selects the top developers by commit count.
func selectTopDevelopers(stats []readers.DeveloperStat, maxPeople int) []readers.DeveloperStat {
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Commits > stats[j].Commits
	})
	if len(stats) > maxPeople {
		return stats[:maxPeople]
	}
	return stats
}

// generateTimeSeries generates synthetic time series data for each developer.
func generateTimeSeries(stats []readers.DeveloperStat) map[string][]float64 {
	devSeries := make(map[string][]float64)
	for _, stat := range stats {
		// Generate a synthetic time series based on commit activity
		// In a real implementation, this would come from daily or weekly data
		series := make([]float64, 52) // 52 weeks in a year
		commitsPerWeek := float64(stat.Commits) / 52.0
		for i := 0; i < len(series); i++ {
			// Add random variation to simulate real activity
			series[i] = commitsPerWeek + float64(i%5)*0.1*commitsPerWeek
		}
		devSeries[stat.Name] = series
	}
	return devSeries
}

// generateTimeSeriesWithProgress generates synthetic time series data with progress tracking
func generateTimeSeriesWithProgress(stats []readers.DeveloperStat, progEstimator *progress.ProgressEstimator) map[string][]float64 {
	// Start detailed progress for time series generation
	progEstimator.StartOperation("Generating time series", len(stats))

	devSeries := make(map[string][]float64)
	for _, stat := range stats {
		progEstimator.UpdateProgress(1)

		// Generate a synthetic time series based on commit activity
		// In a real implementation, this would come from daily or weekly data
		series := make([]float64, 52) // 52 weeks in a year
		commitsPerWeek := float64(stat.Commits) / 52.0
		for i := 0; i < len(series); i++ {
			// Add random variation to simulate real activity
			series[i] = commitsPerWeek + float64(i%5)*0.1*commitsPerWeek
		}
		devSeries[stat.Name] = series
	}

	progEstimator.FinishOperation()
	return devSeries
}

// clusterDevelopers clusters developers based on their contribution patterns (placeholder logic).
func clusterDevelopers(devSeries map[string][]float64) map[string]int {
	// Placeholder logic: assign developers to arbitrary clusters
	clusters := make(map[string]int)
	i := 0
	for dev := range devSeries {
		clusters[dev] = i % 3 // Assign developers to 3 clusters
		i++
	}
	return clusters
}

// plotDevs generates plots for developers' contributions.
func plotDevs(developerStats []readers.DeveloperStat, devSeries map[string][]float64, clusters map[string]int, output string) error {
	// Create a new plot
	p := plot.New()
	p.Title.Text = "Developer Contributions Over Time"
	p.X.Label.Text = "Weeks"
	p.Y.Label.Text = "Commits"

	// Plot each developer's time series
	for _, dev := range developerStats {
		series := devSeries[dev.Name]
		pts := make(plotter.XYs, len(series))
		for i, val := range series {
			pts[i].X = float64(i)
			pts[i].Y = val
		}

		line, err := plotter.NewLine(pts)
		if err != nil {
			return fmt.Errorf("error creating plot line for developer %s: %v", dev.Name, err)
		}

		line.Color = graphics.ColorPalette[0] // Use the first color for now
		p.Add(line)
		p.Legend.Add(dev.Name, line)
	}

	// Python labours renders devs at pyplot figure size 32x16 inches.
	width, height := graphics.GetPythonPlotSize(32, 16)
	if err := graphics.SavePNGWithBackground(p, width, height, output, color.Transparent); err != nil {
		return err
	}

	fmt.Printf("Saved developer plot to %s\n", output)
	return nil
}
