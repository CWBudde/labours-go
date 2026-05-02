package graphics

import (
	"fmt"
	"image/color"
)

// Theme represents a complete visual theme configuration
type Theme struct {
	Name         string     `yaml:"name" json:"name"`
	ColorPalette []ColorRGB `yaml:"colors" json:"colors"`
	Background   ColorRGB   `yaml:"background" json:"background"`
	Grid         GridStyle  `yaml:"grid" json:"grid"`
	Text         TextStyle  `yaml:"text" json:"text"`
	Chart        ChartStyle `yaml:"chart" json:"chart"`
	HeatMap      HeatStyle  `yaml:"heatmap" json:"heatmap"`
}

// ColorRGB represents an RGB color that can be serialized
type ColorRGB struct {
	R uint8 `yaml:"r" json:"r"`
	G uint8 `yaml:"g" json:"g"`
	B uint8 `yaml:"b" json:"b"`
	A uint8 `yaml:"a" json:"a"`
}

// ToColor converts ColorRGB to color.Color
func (c ColorRGB) ToColor() color.Color {
	return color.RGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// GridStyle configures grid appearance
type GridStyle struct {
	Show  bool     `yaml:"show" json:"show"`
	Color ColorRGB `yaml:"color" json:"color"`
	Width float64  `yaml:"width" json:"width"`
}

// TextStyle configures text appearance
type TextStyle struct {
	Font      string   `yaml:"font" json:"font"`
	Size      float64  `yaml:"size" json:"size"`
	Color     ColorRGB `yaml:"color" json:"color"`
	TitleSize float64  `yaml:"title_size" json:"title_size"`
	LabelSize float64  `yaml:"label_size" json:"label_size"`
}

// ChartStyle configures chart-specific styling
type ChartStyle struct {
	LineWidth   float64  `yaml:"line_width" json:"line_width"`
	BorderWidth float64  `yaml:"border_width" json:"border_width"`
	BorderColor ColorRGB `yaml:"border_color" json:"border_color"`
	FillOpacity float64  `yaml:"fill_opacity" json:"fill_opacity"`
	LegendShow  bool     `yaml:"legend_show" json:"legend_show"`
	LegendPos   string   `yaml:"legend_position" json:"legend_position"`
}

// HeatStyle configures heatmap-specific styling
type HeatStyle struct {
	ColdColor   ColorRGB `yaml:"cold_color" json:"cold_color"`
	HotColor    ColorRGB `yaml:"hot_color" json:"hot_color"`
	MidColor    ColorRGB `yaml:"mid_color" json:"mid_color"`
	UseMidPoint bool     `yaml:"use_mid_point" json:"use_mid_point"`
}

// Default themes
var (
	DefaultTheme = Theme{
		Name: "default",
		ColorPalette: []ColorRGB{
			{R: 31, G: 119, B: 180, A: 255},  // Blue
			{R: 255, G: 127, B: 14, A: 255},  // Orange
			{R: 44, G: 160, B: 44, A: 255},   // Green
			{R: 214, G: 39, B: 40, A: 255},   // Red
			{R: 148, G: 103, B: 189, A: 255}, // Purple
			{R: 140, G: 86, B: 75, A: 255},   // Brown
			{R: 227, G: 119, B: 194, A: 255}, // Pink
			{R: 127, G: 127, B: 127, A: 255}, // Gray
			{R: 188, G: 189, B: 34, A: 255},  // Olive
			{R: 23, G: 190, B: 207, A: 255},  // Cyan
		},
		Background: ColorRGB{R: 255, G: 255, B: 255, A: 255}, // White
		Grid: GridStyle{
			Show:  true,
			Color: ColorRGB{R: 224, G: 224, B: 224, A: 255},
			Width: 0.5,
		},
		Text: TextStyle{
			Font:      "Arial",
			Size:      10,
			Color:     ColorRGB{R: 0, G: 0, B: 0, A: 255},
			TitleSize: 14,
			LabelSize: 10,
		},
		Chart: ChartStyle{
			LineWidth:   1.0,
			BorderWidth: 1.0,
			BorderColor: ColorRGB{R: 0, G: 0, B: 0, A: 255},
			FillOpacity: 0.7,
			LegendShow:  true,
			LegendPos:   "right",
		},
		HeatMap: HeatStyle{
			ColdColor:   ColorRGB{R: 31, G: 119, B: 180, A: 255},  // Blue
			HotColor:    ColorRGB{R: 214, G: 39, B: 40, A: 255},   // Red
			MidColor:    ColorRGB{R: 148, G: 103, B: 189, A: 255}, // Purple
			UseMidPoint: false,
		},
	}

	DarkTheme = Theme{
		Name: "dark",
		ColorPalette: []ColorRGB{
			{R: 99, G: 165, B: 255, A: 255},  // Light Blue
			{R: 255, G: 159, B: 64, A: 255},  // Light Orange
			{R: 75, G: 192, B: 75, A: 255},   // Light Green
			{R: 255, G: 99, B: 132, A: 255},  // Light Red
			{R: 186, G: 148, B: 255, A: 255}, // Light Purple
			{R: 200, G: 150, B: 130, A: 255}, // Light Brown
			{R: 255, G: 159, B: 226, A: 255}, // Light Pink
			{R: 180, G: 180, B: 180, A: 255}, // Light Gray
			{R: 220, G: 220, B: 100, A: 255}, // Light Olive
			{R: 100, G: 220, B: 240, A: 255}, // Light Cyan
		},
		Background: ColorRGB{R: 35, G: 39, B: 42, A: 255}, // Dark Gray
		Grid: GridStyle{
			Show:  true,
			Color: ColorRGB{R: 68, G: 74, B: 79, A: 255},
			Width: 0.5,
		},
		Text: TextStyle{
			Font:      "Arial",
			Size:      10,
			Color:     ColorRGB{R: 240, G: 240, B: 240, A: 255}, // Light Gray
			TitleSize: 14,
			LabelSize: 10,
		},
		Chart: ChartStyle{
			LineWidth:   1.0,
			BorderWidth: 1.0,
			BorderColor: ColorRGB{R: 200, G: 200, B: 200, A: 255},
			FillOpacity: 0.8,
			LegendShow:  true,
			LegendPos:   "right",
		},
		HeatMap: HeatStyle{
			ColdColor:   ColorRGB{R: 0, G: 100, B: 200, A: 255},  // Dark Blue
			HotColor:    ColorRGB{R: 255, G: 80, B: 80, A: 255},  // Bright Red
			MidColor:    ColorRGB{R: 150, G: 50, B: 200, A: 255}, // Dark Purple
			UseMidPoint: false,
		},
	}

	MinimalTheme = Theme{
		Name: "minimal",
		ColorPalette: []ColorRGB{
			{R: 70, G: 70, B: 70, A: 255},    // Dark Gray
			{R: 150, G: 150, B: 150, A: 255}, // Medium Gray
			{R: 200, G: 200, B: 200, A: 255}, // Light Gray
			{R: 100, G: 100, B: 100, A: 255}, // Another Dark Gray
			{R: 50, G: 50, B: 50, A: 255},    // Very Dark Gray
			{R: 180, G: 180, B: 180, A: 255}, // Very Light Gray
			{R: 120, G: 120, B: 120, A: 255}, // Mid Gray
			{R: 80, G: 80, B: 80, A: 255},    // Dark Gray 2
			{R: 160, G: 160, B: 160, A: 255}, // Light Gray 2
			{R: 110, G: 110, B: 110, A: 255}, // Mid Gray 2
		},
		Background: ColorRGB{R: 255, G: 255, B: 255, A: 255}, // White
		Grid: GridStyle{
			Show:  false,
			Color: ColorRGB{R: 240, G: 240, B: 240, A: 255},
			Width: 0.25,
		},
		Text: TextStyle{
			Font:      "Arial",
			Size:      9,
			Color:     ColorRGB{R: 60, G: 60, B: 60, A: 255},
			TitleSize: 12,
			LabelSize: 8,
		},
		Chart: ChartStyle{
			LineWidth:   0.8,
			BorderWidth: 0.5,
			BorderColor: ColorRGB{R: 120, G: 120, B: 120, A: 255},
			FillOpacity: 0.9,
			LegendShow:  false,
			LegendPos:   "bottom",
		},
		HeatMap: HeatStyle{
			ColdColor:   ColorRGB{R: 240, G: 240, B: 240, A: 255}, // Very Light Gray
			HotColor:    ColorRGB{R: 60, G: 60, B: 60, A: 255},    // Dark Gray
			MidColor:    ColorRGB{R: 150, G: 150, B: 150, A: 255}, // Medium Gray
			UseMidPoint: true,
		},
	}

	VibranthColorTheme = Theme{
		Name: "vibrant",
		ColorPalette: []ColorRGB{
			{R: 255, G: 0, B: 128, A: 255}, // Hot Pink
			{R: 0, G: 255, B: 128, A: 255}, // Spring Green
			{R: 128, G: 0, B: 255, A: 255}, // Electric Violet
			{R: 255, G: 128, B: 0, A: 255}, // Dark Orange
			{R: 0, G: 128, B: 255, A: 255}, // Dodger Blue
			{R: 255, G: 255, B: 0, A: 255}, // Yellow
			{R: 255, G: 0, B: 0, A: 255},   // Red
			{R: 0, G: 255, B: 0, A: 255},   // Lime
			{R: 0, G: 255, B: 255, A: 255}, // Cyan
			{R: 255, G: 0, B: 255, A: 255}, // Magenta
		},
		Background: ColorRGB{R: 250, G: 250, B: 250, A: 255}, // Very Light Gray
		Grid: GridStyle{
			Show:  true,
			Color: ColorRGB{R: 230, G: 230, B: 230, A: 255},
			Width: 0.8,
		},
		Text: TextStyle{
			Font:      "Arial",
			Size:      11,
			Color:     ColorRGB{R: 40, G: 40, B: 40, A: 255},
			TitleSize: 16,
			LabelSize: 11,
		},
		Chart: ChartStyle{
			LineWidth:   1.5,
			BorderWidth: 1.2,
			BorderColor: ColorRGB{R: 80, G: 80, B: 80, A: 255},
			FillOpacity: 0.6,
			LegendShow:  true,
			LegendPos:   "right",
		},
		HeatMap: HeatStyle{
			ColdColor:   ColorRGB{R: 0, G: 100, B: 255, A: 255}, // Bright Blue
			HotColor:    ColorRGB{R: 255, G: 50, B: 50, A: 255}, // Bright Red
			MidColor:    ColorRGB{R: 255, G: 200, B: 0, A: 255}, // Bright Yellow
			UseMidPoint: true,
		},
	}

	MatplotlibTheme = Theme{
		Name: "matplotlib",
		ColorPalette: []ColorRGB{
			{R: 31, G: 119, B: 180, A: 255},  // Blue (C0) - matplotlib default
			{R: 255, G: 127, B: 14, A: 255},  // Orange (C1)
			{R: 44, G: 160, B: 44, A: 255},   // Green (C2)
			{R: 214, G: 39, B: 40, A: 255},   // Red (C3)
			{R: 148, G: 103, B: 189, A: 255}, // Purple (C4)
			{R: 140, G: 86, B: 75, A: 255},   // Brown (C5)
			{R: 227, G: 119, B: 194, A: 255}, // Pink (C6)
			{R: 127, G: 127, B: 127, A: 255}, // Gray (C7)
			{R: 188, G: 189, B: 34, A: 255},  // Olive (C8)
			{R: 23, G: 190, B: 207, A: 255},  // Cyan (C9)
		},
		Background: ColorRGB{R: 255, G: 255, B: 255, A: 255}, // White
		Grid: GridStyle{
			Show:  true,
			Color: ColorRGB{R: 224, G: 224, B: 224, A: 255},
			Width: 0.5,
		},
		Text: TextStyle{
			Font:      "Arial",
			Size:      10,
			Color:     ColorRGB{R: 0, G: 0, B: 0, A: 255},
			TitleSize: 14,
			LabelSize: 10,
		},
		Chart: ChartStyle{
			LineWidth:   1.0,
			BorderWidth: 1.0,
			BorderColor: ColorRGB{R: 0, G: 0, B: 0, A: 255},
			FillOpacity: 0.7,
			LegendShow:  true,
			LegendPos:   "right",
		},
		HeatMap: HeatStyle{
			ColdColor:   ColorRGB{R: 31, G: 119, B: 180, A: 255},  // Blue
			HotColor:    ColorRGB{R: 214, G: 39, B: 40, A: 255},   // Red
			MidColor:    ColorRGB{R: 148, G: 103, B: 189, A: 255}, // Purple
			UseMidPoint: false,
		},
	}
)

// BuiltinThemes contains all built-in themes
var BuiltinThemes = map[string]Theme{
	"default":    DefaultTheme,
	"dark":       DarkTheme,
	"minimal":    MinimalTheme,
	"vibrant":    VibranthColorTheme,
	"matplotlib": MatplotlibTheme,
}

// GetColorPalette returns the color palette as color.Color slice
func (t *Theme) GetColorPalette() []color.Color {
	colors := make([]color.Color, len(t.ColorPalette))
	for i, c := range t.ColorPalette {
		colors[i] = c.ToColor()
	}
	return colors
}

// GetHeatColor generates a heat map color based on ratio and theme settings
func (t *Theme) GetHeatColor(ratio float64) color.Color {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}

	cold := t.HeatMap.ColdColor
	hot := t.HeatMap.HotColor

	if t.HeatMap.UseMidPoint && ratio <= 0.5 {
		// Interpolate from cold to mid
		mid := t.HeatMap.MidColor
		ratio *= 2 // Scale to 0-1 for cold->mid
		r := uint8(float64(cold.R) + ratio*(float64(mid.R)-float64(cold.R)))
		g := uint8(float64(cold.G) + ratio*(float64(mid.G)-float64(cold.G)))
		b := uint8(float64(cold.B) + ratio*(float64(mid.B)-float64(cold.B)))
		return color.RGBA{R: r, G: g, B: b, A: 255}
	} else if t.HeatMap.UseMidPoint {
		// Interpolate from mid to hot
		mid := t.HeatMap.MidColor
		ratio = (ratio - 0.5) * 2 // Scale to 0-1 for mid->hot
		r := uint8(float64(mid.R) + ratio*(float64(hot.R)-float64(mid.R)))
		g := uint8(float64(mid.G) + ratio*(float64(hot.G)-float64(mid.G)))
		b := uint8(float64(mid.B) + ratio*(float64(hot.B)-float64(mid.B)))
		return color.RGBA{R: r, G: g, B: b, A: 255}
	} else {
		// Direct interpolation from cold to hot
		r := uint8(float64(cold.R) + ratio*(float64(hot.R)-float64(cold.R)))
		g := uint8(float64(cold.G) + ratio*(float64(hot.G)-float64(cold.G)))
		b := uint8(float64(cold.B) + ratio*(float64(hot.B)-float64(cold.B)))
		return color.RGBA{R: r, G: g, B: b, A: 255}
	}
}

// Validate checks if theme configuration is valid
func (t *Theme) Validate() error {
	if len(t.ColorPalette) == 0 {
		return fmt.Errorf("theme must have at least one color in palette")
	}

	if t.Name == "" {
		return fmt.Errorf("theme must have a name")
	}

	if t.Text.Size <= 0 {
		return fmt.Errorf("text size must be positive")
	}

	if t.Chart.FillOpacity < 0 || t.Chart.FillOpacity > 1 {
		return fmt.Errorf("fill opacity must be between 0 and 1")
	}

	return nil
}

// CurrentTheme holds the active theme (defaults to DefaultTheme)
var CurrentTheme = DefaultTheme
