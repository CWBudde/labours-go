package graphics

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"labours-go/internal/burndown"
	"matplotlib-go/backends"
	_ "matplotlib-go/backends/agg"
	_ "matplotlib-go/backends/svg"
	"matplotlib-go/core"
	"matplotlib-go/render"
	"matplotlib-go/style"
)

// PlotBurndownMatplotlib creates a burndown plot with matplotlib-go stackplot rendering.
func PlotBurndownMatplotlib(data *burndown.ProcessedBurndown, output string, relative bool) error {
	if data == nil || len(data.Matrix) == 0 || len(data.DateRange) == 0 {
		return fmt.Errorf("empty burndown data")
	}

	numSeries := len(data.Matrix)
	numPoints := len(data.DateRange)

	// Ensure matrix dimensions are consistent
	if numSeries == 0 {
		return fmt.Errorf("empty matrix")
	}

	// Convert dates to float64 for plotting (Unix timestamps)
	timeValues := make([]float64, numPoints)
	for i, date := range data.DateRange {
		timeValues[i] = float64(date.Unix())
	}

	// Python mutates the matrix after stackplot(), so the rendered reference
	// remains unnormalized and is only clipped by ylim(0, 1).
	matrix := data.Matrix

	if !viper.GetBool("quiet") {
		fmt.Printf("DEBUG MATRIX ANALYSIS:\n")
		fmt.Printf("  Matrix dimensions: %dx%d\n", len(matrix), len(matrix[0]))
		for i := 0; i < len(matrix); i++ {
			minVal, maxVal := matrix[i][0], matrix[i][0]
			negCount, posCount := 0, 0
			for j := 0; j < len(matrix[i]); j++ {
				if matrix[i][j] < minVal {
					minVal = matrix[i][j]
				}
				if matrix[i][j] > maxVal {
					maxVal = matrix[i][j]
				}
				if matrix[i][j] < 0 {
					negCount++
				}
				if matrix[i][j] > 0 {
					posCount++
				}
			}
			fmt.Printf("  Layer %d: min=%.2f, max=%.2f, negatives=%d, positives=%d\n", i, minVal, maxVal, negCount, posCount)
		}
	}

	colors := PythonLaboursColorPalette(numSeries)
	renderColors := make([]render.Color, len(colors))
	for i, c := range colors {
		renderColors[i] = renderColor(c)
	}
	labels := make([]string, numSeries)
	for i := range labels {
		labels[i] = fmt.Sprintf("Layer %d", i)
		if i < len(data.Labels) {
			labels[i] = data.Labels[i]
		}
	}

	width, height := pythonPlotPixelSize(16, 12)
	fontSize := viper.GetInt("font-size")
	if fontSize <= 0 {
		fontSize = 12
	}
	background, foreground := laboursPlotColors(viper.GetString("background"))
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
	ax := fig.GridSpec(
		1,
		1,
		core.WithGridSpecPadding(pythonBurndownAxesPadding(matrix, relative)),
	).Cell(0, 0).AddAxes()
	if ax == nil {
		return fmt.Errorf("failed to create burndown axes")
	}
	ax.SetTitle(fmt.Sprintf("%s %d x %d (granularity %d, sampling %d)",
		data.Name, len(data.Matrix), len(data.DateRange), data.Granularity, data.Sampling))
	ax.SetXLabel("Time")
	ax.SetYLabel("Lines of code")
	plotMatrix := matrix
	if relative {
		// Python lets matplotlib clip the unnormalized stackplot after ylim(0, 1).
		// matplotlib-go's AGG backend currently drops those out-of-range fills,
		// so pre-clip the stacked layers to the same visible result.
		plotMatrix = clippedStackMatrix(matrix, 0, 1)
	}
	ax.StackPlot(timeValues, plotMatrix, core.StackPlotOptions{
		Colors: renderColors,
		Labels: labels,
	})
	ax.SetXLim(timeValues[0], timeValues[len(timeValues)-1])
	if relative {
		ax.SetYLim(0, 1)
	} else {
		configureMatplotlibBurndownYAxis(fig, ax, matrix, float64(fontSize), foreground)
	}
	configureMatplotlibBurndownTimeAxis(ax, data.DateRange, data.ResampleMode)

	legend := ax.AddLegend()
	legend.Location = core.LegendUpperLeft
	if relative {
		legend.Location = core.LegendLowerLeft
	}

	return saveMatplotlibFigureWithoutTightLayout(fig, output, width, height, background)
}

