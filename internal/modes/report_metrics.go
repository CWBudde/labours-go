package modes

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/readers"
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
		if err := plotIntBars(
			"Bus Factor by Subsystem",
			"Subsystem",
			"Bus Factor",
			labels,
			values,
			siblingOutputPath(output, "bus-factor.png", "subsystems"),
			"bus-factor-subsystems.png",
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
	if err := plotIntBars(
		"Knowledge Diffusion",
		"Unique Editors",
		"Files",
		labels,
		values,
		output,
		"knowledge-diffusion.png",
	); err != nil {
		return err
	}

	if err := plotKnowledgeSilos(data, siblingOutputPath(output, "knowledge-diffusion.png", "silos")); err != nil {
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
	if err := plotFloatBars(
		"Hotspot Risk",
		"Files",
		"Risk Score",
		labels,
		values,
		output,
		"hotspot-risk.png",
	); err != nil {
		return err
	}
	if err := writeHotspotRiskTable(files, siblingOutputPath(output, "hotspot-risk.png", "table.tsv")); err != nil {
		return err
	}
	printHotspotRiskTable(files, 10)
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
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	bars, err := plotter.NewBarChart(values, vg.Points(18))
	if err != nil {
		return fmt.Errorf("failed to create bar chart: %v", err)
	}
	bars.Color = graphics.ColorPalette[0]
	p.Add(bars)
	p.NominalX(labels...)
	if len(labels) > 8 {
		p.X.Tick.Label.Rotation = 0.785398
		p.X.Tick.Label.XAlign = -0.5
		p.X.Tick.Label.YAlign = -0.5
	}

	return saveReportPlot(p, output, defaultOutput)
}

func plotKnowledgeSilos(data *readers.KnowledgeDiffusionData, output string) error {
	files := sortedKnowledgeFiles(data.Files)
	if len(files) == 0 {
		return nil
	}
	if len(files) > 20 {
		files = files[:20]
	}

	labels := make([]string, len(files))
	uniqueValues := make(plotter.Values, len(files))
	recentValues := make(plotter.Values, len(files))
	for i, file := range files {
		labels[i] = compactPathLabel(file.Path)
		uniqueValues[i] = float64(file.UniqueEditors)
		recentValues[i] = float64(file.RecentEditors)
	}

	return plotGroupedBars(
		"Knowledge Silos",
		"Files",
		"Editors",
		labels,
		[]namedValues{
			{Name: "Unique editors", Values: uniqueValues},
			{Name: "Recent editors", Values: recentValues},
		},
		output,
		"knowledge-diffusion-silos.png",
	)
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
			if result[i].RecentEditors == result[j].RecentEditors {
				return result[i].Path < result[j].Path
			}
			return result[i].RecentEditors < result[j].RecentEditors
		}
		return result[i].UniqueEditors < result[j].UniqueEditors
	})
	return result
}

type namedValues struct {
	Name   string
	Values plotter.Values
}

func plotGroupedBars(title, xLabel, yLabel string, labels []string, groups []namedValues, output, defaultOutput string) error {
	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = xLabel
	p.Y.Label.Text = yLabel

	width := vg.Points(12)
	for i, group := range groups {
		bars, err := plotter.NewBarChart(group.Values, width)
		if err != nil {
			return fmt.Errorf("failed to create bar chart: %v", err)
		}
		bars.Color = graphics.ColorPalette[i%len(graphics.ColorPalette)]
		bars.Offset = vg.Points(float64(i)-float64(len(groups)-1)/2) * width
		p.Add(bars)
		p.Legend.Add(group.Name, bars)
	}
	p.NominalX(labels...)
	if len(labels) > 8 {
		p.X.Tick.Label.Rotation = 0.785398
		p.X.Tick.Label.XAlign = -0.5
		p.X.Tick.Label.YAlign = -0.5
	}
	return saveReportPlot(p, output, defaultOutput)
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

	return saveReportPlot(p, output, defaultOutput)
}

func saveReportPlot(p *plot.Plot, output, defaultOutput string) error {
	if output == "" {
		output = defaultOutput
	}
	if dir := filepath.Dir(output); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %v", dir, err)
		}
	}
	width, height := graphics.GetPlotSize(graphics.ChartTypeDefault)
	if err := graphics.SavePlotWithFormat(p, width, height, output); err != nil {
		return err
	}
	fmt.Printf("Saved %s\n", output)
	return nil
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
		labels[i] = compactPathLabel(pair.Key)
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
