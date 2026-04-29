package modes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"labours-go/internal/burndown"
	"labours-go/internal/readers"
)

func TestReportMetricModesCreateOutputFiles(t *testing.T) {
	reader := &reportMetricsReader{}
	tests := []struct {
		name   string
		run    func(string) error
		extras []string
	}{
		{
			name: "temporal-activity",
			run: func(output string) error {
				return TemporalActivity(reader, output, 32, 10, nil, nil)
			},
		},
		{
			name: "bus-factor",
			run: func(output string) error {
				return BusFactor(reader, output)
			},
			extras: []string{"bus-factor_subsystems.png"},
		},
		{
			name: "ownership-concentration",
			run: func(output string) error {
				return OwnershipConcentration(reader, output)
			},
		},
		{
			name: "knowledge-diffusion",
			run: func(output string) error {
				return KnowledgeDiffusion(reader, output)
			},
			extras: []string{"knowledge-diffusion_silos.png", "knowledge-diffusion_trend.png"},
		},
		{
			name: "hotspot-risk",
			run: func(output string) error {
				return HotspotRisk(reader, output)
			},
			extras: []string{"hotspot-risk_table.tsv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, tt.name+".png")
			if err := tt.run(output); err != nil {
				t.Fatalf("%s() unexpected error: %v", tt.name, err)
			}
			assertNonEmptyFile(t, output)
			for _, extra := range tt.extras {
				assertNonEmptyFile(t, filepath.Join(dir, extra))
			}
		})
	}
}

func TestReportMetricModesCreateSVGOutputFiles(t *testing.T) {
	reader := &reportMetricsReader{}
	tests := []struct {
		name   string
		run    func(string) error
		extras []string
	}{
		{
			name: "temporal-activity",
			run: func(output string) error {
				return TemporalActivity(reader, output, 32, 10, nil, nil)
			},
		},
		{
			name: "bus-factor",
			run: func(output string) error {
				return BusFactor(reader, output)
			},
			extras: []string{"bus-factor_subsystems.svg"},
		},
		{
			name: "ownership-concentration",
			run: func(output string) error {
				return OwnershipConcentration(reader, output)
			},
		},
		{
			name: "knowledge-diffusion",
			run: func(output string) error {
				return KnowledgeDiffusion(reader, output)
			},
			extras: []string{"knowledge-diffusion_silos.svg", "knowledge-diffusion_trend.svg"},
		},
		{
			name: "hotspot-risk",
			run: func(output string) error {
				return HotspotRisk(reader, output)
			},
			extras: []string{"hotspot-risk_table.tsv"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			output := filepath.Join(dir, tt.name+".svg")
			if err := tt.run(output); err != nil {
				t.Fatalf("%s() unexpected error: %v", tt.name, err)
			}
			assertNonEmptyFile(t, output)
			for _, extra := range tt.extras {
				assertNonEmptyFile(t, filepath.Join(dir, extra))
			}
		})
	}
}

func TestDirectoryChartModesCreatePNGAndSVGAssets(t *testing.T) {
	previousQuiet := viper.GetBool("quiet")
	defer viper.Set("quiet", previousQuiet)
	viper.Set("quiet", true)

	reader := &reportMetricsReader{}
	tests := []struct {
		name  string
		run   func(string) error
		files []string
	}{
		{
			name: "couples-shotness",
			run: func(output string) error {
				return CouplesShotness(reader, output)
			},
			files: []string{
				"shotness_coupling_heatmap.png",
				"shotness_coupling_heatmap.svg",
				"top_shotness_coupling_pairs.png",
				"top_shotness_coupling_pairs.svg",
			},
		},
		{
			name: "devs-efforts",
			run: func(output string) error {
				return DevsEfforts(reader, output, 20)
			},
			files: []string{
				"devs_efforts_scatter.png",
				"devs_efforts_scatter.svg",
				"devs_productivity_ranking.png",
				"devs_productivity_ranking.svg",
			},
		},
		{
			name: "run-times",
			run: func(output string) error {
				return RunTimes(reader, output)
			},
			files: []string{
				"runtime_breakdown.png",
				"runtime_breakdown.svg",
				"runtime_percentage.png",
				"runtime_percentage.svg",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tt.run(dir); err != nil {
				t.Fatalf("%s() unexpected error: %v", tt.name, err)
			}
			for _, file := range tt.files {
				assertNonEmptyFile(t, filepath.Join(dir, file))
			}
		})
	}
}

func TestTopStringIntPairsPreservesPathLabels(t *testing.T) {
	labels, values := topStringIntPairs(map[string]int{
		"/":                               1,
		"cmd/hercules":                    1,
		"vendor/github.com/jeffail/tunny": 2,
	}, 0, false)

	expectedLabels := []string{"/", "cmd/hercules", "vendor/github.com/jeffail/tunny"}
	expectedValues := []int{1, 1, 2}
	if len(labels) != len(expectedLabels) {
		t.Fatalf("got %d labels, want %d: %v", len(labels), len(expectedLabels), labels)
	}
	for i := range expectedLabels {
		if labels[i] != expectedLabels[i] {
			t.Fatalf("label %d = %q, want %q", i, labels[i], expectedLabels[i])
		}
		if values[i] != expectedValues[i] {
			t.Fatalf("value %d = %d, want %d", i, values[i], expectedValues[i])
		}
	}
}

func assertNonEmptyFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected output file %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("expected non-empty output file %s", path)
	}
}

type reportMetricsReader struct{}

func (r *reportMetricsReader) Read(file io.Reader) error { return nil }
func (r *reportMetricsReader) GetName() string           { return "report-metrics" }
func (r *reportMetricsReader) GetHeader() (int64, int64) { return 0, 0 }
func (r *reportMetricsReader) GetProjectBurndown() (string, [][]int) {
	return "", nil
}
func (r *reportMetricsReader) GetBurndownParameters() (burndown.BurndownParameters, error) {
	return burndown.BurndownParameters{}, nil
}
func (r *reportMetricsReader) GetProjectBurndownWithHeader() (burndown.BurndownHeader, string, [][]int, error) {
	return burndown.BurndownHeader{}, "", nil, nil
}
func (r *reportMetricsReader) GetFilesBurndown() ([]readers.FileBurndown, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetPeopleBurndown() ([]readers.PeopleBurndown, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetOwnershipBurndown() ([]string, map[string][][]int, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetPeopleInteraction() ([]string, [][]int, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetFileCooccurrence() ([]string, [][]int, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetPeopleCooccurrence() ([]string, [][]int, error) {
	return nil, nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetShotnessCooccurrence() ([]string, [][]int, error) {
	return []string{"main.go:funcA", "main.go:funcB", "doc.md:section"}, [][]int{
		{4, 3, 1},
		{3, 5, 2},
		{1, 2, 2},
	}, nil
}
func (r *reportMetricsReader) GetShotnessRecords() ([]readers.ShotnessRecord, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetDeveloperStats() ([]readers.DeveloperStat, error) {
	return []readers.DeveloperStat{
		{Name: "alice", Commits: 10, LinesAdded: 100, LinesRemoved: 20, LinesModified: 30, FilesTouched: 4},
		{Name: "bob", Commits: 5, LinesAdded: 60, LinesRemoved: 10, LinesModified: 15, FilesTouched: 2},
	}, nil
}
func (r *reportMetricsReader) GetLanguageStats() ([]readers.LanguageStat, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetRuntimeStats() (map[string]float64, error) {
	return map[string]float64{
		"burndown": 20,
		"couples":  10,
		"devs":     5,
	}, nil
}
func (r *reportMetricsReader) GetDeveloperTimeSeriesData() (*readers.DeveloperTimeSeriesData, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *reportMetricsReader) GetTemporalActivity() (*readers.TemporalActivityData, error) {
	return &readers.TemporalActivityData{
		People: []string{"alice"},
		Activities: map[int]readers.TemporalDeveloperActivity{
			0: {
				Hours: readers.TemporalDimensionData{
					Commits: []int{0, 2, 3},
					Lines:   []int{0, 20, 30},
				},
			},
		},
		Ticks: map[int]map[int]readers.TemporalActivityTick{
			0: {0: {Commits: 2, Lines: 20, Hour: 1}},
			1: {0: {Commits: 3, Lines: 30, Hour: 2}},
		},
		TickSize: int64(24 * 60 * 60 * 1_000_000_000),
	}, nil
}

func (r *reportMetricsReader) GetBusFactor() (*readers.BusFactorData, error) {
	return &readers.BusFactorData{
		People: []string{"alice", "bob"},
		Snapshots: map[int]readers.BusFactorSnapshot{
			0: {BusFactor: 1, TotalLines: 100},
			1: {BusFactor: 2, TotalLines: 120},
		},
		SubsystemBusFactor: map[string]int{"core": 1},
		Threshold:          0.8,
	}, nil
}

func (r *reportMetricsReader) GetOwnershipConcentration() (*readers.OwnershipConcentrationData, error) {
	return &readers.OwnershipConcentrationData{
		People: []string{"alice", "bob"},
		Snapshots: map[int]readers.OwnershipConcentrationSnapshot{
			0: {Gini: 0.2, HHI: 0.4, TotalLines: 100},
			1: {Gini: 0.3, HHI: 0.5, TotalLines: 120},
		},
		SubsystemGini: map[string]float64{"core": 0.2},
		SubsystemHHI:  map[string]float64{"core": 0.4},
	}, nil
}

func (r *reportMetricsReader) GetKnowledgeDiffusion() (*readers.KnowledgeDiffusionData, error) {
	return &readers.KnowledgeDiffusionData{
		People: []string{"alice", "bob"},
		Files: map[string]readers.KnowledgeDiffusionFile{
			"main.go": {UniqueEditors: 2, RecentEditors: 1, UniqueEditorsOverTime: map[int]int{0: 1, 1: 2}},
			"doc.md":  {UniqueEditors: 1, RecentEditors: 1, UniqueEditorsOverTime: map[int]int{0: 1}},
		},
		Distribution: map[int]int{1: 1, 2: 1},
		WindowMonths: 6,
	}, nil
}

func (r *reportMetricsReader) GetHotspotRisk() (*readers.HotspotRiskData, error) {
	return &readers.HotspotRiskData{
		WindowDays: 90,
		Files: []readers.HotspotRiskFile{
			{Path: "main.go", RiskScore: 0.9, Size: 100, Churn: 20},
			{Path: "doc.md", RiskScore: 0.2, Size: 10, Churn: 1},
		},
	}, nil
}
