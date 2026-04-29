package modes

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"labours-go/internal/graphics"
	"labours-go/internal/progress"
	"labours-go/internal/readers"
	"matplotlib-go/backends"
	_ "matplotlib-go/backends/agg"
	_ "matplotlib-go/backends/svg"
	"matplotlib-go/core"
	"matplotlib-go/render"
	"matplotlib-go/style"
)

func OwnershipBurndown(reader readers.Reader, output string) error {
	// Initialize progress tracking
	quiet := viper.GetBool("quiet")
	progEstimator := progress.NewProgressEstimator(!quiet)

	// Start multi-phase operation for ownership analysis
	totalPhases := 4 // validation, data extraction, processing, visualization
	progEstimator.StartMultiOperation(totalPhases, "Ownership Burndown Analysis")

	// Phase 1: Validate output path
	progEstimator.NextOperation("Validating output path")
	if output == "" {
		output = "ownership.png"
		if !quiet {
			fmt.Printf("Output not provided, using default: %s\n", output)
		}
	}

	outputDir := filepath.Dir(output)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to create output directory %s: %v", outputDir, err)
	}

	// Phase 2: Extract data from the reader
	progEstimator.NextOperation("Extracting ownership data")
	peopleSequence, ownershipData, err := reader.GetOwnershipBurndown()
	if err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to get ownership burndown data: %v", err)
	}
	if len(peopleSequence) == 0 {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("no ownership burndown data found")
	}

	params, err := reader.GetBurndownParameters()
	if err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to get burndown parameters: %v", err)
	}
	startUnix, lastUnix := reader.GetHeader()
	tickSize := params.TickSize
	if tickSize <= 0 {
		tickSize = 24 * 60 * 60
	}
	sampling := params.Sampling
	if sampling <= 0 {
		sampling = 1
	}
	startTime := floorTimeBySeconds(time.Unix(startUnix, 0), tickSize).Add(secondsDuration(float64(sampling) * tickSize))
	lastTime := time.Unix(lastUnix, 0)
	if lastUnix == 0 {
		lastTime = startTime
	}

	// Phase 3: Process the data
	progEstimator.NextOperation("Processing ownership data")
	maxPeople := viper.GetInt("max-people")
	if maxPeople <= 0 {
		maxPeople = 20
	}
	orderByTime := viper.GetBool("order-ownership-by-time")
	names, peopleMatrix, dateRange := processOwnershipBurndownWithProgress(
		startTime, lastTime, sampling, tickSize, peopleSequence, ownershipData, maxPeople, orderByTime, progEstimator)

	// Phase 4: Generate output
	progEstimator.NextOperation("Generating visualization")

	// Check if JSON output is required
	if filepath.Ext(output) == ".json" {
		progEstimator.FinishMultiOperation()
		return saveOwnershipBurndownAsJSON(output, names, peopleMatrix, dateRange, lastTime)
	}

	// Visualize the data
	if err := plotOwnershipBurndown(names, peopleMatrix, dateRange, lastTime, output); err != nil {
		progEstimator.FinishMultiOperation()
		return fmt.Errorf("failed to plot ownership burndown: %v", err)
	}

	progEstimator.FinishMultiOperation()
	if !quiet {
		fmt.Println("Ownership burndown chart generated successfully.")
	}
	return nil
}

func processOwnershipBurndown(
	start, last time.Time, sampling int, tickSize float64,
	sequence []string, data map[string][][]int,
	maxPeople int, orderByTime bool,
) ([]string, [][]float64, []time.Time) {
	// Aggregate the ownership data
	people := make([][]float64, len(sequence))
	for i, name := range sequence {
		rows := data[name]
		if len(rows) == 0 {
			continue
		}
		total := make([]float64, len(rows))
		for rowIndex, row := range rows {
			for _, val := range row {
				total[rowIndex] += float64(val)
			}
		}
		people[i] = total
	}
	pointCount := ownershipPointCount(people)
	if pointCount == 0 {
		return sequence, people, nil
	}

	// Create a date range based on sampling
	dateRange := make([]time.Time, pointCount)
	step := ownershipSamplingDuration(sampling, tickSize)
	for i := 0; i < len(dateRange); i++ {
		dateRange[i] = start.Add(time.Duration(i) * step)
	}

	// Truncate to maxPeople
	if len(people) > maxPeople {
		sums := make([]float64, len(people))
		for i, row := range people {
			for _, val := range row {
				sums[i] += val
			}
		}

		indices := argsortDescending(sums)
		chosen := indices[:maxPeople]
		others := indices[maxPeople:]

		// Aggregate "others"
		othersTotal := make([]float64, len(people[0]))
		for _, idx := range others {
			for j, val := range people[idx] {
				othersTotal[j] += val
			}
		}

		// Update people and sequence
		truncatedPeople := make([][]float64, maxPeople+1)
		truncatedNames := make([]string, maxPeople+1)
		for i, idx := range chosen {
			truncatedPeople[i] = people[idx]
			truncatedNames[i] = sequence[idx]
		}
		truncatedPeople[maxPeople] = othersTotal
		truncatedNames[maxPeople] = "others"

		people = truncatedPeople
		sequence = truncatedNames
	}

	// Sort by first appearance or total ownership
	if orderByTime {
		appearances := make([]int, len(people))
		for i, row := range people {
			appearances[i] = findFirstNonZero(row)
		}
		indices := argsortAscending(appearances)
		people = reorder(people, indices)
		sequence = reorderStrings(sequence, indices)
	} else {
		totalOwnership := make([]float64, len(people))
		for i, row := range people {
			for _, val := range row {
				totalOwnership[i] += val
			}
		}
		indices := argsortDescending(totalOwnership)
		people = reorder(people, indices)
		sequence = reorderStrings(sequence, indices)
	}

	return sequence, people, dateRange
}

