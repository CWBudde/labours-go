package graphics

import (
	"testing"

	"github.com/spf13/viper"
	"gonum.org/v1/plot/vg"
)

func TestParsePlotSize(t *testing.T) {
	tests := []struct {
		name           string
		sizeStr        string
		chartType      ChartType
		expectedWidth  float64
		expectedHeight float64
		expectError    bool
	}{
		{
			name:           "empty string uses default",
			sizeStr:        "",
			chartType:      ChartTypeDefault,
			expectedWidth:  16.0,
			expectedHeight: 8.0,
			expectError:    false,
		},
		{
			name:           "empty string uses square default",
			sizeStr:        "",
			chartType:      ChartTypeSquare,
			expectedWidth:  12.0,
			expectedHeight: 12.0,
			expectError:    false,
		},
		{
			name:           "empty string uses compact default",
			sizeStr:        "",
			chartType:      ChartTypeCompact,
			expectedWidth:  10.0,
			expectedHeight: 6.0,
			expectError:    false,
		},
		{
			name:           "valid size string",
			sizeStr:        "14,10",
			chartType:      ChartTypeDefault,
			expectedWidth:  14.0,
			expectedHeight: 10.0,
			expectError:    false,
		},
		{
			name:           "valid size string with spaces",
			sizeStr:        " 12 , 8 ",
			chartType:      ChartTypeDefault,
			expectedWidth:  12.0,
			expectedHeight: 8.0,
			expectError:    false,
		},
		{
			name:           "Python labours compatible format",
			sizeStr:        "16,12",
			chartType:      ChartTypeDefault,
			expectedWidth:  16.0,
			expectedHeight: 12.0,
			expectError:    false,
		},
		{
			name:        "invalid format - no comma",
			sizeStr:     "12x8",
			chartType:   ChartTypeDefault,
			expectError: true,
		},
		{
			name:        "invalid format - too many parts",
			sizeStr:     "12,8,4",
			chartType:   ChartTypeDefault,
			expectError: true,
		},
		{
			name:        "invalid width",
			sizeStr:     "abc,8",
			chartType:   ChartTypeDefault,
			expectError: true,
		},
		{
			name:        "invalid height",
			sizeStr:     "12,xyz",
			chartType:   ChartTypeDefault,
			expectError: true,
		},
		{
			name:        "zero width",
			sizeStr:     "0,8",
			chartType:   ChartTypeDefault,
			expectError: true,
		},
		{
			name:        "negative height",
			sizeStr:     "12,-5",
			chartType:   ChartTypeDefault,
			expectError: true,
		},
		{
			name:        "dimensions too large",
			sizeStr:     "100,80",
			chartType:   ChartTypeDefault,
			expectError: true,
		},
		{
			name:           "decimal values",
			sizeStr:        "12.5,8.5",
			chartType:      ChartTypeDefault,
			expectedWidth:  12.5,
			expectedHeight: 8.5,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height, err := ParsePlotSize(tt.sizeStr, tt.chartType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Convert to inches for comparison
			widthInches := float64(width / vg.Inch)
			heightInches := float64(height / vg.Inch)

			if widthInches != tt.expectedWidth {
				t.Errorf("width mismatch: expected %f, got %f", tt.expectedWidth, widthInches)
			}

			if heightInches != tt.expectedHeight {
				t.Errorf("height mismatch: expected %f, got %f", tt.expectedHeight, heightInches)
			}
		})
	}
}

func TestGetPlotSizeInches(t *testing.T) {
	tests := []struct {
		name           string
		chartType      ChartType
		expectedWidth  float64
		expectedHeight float64
	}{
		{
			name:           "default chart type",
			chartType:      ChartTypeDefault,
			expectedWidth:  16.0,
			expectedHeight: 8.0,
		},
		{
			name:           "square chart type",
			chartType:      ChartTypeSquare,
			expectedWidth:  12.0,
			expectedHeight: 12.0,
		},
		{
			name:           "compact chart type",
			chartType:      ChartTypeCompact,
			expectedWidth:  10.0,
			expectedHeight: 6.0,
		},
		{
			name:           "wide chart type",
			chartType:      ChartTypeWide,
			expectedWidth:  16.0,
			expectedHeight: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height := GetPlotSizeInches(tt.chartType)

			if width != tt.expectedWidth {
				t.Errorf("width mismatch: expected %f, got %f", tt.expectedWidth, width)
			}

			if height != tt.expectedHeight {
				t.Errorf("height mismatch: expected %f, got %f", tt.expectedHeight, height)
			}
		})
	}
}

func TestGetPythonPlotSizeUsesMatplotlibDPI(t *testing.T) {
	oldSize := viper.GetString("size")
	defer viper.Set("size", oldSize)

	viper.Set("size", "")
	width, height := GetPythonPlotSize(16, 12)
	if got, want := int(width), 1600; got != want {
		t.Fatalf("default width = %d, want %d", got, want)
	}
	if got, want := int(height), 1200; got != want {
		t.Fatalf("default height = %d, want %d", got, want)
	}

	viper.Set("size", "12,9")
	width, height = GetPythonPlotSize(16, 12)
	if got, want := int(width), 1200; got != want {
		t.Fatalf("custom width = %d, want %d", got, want)
	}
	if got, want := int(height), 900; got != want {
		t.Fatalf("custom height = %d, want %d", got, want)
	}
}

func TestDefaultSizes(t *testing.T) {
	// Test that all chart types have default sizes defined
	chartTypes := []ChartType{
		ChartTypeDefault,
		ChartTypeSquare,
		ChartTypeCompact,
		ChartTypeWide,
	}

	for _, ct := range chartTypes {
		size, exists := defaultSizes[ct]
		if !exists {
			t.Errorf("no default size defined for chart type %d", ct)
			continue
		}

		if size[0] <= 0 || size[1] <= 0 {
			t.Errorf("invalid default size for chart type %d: [%f, %f]", ct, size[0], size[1])
		}
	}
}

// BenchmarkParsePlotSize benchmarks the parsing function
func BenchmarkParsePlotSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = ParsePlotSize("12,8", ChartTypeDefault)
	}
}

// BenchmarkGetPlotSize benchmarks the main function users will call
func BenchmarkGetPlotSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetPlotSize(ChartTypeDefault)
	}
}
