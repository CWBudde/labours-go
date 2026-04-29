package modes

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"labours-go/internal/graphics"
	"labours-go/internal/readers"
	"matplotlib-go/backends"
	_ "matplotlib-go/backends/agg"
	_ "matplotlib-go/backends/svg"
	"matplotlib-go/core"
	"matplotlib-go/render"
	"matplotlib-go/style"
)

func TemporalActivity(reader readers.Reader, output string, legendThreshold, singleColumnThreshold int, startTime, endTime *time.Time) error {
	temporalReader, ok := reader.(readers.TemporalActivityReader)
	if !ok {
		return fmt.Errorf("reader does not expose temporal activity data")
	}
	data, err := temporalReader.GetTemporalActivity()
	if err != nil {
		return fmt.Errorf("failed to get temporal activity data: %v", err)
	}

	hourlyCommits, hourlyLines := aggregateTemporalHours(data, reader, startTime, endTime)
	if sumInts(hourlyCommits) == 0 && sumInts(hourlyLines) == 0 {
		return fmt.Errorf("no temporal activity values found")
	}

	legendNote := temporalLegendNote(len(data.People), legendThreshold, singleColumnThreshold)
	fmt.Printf("Temporal activity: %d developers, %d commits, %d changed lines%s\n",
		len(data.People), sumInts(hourlyCommits), sumInts(hourlyLines), legendNote)
	return plotIntBars(
		"Temporal Activity by Hour",
		"Hour of Day",
		"Commits",
		hourLabels(len(hourlyCommits)),
		hourlyCommits,
		output,
		"temporal-activity.png",
	)
}

func BusFactor(reader readers.Reader, output string) error {
	busFactorReader, ok := reader.(readers.BusFactorReader)
	if !ok {
		return fmt.Errorf("reader does not expose bus factor data")
	}
	data, err := busFactorReader.GetBusFactor()
	if err != nil {
		return fmt.Errorf("failed to get bus factor data: %v", err)
	}
	if len(data.Snapshots) == 0 {
		return fmt.Errorf("no bus factor snapshots found")
	}

	ticks := sortedIntKeys(data.Snapshots)
	series := make(plotter.XYs, len(ticks))
	for i, tick := range ticks {
		series[i].X = float64(tick)
		series[i].Y = float64(data.Snapshots[tick].BusFactor)
	}

	latest := data.Snapshots[ticks[len(ticks)-1]]
	fmt.Printf("Bus factor: latest=%d, total lines=%d, threshold=%.2f\n",
		latest.BusFactor, latest.TotalLines, data.Threshold)
	if err := plotLineSeries(
		"Bus Factor Over Time",
		"Tick",
		"Bus Factor",
		[]namedSeries{{Name: "Bus factor", Points: series}},
		output,
		"bus-factor.png",
	); err != nil {
		return err
	}

	if len(data.SubsystemBusFactor) > 0 {
		labels, values := topStringIntPairs(data.SubsystemBusFactor, 20, false)
		if err := plotBusFactorSubsystemsMatplotlib(
			reader.GetName(),
			labels,
			values,
			siblingOutputPath(output, "bus-factor.png", "subsystems"),
		); err != nil {
			return fmt.Errorf("failed to plot subsystem bus factor: %v", err)
		}
		fmt.Printf("Bus factor subsystem summary: %d subsystems\n", len(data.SubsystemBusFactor))
	}
	return nil
}

func OwnershipConcentration(reader readers.Reader, output string) error {
	ownershipReader, ok := reader.(readers.OwnershipConcentrationReader)
	if !ok {
		return fmt.Errorf("reader does not expose ownership concentration data")
	}
	data, err := ownershipReader.GetOwnershipConcentration()
	if err != nil {
		return fmt.Errorf("failed to get ownership concentration data: %v", err)
	}
	if len(data.Snapshots) == 0 {
		return fmt.Errorf("no ownership concentration snapshots found")
	}

	ticks := sortedIntKeys(data.Snapshots)
	gini := make(plotter.XYs, len(ticks))
	hhi := make(plotter.XYs, len(ticks))
	for i, tick := range ticks {
		snapshot := data.Snapshots[tick]
		gini[i].X = float64(tick)
		gini[i].Y = snapshot.Gini
		hhi[i].X = float64(tick)
		hhi[i].Y = snapshot.HHI
	}

	latest := data.Snapshots[ticks[len(ticks)-1]]
	fmt.Printf("Ownership concentration: latest gini=%.3f, hhi=%.3f, total lines=%d\n",
		latest.Gini, latest.HHI, latest.TotalLines)
	return plotLineSeries(
		"Ownership Concentration Over Time",
		"Tick",
		"Concentration",
		[]namedSeries{
			{Name: "Gini", Points: gini},
			{Name: "HHI", Points: hhi},
		},
		output,
		"ownership-concentration.png",
	)
}

