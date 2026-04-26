package modes

import (
	"io"
	"os"
	"testing"

	"labours-go/internal/burndown"
	"labours-go/internal/readers"
)

// MockReader implements readers.Reader interface for testing sentiment mode
type MockSentimentReader struct{}

func (m *MockSentimentReader) Read(file io.Reader) error                            { return nil }
func (m *MockSentimentReader) GetName() string                                      { return "test" }
func (m *MockSentimentReader) GetHeader() (int64, int64)                            { return 0, 0 }
func (m *MockSentimentReader) GetProjectBurndown() (string, [][]int)                { return "", nil }
func (m *MockSentimentReader) GetFilesBurndown() ([]readers.FileBurndown, error)    { return nil, nil }
func (m *MockSentimentReader) GetPeopleBurndown() ([]readers.PeopleBurndown, error) { return nil, nil }
func (m *MockSentimentReader) GetOwnershipBurndown() ([]string, map[string][][]int, error) {
	return nil, nil, nil
}
func (m *MockSentimentReader) GetPeopleInteraction() ([]string, [][]int, error) { return nil, nil, nil }
func (m *MockSentimentReader) GetFileCooccurrence() ([]string, [][]int, error)  { return nil, nil, nil }
func (m *MockSentimentReader) GetPeopleCooccurrence() ([]string, [][]int, error) {
	return nil, nil, nil
}
func (m *MockSentimentReader) GetShotnessCooccurrence() ([]string, [][]int, error) {
	return nil, nil, nil
}
func (m *MockSentimentReader) GetShotnessRecords() ([]readers.ShotnessRecord, error) { return nil, nil }
func (m *MockSentimentReader) GetRuntimeStats() (map[string]float64, error)          { return nil, nil }
func (m *MockSentimentReader) GetBurndownParameters() (burndown.BurndownParameters, error) {
	return burndown.BurndownParameters{}, nil
}
func (m *MockSentimentReader) GetProjectBurndownWithHeader() (burndown.BurndownHeader, string, [][]int, error) {
	return burndown.BurndownHeader{}, "", nil, nil
}

func (m *MockSentimentReader) GetDeveloperStats() ([]readers.DeveloperStat, error) {
	return []readers.DeveloperStat{
		{
			Name:          "Alice",
			Commits:       50,
			LinesAdded:    1000,
			LinesRemoved:  100,
			LinesModified: 200,
			FilesTouched:  25,
			Languages:     map[string]int{"Go": 800, "Python": 200},
		},
		{
			Name:          "Bob",
			Commits:       30,
			LinesAdded:    500,
			LinesRemoved:  800,
			LinesModified: 100,
			FilesTouched:  15,
			Languages:     map[string]int{"JavaScript": 300, "CSS": 200},
		},
	}, nil
}

func (m *MockSentimentReader) GetDeveloperTimeSeriesData() (*readers.DeveloperTimeSeriesData, error) {
	return nil, nil
}

func (m *MockSentimentReader) GetLanguageStats() ([]readers.LanguageStat, error) {
	return []readers.LanguageStat{
		{Language: "Go", Lines: 800},
		{Language: "Python", Lines: 200},
		{Language: "JavaScript", Lines: 300},
		{Language: "CSS", Lines: 200},
	}, nil
}

func TestSentiment(t *testing.T) {
	// Create temporary output directory
	tempDir, err := os.MkdirTemp("", "sentiment_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create mock reader with test data
	reader := &MockSentimentReader{}

	// Run sentiment analysis
	err = Sentiment(reader, tempDir)
	if err != nil {
		t.Fatalf("Sentiment analysis failed: %v", err)
	}

	// Check that output files were created
	expectedFiles := []string{
		"sentiment-overview.png",
		"sentiment-overview.svg",
		"sentiment-developers.png",
		"sentiment-developers.svg",
		"sentiment-languages.png",
		"sentiment-languages.svg",
	}

	for _, filename := range expectedFiles {
		filepath := tempDir + "/" + filename
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			t.Errorf("Expected output file %s was not created", filename)
		}
	}
}

// NoDataReader implements readers.Reader but returns no data
type NoDataReader struct{}

func (n *NoDataReader) Read(file io.Reader) error                            { return nil }
func (n *NoDataReader) GetName() string                                      { return "test" }
func (n *NoDataReader) GetHeader() (int64, int64)                            { return 0, 0 }
func (n *NoDataReader) GetProjectBurndown() (string, [][]int)                { return "", nil }
func (n *NoDataReader) GetFilesBurndown() ([]readers.FileBurndown, error)    { return nil, nil }
func (n *NoDataReader) GetPeopleBurndown() ([]readers.PeopleBurndown, error) { return nil, nil }
func (n *NoDataReader) GetOwnershipBurndown() ([]string, map[string][][]int, error) {
	return nil, nil, nil
}
func (n *NoDataReader) GetPeopleInteraction() ([]string, [][]int, error)      { return nil, nil, nil }
func (n *NoDataReader) GetFileCooccurrence() ([]string, [][]int, error)       { return nil, nil, nil }
func (n *NoDataReader) GetPeopleCooccurrence() ([]string, [][]int, error)     { return nil, nil, nil }
func (n *NoDataReader) GetShotnessCooccurrence() ([]string, [][]int, error)   { return nil, nil, nil }
func (n *NoDataReader) GetShotnessRecords() ([]readers.ShotnessRecord, error) { return nil, nil }
func (n *NoDataReader) GetRuntimeStats() (map[string]float64, error)          { return nil, nil }
func (n *NoDataReader) GetBurndownParameters() (burndown.BurndownParameters, error) {
	return burndown.BurndownParameters{}, nil
}
func (n *NoDataReader) GetProjectBurndownWithHeader() (burndown.BurndownHeader, string, [][]int, error) {
	return burndown.BurndownHeader{}, "", nil, nil
}
func (n *NoDataReader) GetDeveloperStats() ([]readers.DeveloperStat, error) { return nil, nil }
func (n *NoDataReader) GetLanguageStats() ([]readers.LanguageStat, error)   { return nil, nil }
func (n *NoDataReader) GetDeveloperTimeSeriesData() (*readers.DeveloperTimeSeriesData, error) {
	return nil, nil
}

func TestSentimentWithNoData(t *testing.T) {
	// Create a mock reader with no data
	noDataReader := &NoDataReader{}

	// Create temp dir
	tempDir, err := os.MkdirTemp("", "sentiment_no_data_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// This should return an error when no data is available
	err = Sentiment(noDataReader, tempDir)
	if err == nil {
		t.Error("Expected error when no sentiment data is available, but got nil")
	}
}

type ZeroActivitySentimentReader struct {
	*NoDataReader
}

func (z *ZeroActivitySentimentReader) GetDeveloperStats() ([]readers.DeveloperStat, error) {
	return []readers.DeveloperStat{
		{Name: "Alice"},
		{Name: "Bob"},
	}, nil
}

func (z *ZeroActivitySentimentReader) GetLanguageStats() ([]readers.LanguageStat, error) {
	return nil, nil
}

func TestSentimentWithZeroActivityDoesNotCreateNaNBars(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sentiment_zero_activity_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	reader := &ZeroActivitySentimentReader{NoDataReader: &NoDataReader{}}
	if err := Sentiment(reader, tempDir); err != nil {
		t.Fatalf("Sentiment analysis with zero activity failed: %v", err)
	}

	if _, err := os.Stat(tempDir + "/sentiment-overview.png"); os.IsNotExist(err) {
		t.Error("Expected sentiment overview output for zero activity data")
	}
}
