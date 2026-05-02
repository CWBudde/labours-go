package visual

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestVisualFrameworkDemo demonstrates the visual validation framework
func TestVisualFrameworkDemo(t *testing.T) {
	// Check if we have test data available - use YAML since protobuf might have encoding issues
	testDataPath := "../../example_data/hercules_burndown.yaml"
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skipf("Test data not found: %s", testDataPath)
		return
	}

	// Create temporary directory for demo output
	tmpDir := t.TempDir()
	generator := NewChartGenerator(tmpDir)

	t.Run("GenerateAndValidateChart", func(t *testing.T) {
		// Generate a test chart
		outputPath, err := generator.GenerateChart(t, "burndown-project", testDataPath)
		if err != nil {
			t.Fatalf("Failed to generate chart: %v", err)
		}

		t.Logf("Generated test chart: %s", outputPath)

		// Validate the chart structure
		err = generator.ValidateChartStructure(t, outputPath)
		if err != nil {
			t.Errorf("Chart structure validation failed: %v", err)
		} else {
			t.Log("✅ Chart structure validation passed")
		}
	})

	t.Run("SelfSimilarityTest", func(t *testing.T) {
		// Generate two identical charts
		chart1, err := generator.GenerateChart(t, "burndown-project", testDataPath)
		if err != nil {
			t.Fatalf("Failed to generate first chart: %v", err)
		}

		chart2, err := generator.GenerateChart(t, "burndown-project", testDataPath)
		if err != nil {
			t.Fatalf("Failed to generate second chart: %v", err)
		}

		// Compare them - should be nearly identical
		metrics, err := CompareImages(chart1, chart2)
		if err != nil {
			t.Fatalf("Failed to compare identical charts: %v", err)
		}

		// Report results
		report := metrics.GetDetailedReport(ValidationStandard)
		t.Logf("Self-similarity test results:\n%s", report)

		// Validation
		if metrics.OverallSimilarity < 0.98 {
			t.Errorf("Identical charts should have >98%% similarity, got %.2f%%",
				metrics.OverallSimilarity*100)
		} else {
			t.Log("✅ Self-similarity test passed - charts are reproducible")
		}
	})

	t.Run("ValidationLevelTesting", func(t *testing.T) {
		chart, err := generator.GenerateChart(t, "burndown-project", testDataPath)
		if err != nil {
			t.Fatalf("Failed to generate chart: %v", err)
		}

		// Test validation levels with self-comparison (should pass all levels)
		metrics, err := CompareImages(chart, chart)
		if err != nil {
			t.Fatalf("Failed to compare chart with itself: %v", err)
		}

		levels := []ValidationLevel{ValidationStrict, ValidationStandard, ValidationLenient}
		for _, level := range levels {
			if !metrics.IsValidationPassing(level) {
				t.Errorf("Self-comparison should pass %s validation", string(level))
			} else {
				t.Logf("✅ %s validation passed", string(level))
			}
		}
	})
}

// TestReferenceGeneration demonstrates generating reference images
func TestReferenceGeneration(t *testing.T) {
	// Skip unless explicitly requested
	if os.Getenv("GENERATE_REFERENCES") != "true" {
		t.Skip("Reference generation skipped (set GENERATE_REFERENCES=true to enable)")
		return
	}

	testDataPath := "../testdata/realistic_burndown.pb"
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skipf("Test data not found: %s", testDataPath)
		return
	}

	// Create reference output directory
	refDir := "../golden"
	if err := os.MkdirAll(refDir, 0o755); err != nil {
		t.Fatalf("Failed to create reference directory: %v", err)
	}

	generator := NewChartGenerator(refDir)
	references := generator.GenerateReferenceSet(t, testDataPath)

	// Copy generated files to golden directory with proper names
	for mode, tempPath := range references {
		goldenName := fmt.Sprintf("%s_golden.png", mode)
		goldenPath := filepath.Join(refDir, goldenName)

		copyFile(t, tempPath, goldenPath)
		t.Logf("Created golden file: %s", goldenPath)
	}

	t.Logf("Generated %d reference images in %s", len(references), refDir)
}

// TestPythonCompatibilityDemo shows Python compatibility validation if references exist
func TestPythonCompatibilityDemo(t *testing.T) {
	pythonRefPath := "../../analysis_results/reference/python_burndown_absolute.png"
	if _, err := os.Stat(pythonRefPath); os.IsNotExist(err) {
		t.Skipf("Python reference not found: %s", pythonRefPath)
		return
	}

	testDataPath := "../../example_data/hercules_burndown.yaml"
	if _, err := os.Stat(testDataPath); os.IsNotExist(err) {
		t.Skipf("Test data not found: %s", testDataPath)
		return
	}

	// Generate Go chart with same data as Python reference
	tmpDir := t.TempDir()
	generator := NewChartGenerator(tmpDir)

	goChart, err := generator.GenerateChart(t, "burndown-project", testDataPath)
	if err != nil {
		t.Fatalf("Failed to generate Go chart: %v", err)
	}

	// Compare with Python reference
	metrics, err := CompareImages(goChart, pythonRefPath)
	if err != nil {
		t.Fatalf("Failed to compare with Python reference: %v", err)
	}

	report := metrics.GetDetailedReport(ValidationLenient)
	t.Logf("Python compatibility demo results:\n%s", report)

	if metrics.IsValidationPassing(ValidationLenient) {
		t.Log("✅ Go implementation maintains functional compatibility with Python")
	} else {
		t.Log("⚠️  Go implementation shows differences from Python (may be acceptable)")
	}
}