func KnowledgeDiffusion(reader readers.Reader, output string) error {
	diffusionReader, ok := reader.(readers.KnowledgeDiffusionReader)
	if !ok {
		return fmt.Errorf("reader does not expose knowledge diffusion data")
	}
	data, err := diffusionReader.GetKnowledgeDiffusion()
	if err != nil {
		return fmt.Errorf("failed to get knowledge diffusion data: %v", err)
	}
	if len(data.Distribution) == 0 && len(data.Files) == 0 {
		return fmt.Errorf("no knowledge diffusion data found")
	}

	labels, values := knowledgeDistribution(data)
	fmt.Printf("Knowledge diffusion: %d files, %d developers, window=%d months\n",
		len(data.Files), len(data.People), data.WindowMonths)
	if err := plotKnowledgeDistribution(labels, values, output); err != nil {
		return err
	}

	if err := plotKnowledgeSilos(reader.GetName(), data, siblingOutputPath(output, "knowledge-diffusion.png", "silos")); err != nil {
		return err
	}
	if err := plotKnowledgeTrend(data, siblingOutputPath(output, "knowledge-diffusion.png", "trend")); err != nil {
		return err
	}
	return nil
}

func HotspotRisk(reader readers.Reader, output string) error {
	hotspotReader, ok := reader.(readers.HotspotRiskReader)
	if !ok {
		return fmt.Errorf("reader does not expose hotspot risk data")
	}
	data, err := hotspotReader.GetHotspotRisk()
	if err != nil {
		return fmt.Errorf("failed to get hotspot risk data: %v", err)
	}
	if len(data.Files) == 0 {
		return fmt.Errorf("no hotspot risk files found")
	}

	files := append([]readers.HotspotRiskFile(nil), data.Files...)
	sort.Slice(files, func(i, j int) bool {
		return files[i].RiskScore > files[j].RiskScore
	})
	if len(files) > 15 {
		files = files[:15]
	}

	labels := make([]string, len(files))
	values := make(plotter.Values, len(files))
	for i, file := range files {
		labels[i] = compactPathLabel(file.Path)
		values[i] = file.RiskScore
	}

	fmt.Printf("Hotspot risk: %d files, window=%d days, top risk=%.3f (%s)\n",
		len(data.Files), data.WindowDays, files[0].RiskScore, files[0].Path)
	if err := plotHotspotRiskRanked(files, labels, values, output); err != nil {
		return err
	}
	if err := writeHotspotRiskTable(files, siblingOutputPath(output, "hotspot-risk.png", "table.tsv")); err != nil {
		return err
	}
	printHotspotRiskTable(files, 10)
	return nil
}

