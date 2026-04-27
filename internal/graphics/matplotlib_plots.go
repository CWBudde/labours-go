package graphics

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"matplotlib-go/core"
	"matplotlib-go/render"
	"matplotlib-go/style"
)

type MatplotlibTimeAreaSeries struct {
	Label  string
	Values []float64
	Color  color.Color
}

type MatplotlibTextLabel struct {
	X        float64
	Y        float64
	Text     string
	HAlign   core.TextAlign
	FontSize float64
}

type MatplotlibTimeAreaOptions struct {
	Title        string
	XLabel       string
	YLabel       string
	Output       string
	WidthInches  float64
	HeightInches float64
	Stacked      bool
	HideY        bool
	ShowGrid     bool
	Legend       bool
	LegendLeft   bool
	LegendTop    bool
	Alpha        float64
	YMin         float64
	YMax         float64
	Baselines    [][]float64
	TextLabels   []MatplotlibTextLabel
}

type MatplotlibBarOptions struct {
	Title        string
	XLabel       string
	YLabel       string
	Output       string
	WidthInches  float64
	HeightInches float64
	RotateX      bool
}

func PlotTimeAreasMatplotlib(dates []time.Time, series []MatplotlibTimeAreaSeries, opts MatplotlibTimeAreaOptions) error {
	if len(dates) == 0 {
		return fmt.Errorf("no dates to plot")
	}
	if len(series) == 0 {
		return fmt.Errorf("no series to plot")
	}

	x := make([]float64, len(dates))
	for i, date := range dates {
		x[i] = float64(date.Unix())
	}

	width, height := pythonPlotPixelSize(defaultPlotWidth(opts.WidthInches), defaultPlotHeight(opts.HeightInches))
	fig := core.NewFigure(
		width,
		height,
		style.WithTheme(style.ThemeGGPlot),
		style.WithFont("DejaVu Sans", 12),
	)
	ax := fig.AddSubplot(1, 1, 1)
	if ax == nil {
		return fmt.Errorf("failed to create axes")
	}
	configureTimeAreaAxes(ax, dates, opts)

	colors := make([]render.Color, len(series))
	matrix := make([][]float64, len(series))
	labels := make([]string, len(series))
	for i, item := range series {
		if len(item.Values) != len(dates) {
			return fmt.Errorf("series %q has %d values for %d dates", item.Label, len(item.Values), len(dates))
		}
		c := item.Color
		if c == nil {
			palette := PythonLaboursColorPalette(len(series))
			c = palette[i%len(palette)]
		}
		colors[i] = renderColor(c)
		matrix[i] = append([]float64(nil), item.Values...)
		labels[i] = item.Label
	}

	alpha := opts.Alpha
	if alpha <= 0 || alpha > 1 {
		alpha = 1
	}
	edgeWidth := 0.0
	zero := make([]float64, len(dates))
	if opts.Stacked {
		ax.StackPlot(x, matrix, core.StackPlotOptions{
			Colors:    colors,
			Labels:    labels,
			Alpha:     &alpha,
			EdgeWidth: &edgeWidth,
		})
	} else {
		for i, item := range series {
			color := colors[i]
			baseline := zero
			if i < len(opts.Baselines) {
				if len(opts.Baselines[i]) != len(dates) {
					return fmt.Errorf("baseline %d has %d values for %d dates", i, len(opts.Baselines[i]), len(dates))
				}
				baseline = opts.Baselines[i]
			}
			ax.FillBetween(x, item.Values, baseline, core.FillOptions{
				Color:     &color,
				Alpha:     &alpha,
				EdgeWidth: &edgeWidth,
				Label:     item.Label,
			})
		}
	}

	for _, label := range opts.TextLabels {
		fontSize := label.FontSize
		if fontSize == 0 {
			fontSize = 12
		}
		ax.Text(label.X, label.Y, label.Text, core.TextOptions{
			FontSize: fontSize,
			Color:    render.Color{R: 0, G: 0, B: 0, A: 1},
			HAlign:   label.HAlign,
			VAlign:   core.TextVAlignMiddle,
		})
	}

	if opts.Legend {
		legend := ax.AddLegend()
		if opts.LegendLeft && opts.LegendTop {
			legend.Location = core.LegendUpperLeft
		} else if opts.LegendLeft {
			legend.Location = core.LegendLowerLeft
		}
	}

	return saveMatplotlibFigure(fig, opts.Output, width, height)
}

