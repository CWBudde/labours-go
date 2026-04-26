package modes

import (
	"fmt"
	"os"
	"path/filepath"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/vg"
)

func savePlotPNGAndSVG(p *plot.Plot, width, height vg.Length, outputDir, baseName string) (string, string, error) {
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create output directory %s: %v", outputDir, err)
	}

	pngPath := filepath.Join(outputDir, baseName+".png")
	if err := p.Save(width, height, pngPath); err != nil {
		return "", "", fmt.Errorf("failed to save PNG plot %s: %v", pngPath, err)
	}

	svgPath := filepath.Join(outputDir, baseName+".svg")
	if err := p.Save(width, height, svgPath); err != nil {
		return "", "", fmt.Errorf("failed to save SVG plot %s: %v", svgPath, err)
	}

	return pngPath, svgPath, nil
}