func plotKnowledgeDistribution(labels []string, values []int, output string) error {
	output, err := resolveReportOutput(output, "knowledge-diffusion.png")
	if err != nil {
		return err
	}
	width, height := reportPlotPixels("knowledge-diffusion.png")
	fig := newReportFigure(width, height)
	ax := fig.AddSubplot(1, 1, 1)
	if ax == nil {
		return fmt.Errorf("failed to create knowledge diffusion axes")
	}
	ax.SetTitle("Knowledge Diffusion")
	ax.SetXLabel("Unique Editors")
	ax.SetYLabel("Files")
	ax.AddYGrid()

	y := make([]float64, len(values))
	ticks := make([]float64, len(labels))
	maxValue := 0.0
	for i, value := range values {
		editorCount := float64(i)
		fmt.Sscanf(labels[i], "%f", &editorCount)
		ticks[i] = editorCount
		y[i] = float64(value)
		if y[i] > maxValue {
			maxValue = y[i]
		}
		c := renderColor(knowledgeDistributionColor(int(editorCount)))
		ax.Bar([]float64{editorCount}, []float64{y[i]}, core.BarOptions{Color: &c})
		ax.Text(editorCount, y[i]+0.3, fmt.Sprintf("%d", value), core.TextOptions{
			FontSize: 9,
			HAlign:   core.TextAlignCenter,
			VAlign:   core.TextVAlignBottom,
		})
	}
	minX, maxX := rangeWithPadding(ticks, 0.5)
	ax.SetXLim(minX, maxX)
	ax.SetYLim(0, math.Max(maxValue+1, maxValue*1.15))
	ax.XAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.XAxis.Formatter = core.FixedFormatter{Labels: append([]string(nil), labels...)}

	if err := saveReportFigure(fig, output, width, height); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

type namedSeries struct {
	Name   string
	Points plotter.XYs
}

func aggregateTemporalHours(data *readers.TemporalActivityData, reader readers.Reader, startTime, endTime *time.Time) ([]int, []int) {
	if (startTime != nil || endTime != nil) && len(data.Ticks) > 0 {
		if commits, lines, ok := aggregateTemporalHoursFromTicks(data, reader, startTime, endTime); ok {
			return commits, lines
		}
	}

	commits := make([]int, 24)
	lines := make([]int, 24)

	for _, activity := range data.Activities {
		for hour, value := range activity.Hours.Commits {
			if hour >= 0 && hour < len(commits) {
				commits[hour] += value
			}
		}
		for hour, value := range activity.Hours.Lines {
			if hour >= 0 && hour < len(lines) {
				lines[hour] += value
			}
		}
	}

	if sumInts(commits) > 0 || sumInts(lines) > 0 {
		return commits, lines
	}

	for _, tickDevs := range data.Ticks {
		for _, tick := range tickDevs {
			if tick.Hour >= 0 && tick.Hour < len(commits) {
				commits[tick.Hour] += tick.Commits
				lines[tick.Hour] += tick.Lines
			}
		}
	}
	return commits, lines
}

func aggregateTemporalHoursFromTicks(data *readers.TemporalActivityData, reader readers.Reader, startTime, endTime *time.Time) ([]int, []int, bool) {
	headerStart, headerEnd := reader.GetHeader()
	if headerStart == 0 || data.TickSize <= 0 {
		return nil, nil, false
	}

	filterStart := time.Unix(headerStart, 0)
	filterEnd := time.Unix(headerEnd, 0)
	if startTime != nil {
		filterStart = *startTime
	}
	if endTime != nil {
		filterEnd = *endTime
	}

	tickDuration := time.Duration(data.TickSize)
	if tickDuration <= 0 {
		return nil, nil, false
	}

	commits := make([]int, 24)
	lines := make([]int, 24)
	repoStart := time.Unix(headerStart, 0)
	for tickID, tickDevs := range data.Ticks {
		tickTime := repoStart.Add(time.Duration(tickID) * tickDuration)
		if tickTime.Before(filterStart) || tickTime.After(filterEnd) {
			continue
		}
		for _, tick := range tickDevs {
			if tick.Hour >= 0 && tick.Hour < len(commits) {
				commits[tick.Hour] += tick.Commits
				lines[tick.Hour] += tick.Lines
			}
		}
	}
	fmt.Printf("Filtering temporal activity to %s - %s\n",
		filterStart.Format("2006-01-02"), filterEnd.Format("2006-01-02"))
	return commits, lines, true
}

func temporalLegendNote(developers, legendThreshold, singleColumnThreshold int) string {
	if legendThreshold > 0 && developers > legendThreshold {
		return fmt.Sprintf(" (legend suppressed above %d developers)", legendThreshold)
	}
	if singleColumnThreshold > 0 && developers <= singleColumnThreshold {
		return " (single-column legend eligible)"
	}
	return ""
}

func knowledgeDistribution(data *readers.KnowledgeDiffusionData) ([]string, []int) {
	distribution := make(map[int]int, len(data.Distribution))
	for editors, files := range data.Distribution {
		distribution[editors] = files
	}
	if len(distribution) == 0 {
		for _, file := range data.Files {
			distribution[file.UniqueEditors]++
		}
	}

	keys := make([]int, 0, len(distribution))
	for editors := range distribution {
		keys = append(keys, editors)
	}
	sort.Ints(keys)

	labels := make([]string, len(keys))
	values := make([]int, len(keys))
	for i, editors := range keys {
		labels[i] = fmt.Sprintf("%d", editors)
		values[i] = distribution[editors]
	}
	return labels, values
}

func plotIntBars(title, xLabel, yLabel string, labels []string, values []int, output, defaultOutput string) error {
	plotValues := make(plotter.Values, len(values))
	for i, value := range values {
		plotValues[i] = float64(value)
	}
	return plotFloatBars(title, xLabel, yLabel, labels, plotValues, output, defaultOutput)
}

func plotFloatBars(title, xLabel, yLabel string, labels []string, values plotter.Values, output, defaultOutput string) error {
	output, err := resolveReportOutput(output, defaultOutput)
	if err != nil {
		return err
	}
	plotValues := make([]float64, len(values))
	for i, value := range values {
		plotValues[i] = float64(value)
	}
	width, height := reportPlotInches(defaultOutput)
	if err := graphics.PlotBarChartMatplotlib(labels, plotValues, graphics.MatplotlibBarOptions{
		Title:        title,
		XLabel:       xLabel,
		YLabel:       yLabel,
		Output:       output,
		WidthInches:  width,
		HeightInches: height,
		RotateX:      len(labels) > 8,
	}); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func plotBusFactorSubsystemsMatplotlib(repoName string, labels []string, values []int, output string) error {
	output, err := resolveReportOutput(output, "bus-factor-subsystems.png")
	if err != nil {
		return err
	}
	width, height := reportPlotPixels("bus-factor-subsystems.png")
	fig := newReportFigure(width, height)
	grid := fig.Subplots(1, 1, core.WithSubplotPadding(0.24, 0.945, 0.105, 0.93))
	if len(grid) == 0 || len(grid[0]) == 0 || grid[0][0] == nil {
		return fmt.Errorf("failed to create bus factor subsystem axes")
	}
	ax := grid[0][0]
	if repoName != "" {
		ax.SetTitle(fmt.Sprintf("%s - Bus Factor Subsystems", repoName))
	} else {
		ax.SetTitle("Bus Factor Subsystems")
	}
	ax.SetXLabel("Bus Factor")
	ax.AddXGrid()

	y := make([]float64, len(values))
	barValues := make([]float64, len(values))
	ticks := make([]float64, len(values))
	maxValue := 0.0
	for i, value := range values {
		y[i] = float64(i)
		ticks[i] = float64(i)
		barValues[i] = float64(value)
		maxValue = math.Max(maxValue, barValues[i])
	}
	orientation := core.BarHorizontal
	barHeight := 0.62
	barColor := renderColor(color.RGBA{R: 244, G: 67, B: 54, A: 255})
	bars := ax.Bar(y, barValues, core.BarOptions{
		Color:       &barColor,
		Width:       &barHeight,
		Orientation: &orientation,
	})
	labelText := make([]string, len(values))
	for i, value := range values {
		labelText[i] = fmt.Sprintf("%d", value)
	}
	ax.BarLabel(bars, labelText, core.BarLabelOptions{
		Padding:  6,
		FontSize: 10,
	})

	limitColor := renderColor(color.RGBA{R: 244, G: 67, B: 54, A: 255})
	lineWidth := 2.0
	ax.AxVLine(1, core.VLineOptions{
		Color:     &limitColor,
		LineWidth: &lineWidth,
		Dashes:    []float64{6, 4},
	})
	ax.SetXLim(0, math.Max(maxValue*1.05, 1.05))
	ax.SetYLim(-0.5, float64(len(labels))-0.5)
	ax.InvertY()
	ax.YAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.YAxis.Formatter = core.FixedFormatter{Labels: append([]string(nil), labels...)}

	if err := saveReportFigure(fig, output, width, height); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func plotKnowledgeSilos(repoName string, data *readers.KnowledgeDiffusionData, output string) error {
	files := sortedKnowledgeFiles(data.Files)
	if len(files) == 0 {
		return nil
	}
	if len(files) > 30 {
		files = files[:30]
	}

	labels := make([]string, len(files))
	uniqueValues := make(plotter.Values, len(files))
	recentValues := make(plotter.Values, len(files))
	for i, file := range files {
		labels[i] = truncateKnowledgeSiloLabel(file.Path)
		uniqueValues[i] = float64(file.UniqueEditors)
		recentValues[i] = float64(file.RecentEditors)
	}

	return plotKnowledgeSilosMatplotlib(repoName, labels, uniqueValues, recentValues, data.WindowMonths, output)
}

func plotKnowledgeSilosMatplotlib(repoName string, labels []string, uniqueValues, recentValues plotter.Values, windowMonths int, output string) error {
	output, err := resolveReportOutput(output, "knowledge-diffusion-silos.png")
	if err != nil {
		return err
	}
	heightInches := math.Max(5, float64(len(labels))*0.35+2)
	width, height := int(14*100), int(heightInches*100)
	fig := newKnowledgeSilosFigure(width, height)
	grid := fig.Subplots(1, 1, core.WithSubplotPadding(0.332, 0.947, 0.05, 0.97))
	if len(grid) == 0 || len(grid[0]) == 0 || grid[0][0] == nil {
		return fmt.Errorf("failed to create knowledge silos axes")
	}
	ax := grid[0][0]
	title := "Knowledge Silos"
	if repoName != "" {
		title = fmt.Sprintf("%s - Knowledge Silos", repoName)
	}
	ax.SetTitle(title)
	ax.SetXLabel("Number of Editors")
	ax.AddXGrid()

	yTotal := make([]float64, len(labels))
	yRecent := make([]float64, len(labels))
	total := make([]float64, len(labels))
	recent := make([]float64, len(labels))
	ticks := make([]float64, len(labels))
	maxValue := 0.0
	for i := range labels {
		yTotal[i] = float64(i) - 0.18
		yRecent[i] = float64(i) + 0.18
		ticks[i] = float64(i)
		total[i] = float64(uniqueValues[i])
		recent[i] = float64(recentValues[i])
		maxValue = math.Max(maxValue, math.Max(total[i], recent[i]))
	}
	orientation := core.BarHorizontal
	barHeight := 0.35
	totalColor := renderColor(color.RGBA{R: 144, G: 202, B: 249, A: 255})
	recentColor := renderColor(color.RGBA{R: 21, G: 101, B: 192, A: 255})
	ax.Bar(yTotal, total, core.BarOptions{
		Color:       &totalColor,
		Width:       &barHeight,
		Orientation: &orientation,
		Label:       "Total unique editors",
	})
	ax.Bar(yRecent, recent, core.BarOptions{
		Color:       &recentColor,
		Width:       &barHeight,
		Orientation: &orientation,
		Label:       fmt.Sprintf("Active in last %d months", windowMonths),
	})
	clipOff := false
	labelColor := render.Color{R: 0, G: 0, B: 0, A: 1}
	for i := range labels {
		ax.Text(total[i]+0.1, yTotal[i], fmt.Sprintf("%.0f", total[i]), core.TextOptions{
			FontSize: 8.4,
			Color:    labelColor,
			VAlign:   core.TextVAlignMiddle,
			ClipOn:   &clipOff,
		})
		ax.Text(recent[i]+0.1, yRecent[i], fmt.Sprintf("%.0f", recent[i]), core.TextOptions{
			FontSize: 8.4,
			Color:    labelColor,
			VAlign:   core.TextVAlignMiddle,
			ClipOn:   &clipOff,
		})
	}
	ax.SetXLim(0, math.Max(maxValue*1.05, 1.05))
	ax.SetYLim(-1.85, float64(len(labels))+0.85)
	ax.InvertY()
	ax.YAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.YAxis.Formatter = core.FixedFormatter{Labels: append([]string(nil), labels...)}
	yLabelStyle := ax.YAxis.MajorLabelStyle
	yLabelStyle.FontKey = "DejaVu Sans Mono"
	ax.YAxis.MajorLabelStyle = yLabelStyle
	legend := ax.AddLegend()
	legend.Location = core.LegendLowerRight
	legend.FontSize = 9.6
	legend.Padding = 5
	legend.RowGap = 2
	legend.BackgroundColor = render.Color{R: 0.9, G: 0.9, B: 0.9, A: 0.8}
	legend.BorderColor = render.Color{R: 0.8, G: 0.8, B: 0.8, A: 0.8}
	legend.TextColor = render.Color{R: 0, G: 0, B: 0, A: 1}

	if err := saveReportFigureWithoutTightLayout(fig, output, width, height); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func plotKnowledgeTrend(data *readers.KnowledgeDiffusionData, output string) error {
	trend := make(map[int]int)
	for _, file := range data.Files {
		for tick, editors := range file.UniqueEditorsOverTime {
			if editors > trend[tick] {
				trend[tick] = editors
			}
		}
	}
	if len(trend) == 0 {
		return nil
	}

	ticks := sortedIntKeys(trend)
	points := make(plotter.XYs, len(ticks))
	for i, tick := range ticks {
		points[i].X = float64(tick)
		points[i].Y = float64(trend[tick])
	}
	return plotLineSeries(
		"Knowledge Diffusion Trend",
		"Tick",
		"Max Unique Editors",
		[]namedSeries{{Name: "Max editors", Points: points}},
		output,
		"knowledge-diffusion-trend.png",
	)
}

type knowledgeFileSummary struct {
	Path          string
	UniqueEditors int
	RecentEditors int
}

func sortedKnowledgeFiles(files map[string]readers.KnowledgeDiffusionFile) []knowledgeFileSummary {
	result := make([]knowledgeFileSummary, 0, len(files))
	for path, file := range files {
		result = append(result, knowledgeFileSummary{
			Path:          path,
			UniqueEditors: file.UniqueEditors,
			RecentEditors: file.RecentEditors,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UniqueEditors == result[j].UniqueEditors {
			return result[i].Path < result[j].Path
		}
		return result[i].UniqueEditors < result[j].UniqueEditors
	})
	return result
}

func truncateKnowledgeSiloLabel(path string) string {
	if len(path) > 60 {
		return "..." + path[len(path)-57:]
	}
	return path
}

type namedValues struct {
	Name   string
	Values plotter.Values
}

func plotGroupedBars(title, xLabel, yLabel string, labels []string, groups []namedValues, output, defaultOutput string) error {
	output, err := resolveReportOutput(output, defaultOutput)
	if err != nil {
		return err
	}
	series := make([]graphics.MatplotlibGroupedBarSeries, len(groups))
	for i, group := range groups {
		values := make([]float64, len(group.Values))
		for j, value := range group.Values {
			values[j] = float64(value)
		}
		series[i] = graphics.MatplotlibGroupedBarSeries{Name: group.Name, Values: values}
	}
	width, height := reportPlotInches(defaultOutput)
	if err := graphics.PlotGroupedBarChartMatplotlib(labels, series, graphics.MatplotlibGroupedBarOptions{
		Title:        title,
		XLabel:       xLabel,
		YLabel:       yLabel,
		Output:       output,
		WidthInches:  width,
		HeightInches: height,
		RotateX:      len(labels) > 8,
	}); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func writeHotspotRiskTable(files []readers.HotspotRiskFile, output string) error {
	var buffer bytes.Buffer
	buffer.WriteString("rank\trisk_score\tsize\tchurn\tcoupling_degree\townership_gini\tfile\n")
	for i, file := range files {
		fmt.Fprintf(&buffer, "%d\t%.6f\t%d\t%d\t%d\t%.6f\t%s\n",
			i+1, file.RiskScore, file.Size, file.Churn, file.CouplingDegree, file.OwnershipGini, file.Path)
	}
	if output == "" {
		output = "hotspot-risk-table.tsv"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil && filepath.Dir(output) != "." {
		return fmt.Errorf("failed to create output directory: %v", err)
	}
	if err := os.WriteFile(output, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write hotspot risk table: %v", err)
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func plotHotspotRiskRanked(files []readers.HotspotRiskFile, labels []string, values plotter.Values, output string) error {
	output, err := resolveReportOutput(output, "hotspot-risk.png")
	if err != nil {
		return err
	}
	width, height := reportPlotPixels("hotspot-risk.png")
	fig := newReportFigure(width, height)
	grid := fig.Subplots(1, 2, core.WithSubplotPadding(0.18, 0.95, 0.14, 0.88), core.WithSubplotSpacing(0.22, 0.1))
	if len(grid) == 0 || len(grid[0]) < 2 {
		return fmt.Errorf("failed to create hotspot risk axes")
	}
	axRisk := grid[0][0]
	axComponents := grid[0][1]

	y := make([]float64, len(files))
	risk := make([]float64, len(files))
	sizeNorm := make([]float64, len(files))
	churnNorm := make([]float64, len(files))
	couplingNorm := make([]float64, len(files))
	ownershipNorm := make([]float64, len(files))
	ticks := make([]float64, len(files))
	displayNames := make([]string, len(files))
	maxRisk := 0.0
	maxSize, maxChurn, maxCoupling := hotspotMaxima(files)
	for i, file := range files {
		y[i] = float64(i)
		ticks[i] = float64(i)
		risk[i] = float64(values[i])
		maxRisk = math.Max(maxRisk, risk[i])
		displayNames[i] = hotspotDisplayName(labels[i], file.Path)
		sizeNorm[i] = math.Log(float64(file.Size)+1) / math.Log(maxSize+1)
		churnNorm[i] = float64(file.Churn) / maxChurn
		couplingNorm[i] = float64(file.CouplingDegree) / maxCoupling
		ownershipNorm[i] = file.OwnershipGini
	}

	orientation := core.BarHorizontal
	barHeight := 0.8
	riskBars := axRisk.Bar(y, risk, core.BarOptions{
		Width:       &barHeight,
		Orientation: &orientation,
	})
	riskBars.Colors = hotspotRiskColors(risk, maxRisk)
	riskBars.EdgeColor = render.Color{R: 0, G: 0, B: 0, A: 1}
	riskBars.EdgeWidth = 0.5
	axRisk.SetTitle("Top Risky Files")
	axRisk.SetXLabel("Composite Risk Score")
	axRisk.SetXLim(0, math.Max(maxRisk*1.1, 1))
	axRisk.SetYLim(-0.5, float64(len(files))-0.5)
	axRisk.InvertY()
	axRisk.AddXGrid()
	axRisk.YAxis.Locator = core.FixedLocator{TicksList: ticks}
	axRisk.YAxis.Formatter = core.FixedFormatter{Labels: displayNames}

	addHotspotComponentBars(axComponents, y, sizeNorm, nil, "#3498db", "Size (log)")
	left := append([]float64(nil), sizeNorm...)
	addHotspotComponentBars(axComponents, y, churnNorm, left, "#e74c3c", "Churn")
	for i := range left {
		left[i] += churnNorm[i]
	}
	addHotspotComponentBars(axComponents, y, couplingNorm, left, "#f39c12", "Coupling")
	for i := range left {
		left[i] += couplingNorm[i]
	}
	addHotspotComponentBars(axComponents, y, ownershipNorm, left, "#9b59b6", "Ownership")
	axComponents.SetTitle("Risk Components")
	axComponents.SetXLabel("Normalized Factors")
	axComponents.SetXLim(0, 4)
	axComponents.SetYLim(-0.5, float64(len(files))-0.5)
	axComponents.InvertY()
	axComponents.YAxis.Locator = core.FixedLocator{TicksList: ticks}
	axComponents.YAxis.Formatter = core.NullFormatter{}
	axComponents.YAxis.ShowTicks = false
	axComponents.AddLegend()

	if err := saveReportFigure(fig, output, width, height); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func addHotspotComponentBars(ax *core.Axes, y, values, left []float64, hex, label string) {
	orientation := core.BarHorizontal
	barHeight := 0.8
	alpha := 0.8
	c := renderColor(mustHexColor(hex))
	ax.Bar(y, values, core.BarOptions{
		Color:       &c,
		Width:       &barHeight,
		Baselines:   left,
		Orientation: &orientation,
		Alpha:       &alpha,
		Label:       label,
	})
}

func hotspotMaxima(files []readers.HotspotRiskFile) (float64, float64, float64) {
	maxSize, maxChurn, maxCoupling := 1.0, 1.0, 1.0
	for _, file := range files {
		maxSize = math.Max(maxSize, float64(file.Size))
		maxChurn = math.Max(maxChurn, float64(file.Churn))
		maxCoupling = math.Max(maxCoupling, float64(file.CouplingDegree))
	}
	return maxSize, maxChurn, maxCoupling
}

func hotspotDisplayName(label, path string) string {
	if strings.Contains(path, "/") {
		parts := strings.Split(path, "/")
		if len(parts) > 3 {
			return ".../" + strings.Join(parts[len(parts)-2:], "/")
		}
		return path
	}
	return label
}

func hotspotRiskColors(values []float64, maxValue float64) []render.Color {
	if maxValue <= 0 {
		maxValue = 1
	}
	colors := make([]render.Color, len(values))
	for i, value := range values {
		colors[i] = renderColor(riskGradient(value / maxValue))
	}
	return colors
}

func riskGradient(ratio float64) color.Color {
	ratio = math.Max(0, math.Min(1, ratio))
	if ratio < 0.5 {
		t := ratio * 2
		return interpolateColor(
			color.RGBA{R: 26, G: 152, B: 80, A: 255},
			color.RGBA{R: 255, G: 255, B: 191, A: 255},
			t,
		)
	}
	return interpolateColor(
		color.RGBA{R: 255, G: 255, B: 191, A: 255},
		color.RGBA{R: 215, G: 48, B: 39, A: 255},
		(ratio-0.5)*2,
	)
}

func printHotspotRiskTable(files []readers.HotspotRiskFile, limit int) {
	if len(files) < limit {
		limit = len(files)
	}
	fmt.Printf("\nTop %d High-Risk Files\n", limit)
	fmt.Printf("%-5s %8s %6s %6s %9s %6s  %s\n", "Rank", "Risk", "Size", "Churn", "Coupling", "Gini", "File")
	for i := 0; i < limit; i++ {
		file := files[i]
		fmt.Printf("%-5d %8.4f %6d %6d %9d %6.3f  %s\n",
			i+1, file.RiskScore, file.Size, file.Churn, file.CouplingDegree, file.OwnershipGini, file.Path)
	}
}

func plotLineSeries(title, xLabel, yLabel string, series []namedSeries, output, defaultOutput string) error {
	if defaultOutput == "knowledge-diffusion-trend.png" {
		return plotLineSeriesGonum(title, xLabel, yLabel, series, output, defaultOutput)
	}

	output, err := resolveReportOutput(output, defaultOutput)
	if err != nil {
		return err
	}
	plotSeries := make([]graphics.MatplotlibLineSeries, len(series))
	for i, item := range series {
		x := make([]float64, len(item.Points))
		y := make([]float64, len(item.Points))
		for j, point := range item.Points {
			x[j] = point.X
			y[j] = point.Y
		}
		plotSeries[i] = graphics.MatplotlibLineSeries{Name: item.Name, X: x, Y: y, Marker: true}
	}
	width, height := reportPlotInches(defaultOutput)
	if err := graphics.PlotLineChartMatplotlib(plotSeries, graphics.MatplotlibLineOptions{
		Title:        title,
		XLabel:       xLabel,
		YLabel:       yLabel,
		Output:       output,
		WidthInches:  width,
		HeightInches: height,
		ShowGrid:     true,
		Legend:       true,
	}); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func plotLineSeriesGonum(title, xLabel, yLabel string, series []namedSeries, output, defaultOutput string) error {
	output, err := resolveReportOutput(output, defaultOutput)
	if err != nil {
		return err
	}

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	for i, item := range series {
		line, err := plotter.NewLine(item.Points)
		if err != nil {
			return fmt.Errorf("failed to create line series %s: %v", item.Name, err)
		}
		line.Color = graphics.ColorPalette[i%len(graphics.ColorPalette)]
		p.Add(line)
		p.Legend.Add(item.Name, line)
	}

	width, height := graphics.GetPlotSize(graphics.ChartTypeDefault)
	if err := graphics.SavePlotWithFormat(p, width, height, output); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
}

func resolveReportOutput(output, defaultOutput string) (string, error) {
	if output == "" {
		output = defaultOutput
	}
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create output directory %s: %v", dir, err)
		}
	}
	return output, nil
}

func reportPlotInches(defaultOutput string) (float64, float64) {
	switch defaultOutput {
	case "temporal-activity.png":
		return 16, 10
	case "bus-factor.png", "ownership-concentration.png":
		return 14, 6
	case "bus-factor-subsystems.png", "knowledge-diffusion.png":
		return 12, 6
	case "knowledge-diffusion-silos.png":
		return 14, 12.5
	case "hotspot-risk.png":
		return 12, 8
	default:
		return 16, 8
	}
}

func reportPlotPixels(defaultOutput string) (int, int) {
	width, height := reportPlotInches(defaultOutput)
	return int(width * 100), int(height * 100)
}

func newReportFigure(width, height int) *core.Figure {
	background := render.Color{R: 1, G: 1, B: 1, A: 0}
	text := render.Color{R: 0, G: 0, B: 0, A: 1}
	return core.NewFigure(
		width,
		height,
		style.WithTheme(style.ThemeGGPlot),
		style.WithFont("DejaVu Sans", 12),
		style.WithTextColor(0, 0, 0, 1),
		style.WithBackground(1, 1, 1, 0),
		style.WithAxesBackground(background),
		style.WithAxesEdgeColor(text),
		style.WithLegendColors(render.Color{R: 1, G: 1, B: 1, A: 0.8}, background, text),
	)
}

func newKnowledgeSilosFigure(width, height int) *core.Figure {
	background := render.Color{R: 1, G: 1, B: 1, A: 0}
	text := render.Color{R: 0, G: 0, B: 0, A: 1}
	return core.NewFigure(
		width,
		height,
		style.WithTheme(style.ThemeGGPlot),
		style.WithFont("DejaVu Sans", 12),
		style.WithTextColor(0, 0, 0, 1),
		style.WithBackground(1, 1, 1, 0),
		style.WithAxesBackground(background),
		style.WithAxesEdgeColor(text),
		style.WithLegendColors(render.Color{R: 1, G: 1, B: 1, A: 0.8}, background, text),
		func(rc *style.RC) {
			rc.YTickLabelFontSize = 12
			rc.LegendFontSize = 9.6
		},
	)
}

func saveReportFigure(fig *core.Figure, output string, width, height int) error {
	fig.TightLayout()
	return saveReportFigureDirect(fig, output, width, height)
}

func saveReportFigureWithoutTightLayout(fig *core.Figure, output string, width, height int) error {
	return saveReportFigureDirect(fig, output, width, height)
}

func saveReportFigureDirect(fig *core.Figure, output string, width, height int) error {
	config := backends.Config{
		Width:      width,
		Height:     height,
		Background: render.Color{R: 1, G: 1, B: 1, A: 0},
		DPI:        100,
	}
	switch strings.ToLower(filepath.Ext(output)) {
	case ".svg":
		renderer, _, err := backends.NewRenderer("svg", config, nil)
		if err != nil {
			return fmt.Errorf("failed to create SVG renderer: %v", err)
		}
		return core.SaveSVG(fig, renderer, output)
	default:
		renderer, _, err := backends.NewRenderer("agg", config, backends.TextCapabilities)
		if err != nil {
			return fmt.Errorf("failed to create AGG renderer: %v", err)
		}
		return core.SavePNG(fig, renderer, output)
	}
}

func knowledgeDistributionColor(editors int) color.Color {
	switch {
	case editors <= 1:
		return mustHexColor("#F44336")
	case editors <= 2:
		return mustHexColor("#FF9800")
	case editors <= 3:
		return mustHexColor("#FFC107")
	default:
		return mustHexColor("#4CAF50")
	}
}

func rangeWithPadding(values []float64, padding float64) (float64, float64) {
	if len(values) == 0 {
		return -padding, padding
	}
	minValue, maxValue := values[0], values[0]
	for _, value := range values[1:] {
		minValue = math.Min(minValue, value)
		maxValue = math.Max(maxValue, value)
	}
	return minValue - padding, maxValue + padding
}

func renderColor(c color.Color) render.Color {
	r, g, b, a := c.RGBA()
	return render.Color{
		R: float64(r) / 65535,
		G: float64(g) / 65535,
		B: float64(b) / 65535,
		A: float64(a) / 65535,
	}
}

func mustHexColor(hex string) color.RGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.RGBA{A: 255}
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b); err != nil {
		return color.RGBA{A: 255}
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

func interpolateColor(a, b color.RGBA, t float64) color.RGBA {
	t = math.Max(0, math.Min(1, t))
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

func sortedIntKeys[T any](values map[int]T) []int {
	keys := make([]int, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	return keys
}

func topStringIntPairs(values map[string]int, limit int, descending bool) ([]string, []int) {
	type pair struct {
		Key   string
		Value int
	}
	pairs := make([]pair, 0, len(values))
	for key, value := range values {
		pairs = append(pairs, pair{Key: key, Value: value})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Value == pairs[j].Value {
			return pairs[i].Key < pairs[j].Key
		}
		if descending {
			return pairs[i].Value > pairs[j].Value
		}
		return pairs[i].Value < pairs[j].Value
	})
	if limit > 0 && len(pairs) > limit {
		pairs = pairs[:limit]
	}
	labels := make([]string, len(pairs))
	resultValues := make([]int, len(pairs))
	for i, pair := range pairs {
		labels[i] = pair.Key
		resultValues[i] = pair.Value
	}
	return labels, resultValues
}

func siblingOutputPath(output, defaultOutput, suffix string) string {
	if output == "" {
		output = defaultOutput
	}
	ext := filepath.Ext(output)
	if ext == "" {
		ext = ".png"
	}
	base := output[:len(output)-len(filepath.Ext(output))]
	if filepath.Ext(suffix) != "" {
		return base + "_" + suffix
	}
	return base + "_" + suffix + ext
}

func hourLabels(hours int) []string {
	labels := make([]string, hours)
	for hour := range labels {
		labels[hour] = fmt.Sprintf("%02d", hour)
	}
	return labels
}

func sumInts(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func compactPathLabel(path string) string {
	base := filepath.Base(path)
	if len(base) <= 24 {
		return base
	}
	return base[:21] + "..."
}