func PlotBarChartMatplotlib(labels []string, values []float64, opts MatplotlibBarOptions) error {
	if len(labels) == 0 || len(values) == 0 {
		return fmt.Errorf("no bar data to plot")
	}
	if len(labels) != len(values) {
		return fmt.Errorf("bar labels and values length mismatch")
	}

	width, height := pythonPlotPixelSize(defaultPlotWidth(opts.WidthInches), defaultPlotHeight(opts.HeightInches))
	fig := core.NewFigure(
		width,
		height,
		style.WithTheme(style.ThemeGGPlot),
		style.WithFont("DejaVu Sans", 12),
	)
	ax := fig.AddSubplot(1, 1, 1)
	if ax == nil {
		return fmt.Errorf("failed to create axes")
	}
	ax.SetTitle(opts.Title)
	ax.SetXLabel(opts.XLabel)
	ax.SetYLabel(opts.YLabel)
	ax.AddYGrid()

	x := make([]float64, len(values))
	ticks := make([]float64, len(values))
	for i := range values {
		x[i] = float64(i)
		ticks[i] = float64(i)
	}
	color := renderColor(PythonLaboursColorPalette(1)[0])
	ax.Bar(x, values, core.BarOptions{Color: &color})
	ax.SetXLim(-0.5, float64(len(values))-0.5)
	ax.SetYLim(0, math.Max(maxFloat64(values)*1.05, 1))
	ax.XAxis.Locator = core.FixedLocator{TicksList: ticks}
	ax.XAxis.Formatter = core.FixedFormatter{Labels: append([]string(nil), labels...)}
	if opts.RotateX {
		ax.XAxis.MajorLabelStyle = core.TickLabelStyle{Rotation: 45, AutoAlign: true}
	}

	return saveMatplotlibFigure(fig, opts.Output, width, height)
}

func configureTimeAreaAxes(ax *core.Axes, dates []time.Time, opts MatplotlibTimeAreaOptions) {
	ax.SetTitle(opts.Title)
	ax.SetXLabel(opts.XLabel)
	ax.SetYLabel(opts.YLabel)
	xMin := float64(dates[0].Unix())
	xMax := float64(dates[len(dates)-1].Unix())
	if xMin == xMax {
		xMin = float64(dates[0].AddDate(-2, 0, 0).Unix())
		xMax = float64(dates[0].AddDate(2, 0, 0).Unix())
	}
	ax.SetXLim(xMin, xMax)
	if opts.YMax > opts.YMin {
		ax.SetYLim(opts.YMin, opts.YMax)
	}
	ticks, labels := burndownDateTicks(dates, "")
	if len(ticks) > 0 {
		ax.XAxis.Locator = core.FixedLocator{TicksList: ticks}
		ax.XAxis.Formatter = core.FixedFormatter{Labels: labels}
		if len(labels) > 6 {
			ax.XAxis.MajorLabelStyle = core.TickLabelStyle{Rotation: 30, AutoAlign: true}
		}
	}
	if opts.HideY {
		ax.YAxis.ShowSpine = false
		ax.YAxis.ShowTicks = false
		ax.YAxis.ShowLabels = false
	}
	if opts.ShowGrid {
		ax.AddXGrid()
		ax.AddYGrid()
	}
}

func defaultPlotWidth(width float64) float64 {
	if width > 0 {
		return width
	}
	return 16
}

func defaultPlotHeight(height float64) float64 {
	if height > 0 {
		return height
	}
	return 12
}

func maxFloat64(values []float64) float64 {
	maxValue := 0.0
	for _, value := range values {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