// processOwnershipBurndownWithProgress processes ownership data with progress tracking
func processOwnershipBurndownWithProgress(
	start, last time.Time, sampling int, tickSize float64,
	sequence []string, data map[string][][]int,
	maxPeople int, orderByTime bool,
	progEstimator *progress.ProgressEstimator,
) ([]string, [][]float64, []time.Time) {
	// Start detailed progress for data processing
	totalSteps := len(sequence) + 2 // aggregation steps + sorting + date range creation
	progEstimator.StartOperation("Aggregating ownership data", totalSteps)

	// Aggregate the ownership data
	people := make([][]float64, len(sequence))
	for i, name := range sequence {
		progEstimator.UpdateProgress(1)
		rows := data[name]
		if len(rows) == 0 {
			continue
		}
		total := make([]float64, len(rows))
		for rowIndex, row := range rows {
			for _, val := range row {
				total[rowIndex] += float64(val)
			}
		}
		people[i] = total
	}
	pointCount := ownershipPointCount(people)
	if pointCount == 0 {
		progEstimator.FinishOperation()
		return sequence, people, nil
	}

	// Create a date range based on sampling
	progEstimator.UpdateProgress(1)
	dateRange := make([]time.Time, pointCount)
	step := ownershipSamplingDuration(sampling, tickSize)
	for i := 0; i < len(dateRange); i++ {
		dateRange[i] = start.Add(time.Duration(i) * step)
	}

	// Truncate to maxPeople
	if len(people) > maxPeople {
		sums := make([]float64, len(people))
		for i, row := range people {
			for _, val := range row {
				sums[i] += val
			}
		}

		indices := argsortDescending(sums)
		chosen := indices[:maxPeople]
		others := indices[maxPeople:]

		// Aggregate "others"
		othersTotal := make([]float64, len(people[0]))
		for _, idx := range others {
			for j, val := range people[idx] {
				othersTotal[j] += val
			}
		}

		// Update people and sequence
		truncatedPeople := make([][]float64, maxPeople+1)
		truncatedNames := make([]string, maxPeople+1)
		for i, idx := range chosen {
			truncatedPeople[i] = people[idx]
			truncatedNames[i] = sequence[idx]
		}
		truncatedPeople[maxPeople] = othersTotal
		truncatedNames[maxPeople] = "others"

		people = truncatedPeople
		sequence = truncatedNames
	}

	// Sort by first appearance or total ownership
	progEstimator.UpdateProgress(1)
	if orderByTime {
		appearances := make([]int, len(people))
		for i, row := range people {
			appearances[i] = findFirstNonZero(row)
		}
		indices := argsortAscending(appearances)
		people = reorder(people, indices)
		sequence = reorderStrings(sequence, indices)
	} else {
		totalOwnership := make([]float64, len(people))
		for i, row := range people {
			for _, val := range row {
				totalOwnership[i] += val
			}
		}
		indices := argsortDescending(totalOwnership)
		people = reorder(people, indices)
		sequence = reorderStrings(sequence, indices)
	}

	progEstimator.FinishOperation()
	return sequence, people, dateRange
}

