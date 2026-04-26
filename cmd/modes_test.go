package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"labours-go/internal/burndown"
	"labours-go/internal/readers"
)

type cliTestReader struct {
	developerStats []readers.DeveloperStat
}

func (r cliTestReader) Read(file io.Reader) error { return nil }
func (r cliTestReader) GetName() string           { return "test-repo" }
func (r cliTestReader) GetHeader() (int64, int64) { return 0, 0 }
func (r cliTestReader) GetProjectBurndown() (string, [][]int) {
	return "test-repo", [][]int{{1, 2}, {3, 4}}
}
func (r cliTestReader) GetBurndownParameters() (burndown.BurndownParameters, error) {
	return burndown.BurndownParameters{Sampling: 1, Granularity: 1, TickSize: 86400}, nil
}
func (r cliTestReader) GetProjectBurndownWithHeader() (burndown.BurndownHeader, string, [][]int, error) {
	return burndown.BurndownHeader{Start: 0, Last: 86400, Sampling: 1, Granularity: 1, TickSize: 86400}, "test-repo", [][]int{{1, 2}, {3, 4}}, nil
}
func (r cliTestReader) GetFilesBurndown() ([]readers.FileBurndown, error) {
	return nil, fmt.Errorf("missing files data")
}
func (r cliTestReader) GetPeopleBurndown() ([]readers.PeopleBurndown, error) {
	return nil, fmt.Errorf("missing people data")
}
func (r cliTestReader) GetOwnershipBurndown() ([]string, map[string][][]int, error) {
	return nil, nil, fmt.Errorf("missing people data")
}
func (r cliTestReader) GetPeopleInteraction() ([]string, [][]int, error) {
	return nil, nil, fmt.Errorf("missing people interaction")
}
func (r cliTestReader) GetFileCooccurrence() ([]string, [][]int, error) {
	return nil, nil, fmt.Errorf("missing couples data")
}
func (r cliTestReader) GetPeopleCooccurrence() ([]string, [][]int, error) {
	return nil, nil, fmt.Errorf("missing couples data")
}
func (r cliTestReader) GetShotnessCooccurrence() ([]string, [][]int, error) {
	return nil, nil, fmt.Errorf("missing shotness data")
}
func (r cliTestReader) GetShotnessRecords() ([]readers.ShotnessRecord, error) {
	return nil, fmt.Errorf("missing shotness data")
}
func (r cliTestReader) GetDeveloperStats() ([]readers.DeveloperStat, error) {
	if r.developerStats == nil {
		return nil, fmt.Errorf("missing Devs data")
	}
	return r.developerStats, nil
}
func (r cliTestReader) GetLanguageStats() ([]readers.LanguageStat, error) {
	return nil, fmt.Errorf("missing Devs data")
}
func (r cliTestReader) GetRuntimeStats() (map[string]float64, error) {
	return map[string]float64{"Burndown": 1.25}, nil
}
func (r cliTestReader) GetDeveloperTimeSeriesData() (*readers.DeveloperTimeSeriesData, error) {
	return nil, fmt.Errorf("missing Devs data")
}

func TestExecuteModesWithNoModesIsNoop(t *testing.T) {
	output := captureStdout(t, func() {
		executeModes(nil, cliTestReader{}, "", nil, nil)
	})
	if output != "" {
		t.Fatalf("executeModes() wrote %q, want no output", output)
	}
}

func TestExecuteModesPrintsPythonMissingDataWarning(t *testing.T) {
	previousQuiet := viper.GetBool("quiet")
	defer viper.Set("quiet", previousQuiet)
	viper.Set("quiet", true)

	output := captureStdout(t, func() {
		executeModes([]string{"devs"}, cliTestReader{}, filepath.Join(t.TempDir(), "devs.png"), nil, nil)
	})

	if !strings.Contains(output, "Devs stats were not collected. Re-run hercules with --devs.") {
		t.Fatalf("missing Python-compatible warning in output: %q", output)
	}
	if strings.Contains(output, "Error in mode devs") {
		t.Fatalf("missing data should not be reported as hard mode error: %q", output)
	}
}

func TestExecuteModesJSONWritesReaderData(t *testing.T) {
	previousQuiet := viper.GetBool("quiet")
	defer viper.Set("quiet", previousQuiet)
	viper.Set("quiet", true)

	outputPath := filepath.Join(t.TempDir(), "devs.json")
	reader := cliTestReader{
		developerStats: []readers.DeveloperStat{{
			Name:         "Ada",
			Commits:      3,
			LinesAdded:   10,
			LinesRemoved: 2,
		}},
	}

	executeModes([]string{"devs"}, reader, outputPath, nil, nil)

	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read JSON output: %v", err)
	}
	var payload struct {
		Results map[string]struct {
			DeveloperStats []readers.DeveloperStat `json:"developer_stats"`
			Message        string                  `json:"message"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, raw)
	}
	stats := payload.Results["devs"].DeveloperStats
	if len(stats) != 1 || stats[0].Name != "Ada" || stats[0].Commits != 3 {
		t.Fatalf("JSON output did not contain developer data: %#v", payload.Results["devs"])
	}
	if payload.Results["devs"].Message != "" {
		t.Fatalf("JSON output should not use placeholder message: %#v", payload.Results["devs"])
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = writePipe

	fn()

	if err := writePipe.Close(); err != nil {
		t.Fatalf("failed to close stdout pipe: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, readPipe); err != nil {
		t.Fatalf("failed to capture stdout: %v", err)
	}
	return buf.String()
}

func TestMissingAnalysisWarningClassifiesTypedErrors(t *testing.T) {
	warning, ok := missingAnalysisWarning("temporal-activity", fmt.Errorf("failed to get temporal activity data: %w", readers.ErrAnalysisMissing))
	if !ok {
		t.Fatal("expected typed missing analysis error to become a warning")
	}
	want := "Temporal activity stats were not collected. Re-run hercules with --temporal-activity."
	if warning != want {
		t.Fatalf("warning = %q, want %q", warning, want)
	}
}

func TestMissingAnalysisWarningDoesNotHideProcessingErrors(t *testing.T) {
	_, ok := missingAnalysisWarning("devs", fmt.Errorf("plot failed at %s", time.Unix(0, 0)))
	if ok {
		t.Fatal("unexpected warning classification for a non-missing processing error")
	}
}