func pythonBurndownAxesPadding(matrix [][]float64, relative bool) (left, right, bottom, top float64) {
	left = 0.036
	if relative {
		left = 0.045
	} else if maxStackY(matrix) < 1000 {
		left = 0.041
	}
	return left, 0.991, 0.049, 0.968
}

func clippedStackMatrix(matrix [][]float64, minY, maxY float64) [][]float64 {
	if len(matrix) == 0 {
		return nil
	}
	points := 0
	for _, row := range matrix {
		if points == 0 || len(row) < points {
			points = len(row)
		}
	}
	if points == 0 {
		return nil
	}

	clipped := make([][]float64, len(matrix))
	cumulative := make([]float64, points)
	for i, row := range matrix {
		clipped[i] = make([]float64, points)
		for j := 0; j < points; j++ {
			lower := cumulative[j]
			upper := lower + row[j]
			clippedLower := clampFloat(lower, minY, maxY)
			clippedUpper := clampFloat(upper, minY, maxY)
			if clippedUpper > clippedLower {
				clipped[i][j] = clippedUpper - clippedLower
			}
			cumulative[j] = upper
		}
	}
	return clipped
}

func configureMatplotlibBurndownYAxis(fig *core.Figure, ax *core.Axes, matrix [][]float64, fontSize float64, foreground render.Color) {
	maxY := maxStackY(matrix)
	if maxY <= 0 {
		ax.SetYLim(0, 1)
		return
	}
	ax.SetYLim(0, maxY*1.05)

	ticks, labels, offset := burndownYAxisTicks(maxY)
	if len(ticks) == 0 {
		return
	}
	ax.YAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.YAxis.Formatter = core.FixedFormatter{Labels: labels}
	clipOff := false
	fig.Text(0.036, 0.985, offset, core.TextOptions{
		FontSize: fontSize,
		Color:    foreground,
		ClipOn:   &clipOff,
	})
}

func burndownYAxisTicks(maxY float64) ([]float64, []string, string) {
	if maxY < 1000 {
		return nil, nil, ""
	}

	exponent := int(math.Floor(math.Log10(maxY)))
	scale := math.Pow10(exponent)
	scaledTop := maxY * 1.05 / scale
	step := niceBurndownTickStep(scaledTop)
	if step <= 0 {
		return nil, nil, ""
	}
	count := int(math.Floor(scaledTop/step + 1e-9))
	if count < 1 {
		count = 1
	}

	ticks := make([]float64, 0, count+1)
	labels := make([]string, 0, count+1)
	for i := 0; i <= count; i++ {
		scaled := float64(i) * step
		ticks = append(ticks, scaled*scale)
		labels = append(labels, formatBurndownScaledTick(scaled))
	}
	return ticks, labels, fmt.Sprintf("1e%d", exponent)
}

func niceBurndownTickStep(scaledTop float64) float64 {
	if scaledTop <= 0 {
		return 0
	}
	raw := scaledTop / 8
	base := math.Pow10(int(math.Floor(math.Log10(raw))))
	for _, multiplier := range []float64{1, 2, 2.5, 5, 10} {
		step := multiplier * base
		if raw <= step {
			return step
		}
	}
	return 10 * base
}

func formatBurndownScaledTick(value float64) string {
	if math.Abs(value-math.Round(value)) < 1e-9 {
		return fmt.Sprintf("%.0f", math.Round(value))
	}
	label := fmt.Sprintf("%.3f", value)
	label = strings.TrimRight(label, "0")
	return strings.TrimRight(label, ".")
}

func maxStackY(matrix [][]float64) float64 {
	points := 0
	for _, row := range matrix {
		if points == 0 || len(row) < points {
			points = len(row)
		}
	}
	maxY := 0.0
	for j := 0; j < points; j++ {
		total := 0.0
		for i := range matrix {
			total += matrix[i][j]
		}
		if total > maxY {
			maxY = total
		}
	}
	return maxY
}

func laboursPlotColors(backgroundName string) (background, foreground render.Color) {
	if strings.EqualFold(backgroundName, "black") {
		return render.Color{R: 0, G: 0, B: 0, A: 1}, render.Color{R: 1, G: 1, B: 1, A: 1}
	}
	return render.Color{R: 1, G: 1, B: 1, A: 1}, render.Color{R: 0, G: 0, B: 0, A: 1}
}

