package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataFileGeneration(t *testing.T) {
	// Test that our test data generation tool works
	testDataPath := filepath.Join("..", "testdata", "simple_burndown.pb")

	// Check if test data exists
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skipf("Test data file not found: %s - run 'go run test/create_sample_data.go' first", testDataPath)
	}

	// Check file size
	info, err := os.Stat(testDataPath)
	if err != nil {
		t.Fatalf("Failed to stat test data file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Test data file is empty")
	}

	t.Logf("Test data file exists and is %d bytes", info.Size())
}

func TestRealisticDataFileGeneration(t *testing.T) {
	testDataPath := filepath.Join("..", "testdata", "realistic_burndown.pb")

	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skipf("Realistic test data file not found: %s", testDataPath)
	}

	info, err := os.Stat(testDataPath)
	if err != nil {
		t.Fatalf("Failed to stat realistic test data file: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Realistic test data file is empty")
	}

	// Realistic data should be larger than simple data
	if info.Size() < 1000 {
		t.Errorf("Realistic test data seems too small: %d bytes", info.Size())
	}

	t.Logf("Realistic test data file exists and is %d bytes", info.Size())
}

func TestTestDataReadme(t *testing.T) {
	readmePath := filepath.Join("..", "testdata", "README.md")

	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Skipf("README file not found: %s", readmePath)
	}

	content, err := os.ReadFile(readmePath) // #nosec G304 - test fixture path is fixed above.
	if err != nil {
		t.Fatalf("Failed to read README: %v", err)
	}

	if len(content) == 0 {
		t.Error("README file is empty")
	}

	// Check that README contains expected sections
	readmeText := string(content)
	expectedSections := []string{
		"# Test Data Files",
		"simple_burndown.pb",
		"realistic_burndown.pb",
	}

	for _, section := range expectedSections {
		if !contains(readmeText, section) {
			t.Errorf("README should contain '%s'", section)
		}
	}

	t.Logf("README file exists and contains expected sections")
}

func TestDirectoryStructure(t *testing.T) {
	// Test that our test directory structure is correct
	expectedPaths := []string{
		filepath.Join("..", "testdata"),
		filepath.Join("..", "testdata", "simple_burndown.pb"),
		filepath.Join("..", "testdata", "realistic_burndown.pb"),
		filepath.Join("..", "testdata", "README.md"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected path does not exist: %s", path)
		} else {
			t.Logf("✓ Path exists: %s", path)
		}
	}
}

func TestDataFilesSizeDifference(t *testing.T) {
	simplePath := filepath.Join("..", "testdata", "simple_burndown.pb")
	realisticPath := filepath.Join("..", "testdata", "realistic_burndown.pb")

	simpleInfo, err := os.Stat(simplePath)
	if err != nil {
		t.Skipf("Simple data file not found: %v", err)
	}

	realisticInfo, err := os.Stat(realisticPath)
	if err != nil {
		t.Skipf("Realistic data file not found: %v", err)
	}

	if realisticInfo.Size() <= simpleInfo.Size() {
		t.Errorf("Realistic data (%d bytes) should be larger than simple data (%d bytes)",
			realisticInfo.Size(), simpleInfo.Size())
	}

	ratio := float64(realisticInfo.Size()) / float64(simpleInfo.Size())
	t.Logf("Realistic data is %.2fx larger than simple data", ratio)
}

// Helper function
func contains(text, substring string) bool {
	return len(text) > 0 && len(substring) > 0 &&
		len(text) >= len(substring) &&
		findSubstring(text, substring)
}

func findSubstring(text, substring string) bool {
	for i := 0; i <= len(text)-len(substring); i++ {
		if text[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}