func plotOwnershipBurndown(names []string, people [][]float64, dateRange []time.Time, lastTime time.Time, output string) error {
	if len(people) == 0 || len(dateRange) == 0 {
		return fmt.Errorf("no ownership burndown data to plot")
	}
	if lastTime.Before(dateRange[len(dateRange)-1]) {
		lastTime = dateRange[len(dateRange)-1]
	}
	x := make([]float64, len(dateRange))
	for i, date := range dateRange {
		x[i] = float64(date.Unix())
	}
	matrix := make([][]float64, 0, len(people))
	labels := make([]string, 0, len(people))
	for i, row := range people {
		if len(row) == 0 {
			continue
		}
		values := make([]float64, len(dateRange))
		for j := range values {
			if j < len(row) {
				values[j] = row[j]
			}
		}
		label := fmt.Sprintf("Developer %d", i+1)
		if i < len(names) {
			label = names[i]
		}
		label = truncateOwnershipLabel(label)
		matrix = append(matrix, values)
		labels = append(labels, label)
	}
	if len(matrix) == 0 {
		return fmt.Errorf("no ownership burndown data to plot")
	}
	if viper.GetBool("relative") {
		normalizeOwnershipColumns(matrix)
	}

	width, height := ownershipPlotPixelSize(16, 12)
	fontSize := viper.GetInt("font-size")
	if fontSize <= 0 {
		fontSize = 12
	}
	background, foreground := ownershipPlotColors(viper.GetString("background"))
	fig := core.NewFigure(
		width,
		height,
		style.WithTheme(style.ThemeGGPlot),
		style.WithFont("DejaVu Sans", float64(fontSize)),
		style.WithBackground(background.R, background.G, background.B, background.A),
		style.WithAxesBackground(background),
		style.WithAxesEdgeColor(foreground),
		style.WithTextColor(foreground.R, foreground.G, foreground.B, foreground.A),
		style.WithLegendColors(
			render.Color{R: background.R, G: background.G, B: background.B, A: 1},
			background,
			foreground,
		),
	)
	ax := fig.AddSubplot(1, 1, 1)
	if ax == nil {
		return fmt.Errorf("failed to create ownership axes")
	}
	ax.SetTitle("Ownership Burndown")
	ax.SetXLabel("Time")
	ax.SetYLabel("Ownership")
	colors := graphics.PythonLaboursColorPalette(len(matrix))
	renderColors := make([]render.Color, len(colors))
	for i, color := range colors {
		renderColors[i] = ownershipRenderColor(color)
	}
	ax.StackPlot(x, matrix, core.StackPlotOptions{
		Colors: renderColors,
		Labels: labels,
	})
	xMin := float64(dateRange[0].Unix())
	xMax := float64(lastTime.Unix())
	if xMin == xMax {
		xMin = float64(dateRange[0].AddDate(-2, 0, 0).Unix())
		xMax = float64(dateRange[0].AddDate(2, 0, 0).Unix())
	}
	ax.SetXLim(xMin, xMax)
	if viper.GetBool("relative") {
		ax.SetYLim(0, 1)
	} else {
		ax.SetYLim(0, math.Max(maxOwnershipStackY(matrix)*1.05, 1))
	}
	configureOwnershipTimeAxis(ax, dateRange)
	xGrid := ax.AddXGrid()
	yGrid := ax.AddYGrid()
	xGrid.Color = fig.RC.GridColor
	xGrid.LineWidth = fig.RC.GridLineWidth
	yGrid.Color = fig.RC.GridColor
	yGrid.LineWidth = fig.RC.GridLineWidth
	legend := ax.AddLegend()
	legend.Location = core.LegendUpperLeft
	if viper.GetBool("relative") {
		legend.Location = core.LegendLowerLeft
	}

	return saveOwnershipMatplotlibFigure(fig, output, width, height, background)
}

func ownershipSamplingDuration(sampling int, tickSize float64) time.Duration {
	if sampling <= 0 {
		sampling = 1
	}
	if tickSize <= 0 {
		tickSize = 86400
	}
	return secondsDuration(float64(sampling) * tickSize)
}

func secondsDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

func ownershipPointCount(people [][]float64) int {
	for _, row := range people {
		if len(row) > 0 {
			return len(row)
		}
	}
	return 0
}

func truncateOwnershipLabel(label string) string {
	const maxLabelLength = 40
	if len(label) <= maxLabelLength {
		return label
	}
	return label[:maxLabelLength-3] + "..."
}

func normalizeOwnershipColumns(matrix [][]float64) {
	if len(matrix) == 0 {
		return
	}
	points := len(matrix[0])
	for col := 0; col < points; col++ {
		total := 0.0
		for _, row := range matrix {
			if col < len(row) {
				total += row[col]
			}
		}
		if total == 0 {
			continue
		}
		for _, row := range matrix {
			if col < len(row) {
				row[col] /= total
			}
		}
	}
}

func floorTimeBySeconds(t time.Time, seconds float64) time.Time {
	if seconds <= 0 {
		return t
	}
	step := int64(seconds)
	if step <= 0 {
		return t
	}
	return time.Unix((t.Unix()/step)*step, 0)
}