func configureMatplotlibBurndownTimeAxis(ax *core.Axes, dates []time.Time, resampleMode string) {
	if ax == nil || len(dates) == 0 {
		return
	}

	ticks, labels := burndownDateTicks(dates, resampleMode)
	if len(ticks) == 0 {
		return
	}
	ax.XAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.XAxis.Formatter = core.FixedFormatter{Labels: labels}
	if len(labels) > 6 {
		ax.XAxis.MajorLabelStyle = core.TickLabelStyle{Rotation: 30, AutoAlign: true}
	}
}

func burndownDateTicks(dates []time.Time, resampleMode string) ([]float64, []string) {
	if len(dates) == 0 {
		return nil, nil
	}
	start := dates[0]
	end := dates[len(dates)-1]
	if end.Before(start) {
		start, end = end, start
	}

	mode := strings.ToLower(resampleMode)
	switch {
	case strings.Contains(mode, "m"):
		return buildMonthlyTicks(start, end)
	case strings.Contains(mode, "a"), strings.Contains(mode, "y"), strings.Contains(mode, "year"):
		return buildYearlyTicks(start, end)
	}

	days := end.Sub(start).Hours() / 24
	switch {
	case days <= 45:
		return buildDailyTicks(start, end)
	case days <= 730:
		return buildMonthlyTicks(start, end)
	default:
		return buildYearlyTicks(start, end)
	}
}

func buildDailyTicks(start, end time.Time) ([]float64, []string) {
	days := int(math.Ceil(end.Sub(start).Hours() / 24))
	step := max(1, int(math.Ceil(float64(max(1, days))/10)))
	ticks := []time.Time{start}
	for t := start.Truncate(24 * time.Hour); !t.After(end); t = t.AddDate(0, 0, step) {
		if !t.Before(start) {
			ticks = append(ticks, t)
		}
	}
	ticks = append(ticks, end)
	return formatDateTicks(ticks, "2006-01-02")
}

func buildMonthlyTicks(start, end time.Time) ([]float64, []string) {
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	step := max(1, int(math.Ceil(float64(max(1, months))/10)))
	ticks := []time.Time{start}
	for t := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location()); !t.After(end); t = t.AddDate(0, step, 0) {
		if !t.Before(start) {
			ticks = append(ticks, t)
		}
	}
	ticks = append(ticks, end)
	return formatDateTicks(ticks, "2006-01")
}

func buildYearlyTicks(start, end time.Time) ([]float64, []string) {
	years := max(1, end.Year()-start.Year()+1)
	step := max(1, int(math.Ceil(float64(years)/10)))
	ticks := make([]time.Time, 0, years)
	for year := start.Year(); year <= end.Year(); year += step {
		t := time.Date(year, 1, 1, 0, 0, 0, 0, start.Location())
		if !t.Before(start) && !t.After(end) {
			ticks = append(ticks, t)
		}
	}
	if len(ticks) == 0 {
		ticks = append(ticks, start)
	}
	if len(ticks) == 1 {
		return formatDateTicks(ticks, "2006-01-02")
	}
	return formatDateTicks(ticks, "2006")
}

func formatDateTicks(dates []time.Time, layout string) ([]float64, []string) {
	seen := map[int64]bool{}
	ticks := make([]float64, 0, len(dates))
	labels := make([]string, 0, len(dates))
	for _, date := range dates {
		unix := date.Unix()
		if seen[unix] {
			continue
		}
		seen[unix] = true
		ticks = append(ticks, float64(unix))
		labels = append(labels, date.Format(layout))
	}
	return ticks, labels
}

func saveMatplotlibFigure(fig *core.Figure, output string, width, height int, backgrounds ...render.Color) error {
	return saveMatplotlibFigureWithLayout(fig, output, width, height, true, backgrounds...)
}

func saveMatplotlibFigureWithoutTightLayout(fig *core.Figure, output string, width, height int, backgrounds ...render.Color) error {
	return saveMatplotlibFigureWithLayout(fig, output, width, height, false, backgrounds...)
}

func saveMatplotlibFigureWithLayout(fig *core.Figure, output string, width, height int, tight bool, backgrounds ...render.Color) error {
	if output == "" {
		output = "burndown_project_python.png"
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return fmt.Errorf("failed to create output directory for %s: %v", output, err)
	}

	if tight {
		fig.TightLayout()
	}
	background := render.Color{R: 1, G: 1, B: 1, A: 0}
	if len(backgrounds) > 0 {
		background = backgrounds[0]
	}
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
		if err := core.SavePNG(fig, renderer, output); err != nil {
			return err
		}
		if background.A == 0 {
			return setTransparentPNGRGB(output, background)
		}
		return nil
	}
}

