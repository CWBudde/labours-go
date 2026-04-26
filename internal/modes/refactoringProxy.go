package modes

import (
	"fmt"
	"os"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/readers"
)

func RefactoringProxy(reader readers.Reader, output string) error {
	proxyReader, ok := reader.(readers.RefactoringProxyReader)
	if !ok {
		return fmt.Errorf("%w: RefactoringProxy", readers.ErrAnalysisMissing)
	}
	data, err := proxyReader.GetRefactoringProxy()
	if err != nil {
		return err
	}
	if data == nil || len(data.Ticks) == 0 {
		return fmt.Errorf("%w: RefactoringProxy", readers.ErrAnalysisMissing)
	}
	return plotRefactoringProxy(data, output)
}

func plotRefactoringProxy(data *readers.RefactoringProxyData, output string) error {
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil && filepath.Dir(output) != "." {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	p := plot.New()
	p.Title.Text = "Refactoring Proxy"
	p.X.Label.Text = "Tick"
	p.Y.Label.Text = "Rename Ratio"

	points := make(plotter.XYs, len(data.Ticks))
	refactoringPoints := make(plotter.XYs, 0, len(data.Ticks))
	for i, tick := range data.Ticks {
		points[i].X = float64(i)
		points[i].Y = float64(tick.RefactoringRate)
		if tick.IsRefactoring {
			refactoringPoints = append(refactoringPoints, points[i])
		}
	}

	line, err := plotter.NewLine(points)
	if err != nil {
		return fmt.Errorf("failed to create refactoring proxy line: %v", err)
	}
	line.Color = graphics.ColorPalette[0]
	line.Width = vg.Points(2)
	p.Add(line)
	p.Legend.Add("Rename ratio", line)

	if len(refactoringPoints) > 0 {
		scatter, err := plotter.NewScatter(refactoringPoints)
		if err != nil {
			return fmt.Errorf("failed to create refactoring markers: %v", err)
		}
		scatter.GlyphStyle.Color = graphics.ColorPalette[3]
		scatter.GlyphStyle.Radius = vg.Points(4)
		p.Add(scatter)
		p.Legend.Add("Refactoring", scatter)
	}

	threshold := float64(data.Threshold)
	if threshold > 0 {
		thresholdLine := plotter.NewFunction(func(float64) float64 { return threshold })
		thresholdLine.Color = graphics.ColorPalette[1]
		thresholdLine.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
		p.Add(thresholdLine)
		p.Legend.Add("Threshold", thresholdLine)
	}

	width, height := graphics.GetPlotSize(graphics.ChartTypeDefault)
	if output == "" {
		output = "refactoring-proxy.png"
	}
	if err := p.Save(width, height, output); err != nil {
		return fmt.Errorf("failed to save refactoring proxy chart: %v", err)
	}
	fmt.Printf("Saved refactoring proxy chart to %s\n", output)
	return nil
}