func configureOwnershipTimeAxis(ax *core.Axes, dates []time.Time) {
	if len(dates) == 0 {
		return
	}
	limit := 8
	step := int(math.Ceil(float64(len(dates)) / float64(limit)))
	if step < 1 {
		step = 1
	}
	ticks := make([]float64, 0, limit+1)
	labels := make([]string, 0, limit+1)
	for i := 0; i < len(dates); i += step {
		ticks = append(ticks, float64(dates[i].Unix()))
		labels = append(labels, dates[i].Format("2006-01-02"))
	}
	lastTick := float64(dates[len(dates)-1].Unix())
	if len(ticks) == 0 || ticks[len(ticks)-1] != lastTick {
		ticks = append(ticks, lastTick)
		labels = append(labels, dates[len(dates)-1].Format("2006-01-02"))
	}
	ax.XAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.XAxis.Formatter = core.FixedFormatter{Labels: labels}
	if len(labels) > 6 {
		ax.XAxis.MajorLabelStyle = core.TickLabelStyle{Rotation: 30, AutoAlign: true}
	}
}

func maxOwnershipStackY(matrix [][]float64) float64 {
	if len(matrix) == 0 {
		return 0
	}
	points := len(matrix[0])
	maxY := 0.0
	for i := 0; i < points; i++ {
		total := 0.0
		for _, row := range matrix {
			if i < len(row) {
				total += row[i]
			}
		}
		if total > maxY {
			maxY = total
		}
	}
	return maxY
}

func ownershipPlotColors(backgroundName string) (background, foreground render.Color) {
	if strings.EqualFold(backgroundName, "black") {
		return render.Color{R: 0, G: 0, B: 0, A: 1}, render.Color{R: 1, G: 1, B: 1, A: 1}
	}
	return render.Color{R: 1, G: 1, B: 1, A: 1}, render.Color{R: 0, G: 0, B: 0, A: 1}
}

func ownershipRenderColor(c color.Color) render.Color {
	r, g, b, a := c.RGBA()
	return render.Color{
		R: float64(r) / 0xffff,
		G: float64(g) / 0xffff,
		B: float64(b) / 0xffff,
		A: float64(a) / 0xffff,
	}
}

func ownershipPlotPixelSize(defaultWidth, defaultHeight float64) (int, int) {
	width := defaultWidth
	height := defaultHeight
	if sizeStr := viper.GetString("size"); sizeStr != "" {
		if parsedWidth, parsedHeight, err := parseOwnershipPlotSize(sizeStr); err == nil {
			width, height = parsedWidth, parsedHeight
		} else {
			fmt.Printf("Warning: %v, using default size\n", err)
		}
	}
	return max(1, int(math.Round(width*100))), max(1, int(math.Round(height*100)))
}

func parseOwnershipPlotSize(sizeStr string) (float64, float64, error) {
	parts := strings.Split(sizeStr, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size %q: expected width,height", sizeStr)
	}
	width, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil || width <= 0 {
		return 0, 0, fmt.Errorf("invalid plot width %q", parts[0])
	}
	height, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil || height <= 0 {
		return 0, 0, fmt.Errorf("invalid plot height %q", parts[1])
	}
	return width, height, nil
}

func saveOwnershipMatplotlibFigure(fig *core.Figure, output string, width, height int, background render.Color) error {
	if output == "" {
		output = "ownership.png"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory for %s: %v", output, err)
	}

	fig.TightLayout()
	config := backends.Config{Width: width, Height: height, Background: background, DPI: 100}
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

func saveOwnershipBurndownAsJSON(output string, names []string, people [][]float64, dateRange []time.Time, lastTime time.Time) error {
	data := struct {
		Type      string      `json:"type"`
		Names     []string    `json:"names"`
		People    [][]float64 `json:"people"`
		DateRange []time.Time `json:"date_range"`
		Last      time.Time   `json:"last"`
	}{
		Type:      "ownership",
		Names:     names,
		People:    people,
		DateRange: dateRange,
		Last:      lastTime,
	}

	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create JSON output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to write JSON data: %v", err)
	}

	fmt.Printf("JSON data saved to %s\n", output)
	return nil
}

func argsortDescending(data []float64) []int {
	indices := make([]int, len(data))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return data[indices[i]] > data[indices[j]]
	})
	return indices
}

func argsortAscending(data []int) []int {
	indices := make([]int, len(data))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return data[indices[i]] < data[indices[j]]
	})
	return indices
}

func findFirstNonZero(row []float64) int {
	for i, val := range row {
		if val > 0 {
			return i
		}
	}
	return math.MaxInt
}

func reorder(data [][]float64, indices []int) [][]float64 {
	reordered := make([][]float64, len(indices))
	for i, idx := range indices {
		reordered[i] = data[idx]
	}
	return reordered
}

func reorderStrings(data []string, indices []int) []string {
	reordered := make([]string, len(indices))
	for i, idx := range indices {
		reordered[i] = data[idx]
	}
	return reordered
}