func renderColor(c color.Color) render.Color {
	r, g, b, a := c.RGBA()
	return render.Color{
		R: float64(r) / 0xffff,
		G: float64(g) / 0xffff,
		B: float64(b) / 0xffff,
		A: float64(a) / 0xffff,
	}
}

func pythonPlotPixelSize(defaultWidth, defaultHeight float64) (int, int) {
	width := defaultWidth
	height := defaultHeight
	if sizeStr := viper.GetString("size"); sizeStr != "" {
		parsedWidth, parsedHeight, err := parsePlotSizeFloats(sizeStr)
		if err == nil {
			width, height = parsedWidth, parsedHeight
		} else {
			fmt.Printf("Warning: %v, using default size\n", err)
		}
	}
	return max(1, int(math.Round(width*100))), max(1, int(math.Round(height*100)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampFloat(v, minVal, maxVal float64) float64 {
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}

func setTransparentPNGRGB(path string, background render.Color) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	img, err := png.Decode(file)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}

	bounds := img.Bounds()
	rgba := imageToNRGBA(img)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			offset := rgba.PixOffset(x, y)
			if rgba.Pix[offset+3] == 0 {
				rgba.Pix[offset+0] = uint8(math.Round(background.R * 255))
				rgba.Pix[offset+1] = uint8(math.Round(background.G * 255))
				rgba.Pix[offset+2] = uint8(math.Round(background.B * 255))
			}
		}
	}

	file, err = os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, rgba)
}

func imageToNRGBA(img image.Image) *image.NRGBA {
	if rgba, ok := img.(*image.NRGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewNRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

// PrintSurvivalFunction prints survival ratios to match Python output (placeholder)
func PrintSurvivalFunction(matrix [][]float64) {
	fmt.Println("           Ratio of survived lines")
	// TODO: Implement Kaplan-Meier survival analysis like Python
	// For now, just print a placeholder that shows we're processing survival data

	if len(matrix) > 0 && len(matrix[0]) > 0 {
		total := 0.0
		for i := range matrix {
			for j := range matrix[i] {
				total += matrix[i][j]
			}
		}

		for i := 0; i < len(matrix[0]); i++ {
			alive := 0.0
			for j := range matrix {
				if i < len(matrix[j]) {
					alive += matrix[j][i]
				}
			}
			if total > 0 {
				ratio := alive / total
				fmt.Printf("%d days\t\t%.6f\n", i, ratio)
			}
		}
	}
}

// generatePythonLaboursColorPalette matches Python labours' tab20 color cycle.
func PythonLaboursColorPalette(n int) []color.Color {
	tab20Colors := []color.Color{
		color.RGBA{R: 31, G: 119, B: 180, A: 255},
		color.RGBA{R: 174, G: 199, B: 232, A: 255},
		color.RGBA{R: 255, G: 127, B: 14, A: 255},
		color.RGBA{R: 255, G: 187, B: 120, A: 255},
		color.RGBA{R: 44, G: 160, B: 44, A: 255},
		color.RGBA{R: 152, G: 223, B: 138, A: 255},
		color.RGBA{R: 214, G: 39, B: 40, A: 255},
		color.RGBA{R: 255, G: 152, B: 150, A: 255},
		color.RGBA{R: 148, G: 103, B: 189, A: 255},
		color.RGBA{R: 197, G: 176, B: 213, A: 255},
		color.RGBA{R: 140, G: 86, B: 75, A: 255},
		color.RGBA{R: 196, G: 156, B: 148, A: 255},
		color.RGBA{R: 227, G: 119, B: 194, A: 255},
		color.RGBA{R: 247, G: 182, B: 210, A: 255},
		color.RGBA{R: 127, G: 127, B: 127, A: 255},
		color.RGBA{R: 199, G: 199, B: 199, A: 255},
		color.RGBA{R: 188, G: 189, B: 34, A: 255},
		color.RGBA{R: 219, G: 219, B: 141, A: 255},
		color.RGBA{R: 23, G: 190, B: 207, A: 255},
		color.RGBA{R: 158, G: 218, B: 229, A: 255},
	}

	colors := make([]color.Color, n)
	for i := 0; i < n; i++ {
		colors[i] = tab20Colors[i%len(tab20Colors)]
	}

	return colors
}
