package modes

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"labours-go/internal/burndown"
	"labours-go/internal/readers"
)

func TestReportMetricModesCreateOutputFiles(t *testing.T) {
	reader := &reportMetricsReader{}
	tests := []struct {
		name string
		run  func(string) error
	}{
		{
			name: "temporal-activity",
			run: func(output string) error {
				return TemporalActivity(reader, output, 32, 10)
			},
		},
		{
			name: "bus-factor",
			run: func(output string) error {
				return BusFactor(reader, output)
			},
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
		},
		{
			name: "hotspot-risk",
			run: func(output string) error {
				return HotspotRisk(reader, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), tt.name+".png")
			if err := tt.run(output); err != nil {
				t.Fatalf("%s() unexpected error: %v", tt.name, err)
			}
			if info, err := os.Stat(output); err != nil {
				t.Fatalf("expected output file %s: %v", output, err)
			} else if info.Size() == 0 {
				t.Fatalf("expected non-empty output file %s", output)
			}
		})
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
	return nil, nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetShotnessRecords() ([]readers.ShotnessRecord, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetDeveloperStats() ([]readers.DeveloperStat, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetLanguageStats() ([]readers.LanguageStat, error) {
	return nil, fmt.Errorf("not implemented")
}
func (r *reportMetricsReader) GetRuntimeStats() (map[string]float64, error) {
	return nil, fmt.Errorf("not implemented")
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
			"main.go": {UniqueEditors: 2, RecentEditors: 1},
			"doc.md":  {UniqueEditors: 1, RecentEditors: 1},
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
