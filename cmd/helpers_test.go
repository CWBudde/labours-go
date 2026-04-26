package cmd

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

func TestResolveModesFromPythonCompatibleAliases(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "burndown alias",
			input: []string{"burndown"},
			want:  []string{"burndown-project"},
		},
		{
			name:  "couples alias",
			input: []string{"couples"},
			want:  []string{"couples-files", "couples-people", "couples-shotness"},
		},
		{
			name:  "comma separated modes",
			input: []string{"burndown-project,devs"},
			want:  []string{"burndown-project", "devs"},
		},
		{
			name:  "python all",
			input: []string{"all"},
			want:  pythonAllModes,
		},
		{
			name:  "known unimplemented report mode is valid",
			input: []string{"temporal-activity"},
			want:  []string{"temporal-activity"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveModesFrom(tt.input)
			if err != nil {
				t.Fatalf("resolveModesFrom() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("resolveModesFrom() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestResolveModesFromRejectsUnknownMode(t *testing.T) {
	_, err := resolveModesFrom([]string{"not-a-mode"})
	if err == nil {
		t.Fatal("expected unknown mode error")
	}
}

func TestResolveModesFromRejectsEmptyModes(t *testing.T) {
	_, err := resolveModesFrom(nil)
	if err == nil {
		t.Fatal("expected empty mode error")
	}
}

func TestNormalizeInputFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "", want: "auto"},
		{input: "AUTO", want: "auto"},
		{input: "yaml", want: "yaml"},
		{input: "pb", want: "pb"},
		{input: "json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeInputFormat(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeInputFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseFlexibleDateAcceptsCommonPythonCompatibleForms(t *testing.T) {
	for _, date := range []string{
		"2024-01-02",
		"January 2, 2024",
		"2024-01-02T15:04:05Z",
	} {
		t.Run(date, func(t *testing.T) {
			if _, err := parseFlexibleDate(date); err != nil {
				t.Fatalf("parseFlexibleDate() unexpected error: %v", err)
			}
		})
	}
}

func TestDetectOutputFormatTreatsAggAsRenderingBackend(t *testing.T) {
	previousBackend := viper.GetString("backend")
	defer viper.Set("backend", previousBackend)

	viper.Set("backend", "Agg")
	if got := detectOutputFormat("chart.svg"); got != "svg" {
		t.Fatalf("detectOutputFormat() = %q, want svg", got)
	}
	if got := detectOutputFormat("chart"); got != "png" {
		t.Fatalf("detectOutputFormat() = %q, want png", got)
	}
}

func TestPythonCompatibleFlagsAreRegistered(t *testing.T) {
	for _, name := range []string{
		"mode",
		"temporal-legend-threshold",
		"temporal-legend-single-col-threshold",
	} {
		if rootCmd.PersistentFlags().Lookup(name) == nil {
			t.Fatalf("expected flag %q to be registered", name)
		}
	}
}

func TestPlanModeOutputSingleMode(t *testing.T) {
	previousBackend := viper.GetString("backend")
	defer viper.Set("backend", previousBackend)
	viper.Set("backend", "auto")

	tmpDir := t.TempDir()
	tests := []struct {
		name       string
		baseOutput string
		mode       string
		modeCount  int
		want       string
	}{
		{
			name:       "file path is preserved",
			baseOutput: filepath.Join(tmpDir, "chart.svg"),
			mode:       "devs",
			modeCount:  1,
			want:       filepath.Join(tmpDir, "chart.svg"),
		},
		{
			name:       "directory output receives mode filename",
			baseOutput: tmpDir,
			mode:       "devs",
			modeCount:  1,
			want:       filepath.Join(tmpDir, "devs.png"),
		},
		{
			name:       "extensionless single output is file base",
			baseOutput: filepath.Join(tmpDir, "chart"),
			mode:       "devs",
			modeCount:  1,
			want:       filepath.Join(tmpDir, "chart.png"),
		},
		{
			name:       "empty output receives mode filename",
			baseOutput: "",
			mode:       "devs",
			modeCount:  1,
			want:       "devs.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := planModeOutput(tt.baseOutput, tt.mode, tt.modeCount); got != tt.want {
				t.Fatalf("planModeOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlanModeOutputMultipleModes(t *testing.T) {
	previousBackend := viper.GetString("backend")
	defer viper.Set("backend", previousBackend)
	viper.Set("backend", "auto")

	tmpDir := t.TempDir()
	tests := []struct {
		name       string
		baseOutput string
		mode       string
		want       string
	}{
		{
			name:       "directory base gets per-mode file",
			baseOutput: tmpDir,
			mode:       "devs",
			want:       filepath.Join(tmpDir, "devs.png"),
		},
		{
			name:       "extensionless base is directory in multi-mode",
			baseOutput: filepath.Join(tmpDir, "charts"),
			mode:       "languages",
			want:       filepath.Join(tmpDir, "charts", "languages.png"),
		},
		{
			name:       "file base contributes extension and parent directory",
			baseOutput: filepath.Join(tmpDir, "report.svg"),
			mode:       "ownership",
			want:       filepath.Join(tmpDir, "ownership.svg"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := planModeOutput(tt.baseOutput, tt.mode, 2); got != tt.want {
				t.Fatalf("planModeOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPlanModeOutputMultiAssetModes(t *testing.T) {
	tmpDir := t.TempDir()
	tests := []struct {
		name       string
		baseOutput string
		mode       string
		want       string
	}{
		{
			name:       "file output means write assets next to requested file",
			baseOutput: filepath.Join(tmpDir, "couples-files.png"),
			mode:       "couples-files",
			want:       tmpDir,
		},
		{
			name:       "directory output is passed through",
			baseOutput: tmpDir,
			mode:       "shotness",
			want:       tmpDir,
		},
		{
			name:       "empty output defaults to current directory",
			baseOutput: "",
			mode:       "sentiment",
			want:       ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := planModeOutput(tt.baseOutput, tt.mode, 1); got != tt.want {
				t.Fatalf("planModeOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}
