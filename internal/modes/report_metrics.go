package modes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/readers"
)

func TemporalActivity(reader readers.Reader, output string, legendThreshold, singleColumnThreshold int) error {
	_ = legendThreshold
	_ = singleColumnThreshold

	temporalReader, ok := reader.(readers.TemporalActivityReader)
	if !ok {
		return fmt.Errorf("reader does not expose temporal activity data")
	}
	data, err := temporalReader.GetTemporalActivity()
	if err != nil {
		return fmt.Errorf("failed to get temporal activity data: %v", err)
	}

	hourlyCommits, hourlyLines := aggregateTemporalHours(data)
	if sumInts(hourlyCommits) == 0 && sumInts(hourlyLines) == 0 {
		return fmt.Errorf("no temporal activity values found")
	}

	fmt.Printf("Temporal activity: %d developers, %d commits, %d changed lines\n",
		len(data.People), sumInts(hourlyCommits), sumInts(hourlyLines))
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
	return plotLineSeries(
		"Bus Factor Over Time",
		"Tick",
		"Bus Factor",
		[]namedSeries{{Name: "Bus factor", Points: series}},
		output,
		"bus-factor.png",
	)
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
	return plotIntBars(
		"Knowledge Diffusion",
		"Unique Editors",
		"Files",
		labels,
		values,
		output,
		"knowledge-diffusion.png",
	)
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
	return plotFloatBars(
		"Hotspot Risk",
		"Files",
		"Risk Score",
		labels,
		values,
		output,
		"hotspot-risk.png",
	)
}

type namedSeries struct {
	Name   string
	Points plotter.XYs
}

func aggregateTemporalHours(data *readers.TemporalActivityData) ([]int, []int) {
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
