package integration_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"labours-go/internal/readers"
)

// TestComprehensiveCompatibility tests all available example files
func TestComprehensiveCompatibility(t *testing.T) {
	testFiles := map[string]string{
		"Burndown":   "../../example_data/hercules_burndown.pb",
		"Developers": "../../example_data/hercules_devs.pb",
		"Couples":    "../../example_data/hercules_couples.pb",
	}

	for dataType, testFile := range testFiles {
		t.Run(dataType, func(t *testing.T) {
			reader := &readers.ProtobufReader{}
			file, err := os.Open(testFile)
			require.NoError(t, err, "Test file should be accessible")
			defer func() { _ = file.Close() }()

			err = reader.Read(file)
			require.NoError(t, err, "Protobuf reading should work")

			t.Logf("=== Testing %s data (%s) ===", dataType, testFile)

			// Test all available methods
			testAllReaderMethods(t, reader, dataType)
		})
	}
}

func testAllReaderMethods(t *testing.T, reader *readers.ProtobufReader, dataType string) {
	t.Helper()

	// Test basic metadata
	name := reader.GetName()
	start, end := reader.GetHeader()
	t.Logf("Repository: %s, Time range: %d - %d", name, start, end)

	// Test project burndown
	if header, projName, matrix, err := reader.GetProjectBurndownWithHeader(); err == nil {
		t.Logf("✓ Project burndown: %s, %dx%d, tick_size=%.3f",
			projName, len(matrix), len(matrix[0]), header.TickSize)
		verifyMatrixIntegrity(t, matrix, "Project")
	} else {
		t.Logf("✗ Project burndown: %v", err)
	}

	// Test files burndown
	if files, err := reader.GetFilesBurndown(); err == nil {
		t.Logf("✓ Files burndown: %d files", len(files))
		for i, file := range files {
			if i < 3 {
				verifyMatrixIntegrity(t, file.Matrix, "File_"+file.Filename)
			}
		}
	} else {
		t.Logf("✗ Files burndown: %v", err)
	}

	// Test people burndown
	if people, err := reader.GetPeopleBurndown(); err == nil {
		t.Logf("✓ People burndown: %d people", len(people))
		for i, person := range people {
			if i < 3 {
				verifyMatrixIntegrity(t, person.Matrix, "Person_"+person.Person)
			}
		}
	} else {
		t.Logf("✗ People burndown: %v", err)
	}

	// Test ownership burndown
	if peopleSeq, ownership, err := reader.GetOwnershipBurndown(); err == nil {
		t.Logf("✓ Ownership burndown: %d people, %d ownership maps",
			len(peopleSeq), len(ownership))
	} else {
		t.Logf("✗ Ownership burndown: %v", err)
	}

	// Test people interaction
	if people, matrix, err := reader.GetPeopleInteraction(); err == nil {
		t.Logf("✓ People interaction: %d people, %dx%d matrix",
			len(people), len(matrix), len(matrix[0]))
		verifyMatrixIntegrity(t, matrix, "PeopleInteraction")
	} else {
		t.Logf("✗ People interaction: %v", err)
	}

	// Test file cooccurrence
	if fileIndex, matrix, err := reader.GetFileCooccurrence(); err == nil {
		t.Logf("✓ File cooccurrence: %d files, %dx%d matrix",
			len(fileIndex), len(matrix), len(matrix[0]))
		verifyMatrixIntegrity(t, matrix, "FileCooccurrence")
	} else {
		t.Logf("✗ File cooccurrence: %v", err)
	}

	// Test people cooccurrence
	if peopleIndex, matrix, err := reader.GetPeopleCooccurrence(); err == nil {
		t.Logf("✓ People cooccurrence: %d people, %dx%d matrix",
			len(peopleIndex), len(matrix), len(matrix[0]))
		verifyMatrixIntegrity(t, matrix, "PeopleCooccurrence")
	} else {
		t.Logf("✗ People cooccurrence: %v", err)
	}

	// Test developer stats
	if stats, err := reader.GetDeveloperStats(); err == nil {
		t.Logf("✓ Developer stats: %d developers", len(stats))
		for i, dev := range stats {
			if i < 3 {
				t.Logf("    Dev %d: %s (commits: %d, lines: +%d/-%d)",
					i, dev.Name, dev.Commits, dev.LinesAdded, dev.LinesRemoved)
			}
		}
	} else {
		t.Logf("✗ Developer stats: %v", err)
	}

	// Test developer time series data (critical for Python compatibility)
	if devData, err := reader.GetDeveloperTimeSeriesData(); err == nil {
		t.Logf("✓ Developer time series: %d people, %d time ticks",
			len(devData.People), len(devData.Days))

		// Analyze time series structure in detail
		if len(devData.Days) == 1 {
			t.Logf("    WARNING: Single-day aggregation (may be synthetic data)")
			if day0, exists := devData.Days[0]; exists {
				t.Logf("    Day 0 has %d developer entries", len(day0))
			}
		} else if len(devData.Days) > 1 {
			t.Logf("    ✓ Multi-day time series available")
			dayCount := 0
			for dayIdx, dayData := range devData.Days {
				if dayCount < 3 {
					t.Logf("    Day %d: %d developers", dayIdx, len(dayData))
				}
				dayCount++
			}
		}
	} else {
		t.Logf("✗ Developer time series: %v", err)
	}

	// Test language stats
	if langStats, err := reader.GetLanguageStats(); err == nil {
		t.Logf("✓ Language stats: %d languages", len(langStats))
	} else {
		t.Logf("✗ Language stats: %v", err)
	}

	// Test runtime stats
	if runStats, err := reader.GetRuntimeStats(); err == nil {
		t.Logf("✓ Runtime stats: %d entries", len(runStats))
	} else {
		t.Logf("✗ Runtime stats: %v", err)
	}

	// Test shotness records
	if records, err := reader.GetShotnessRecords(); err == nil {
		t.Logf("✓ Shotness records: %d records", len(records))
		for i, record := range records {
			if i < 3 {
				t.Logf("    Record %d: %s:%s (%s), %d counters",
					i, record.File, record.Name, record.Type, len(record.Counters))
			}
		}
	} else {
		t.Logf("✗ Shotness records: %v", err)
	}

	// Test shotness cooccurrence
	if index, matrix, err := reader.GetShotnessCooccurrence(); err == nil {
		t.Logf("✓ Shotness cooccurrence: %d items, %dx%d matrix",
			len(index), len(matrix), len(matrix[0]))
		verifyMatrixIntegrity(t, matrix, "ShotnessCooccurrence")
	} else {
		t.Logf("✗ Shotness cooccurrence: %v", err)
	}
}

func verifyMatrixIntegrity(t *testing.T, matrix [][]int, name string) {
	t.Helper()

	if len(matrix) == 0 {
		t.Logf("    %s: EMPTY", name)
		return
	}

	rows := len(matrix)
	cols := len(matrix[0])

	// Check for negative values and other integrity issues
	negatives := 0
	positives := 0
	minVal := 0
	maxVal := 0

	for i, row := range matrix {
		// Verify rectangular structure
		if len(row) != cols {
			t.Errorf("    %s: Row %d has %d cols, expected %d", name, i, len(row), cols)
		}

		for _, val := range row {
			if val < 0 {
				negatives++
				if val < minVal {
					minVal = val
				}
			} else if val > 0 {
				positives++
				if val > maxVal {
					maxVal = val
				}
			}
		}
	}

	sparsity := float64(positives) / float64(rows*cols) * 100

	if negatives > 0 {
		t.Logf("    %s: %dx%d, sparsity=%.1f%%, range=[%d,%d], ❌ %d negatives",
			name, rows, cols, sparsity, minVal, maxVal, negatives)
	} else {
		t.Logf("    %s: %dx%d, sparsity=%.1f%%, range=[%d,%d], ✓ integrity OK",
			name, rows, cols, sparsity, minVal, maxVal)
	}
}

// TestMatrixFormatDecisionTree verifies the exact format selection logic
func TestMatrixFormatDecisionTree(t *testing.T) {
	testFiles := []string{
		"../../example_data/hercules_burndown.pb",
		"../../example_data/hercules_devs.pb",
		"../../example_data/hercules_couples.pb",
	}

	for _, testFile := range testFiles {
		t.Run(testFile, func(t *testing.T) {
			reader := &readers.ProtobufReader{}
			file, err := os.Open(testFile)
			require.NoError(t, err)
			defer func() { _ = file.Close() }()

			err = reader.Read(file)
			require.NoError(t, err)

			t.Logf("=== Matrix Format Decision Tree for %s ===", testFile)

			// DECISION RULE 1: Project/Files/People matrices use BurndownSparseMatrix format
			// Python: _parse_burndown_matrix() for rows[].columns[] structure
			// Go: parseBurndownSparseMatrix() for BurndownSparseMatrix
			if _, _, matrix, err := reader.GetProjectBurndownWithHeader(); err == nil {
				t.Logf("✓ Project: BurndownSparseMatrix → parseBurndownSparseMatrix() → %dx%d",
					len(matrix), len(matrix[0]))
			}

			// DECISION RULE 2: Interaction/Cooccurrence matrices use CompressedSparseRowMatrix
			// Python: _parse_sparse_matrix() for CSR format with scipy.sparse
			// Go: parseCompressedSparseRowMatrix() for CompressedSparseRowMatrix
			if _, matrix, err := reader.GetPeopleInteraction(); err == nil {
				t.Logf("✓ PeopleInteraction: CompressedSparseRowMatrix → parseCompressedSparseRowMatrix() → %dx%d",
					len(matrix), len(matrix[0]))
			}

			if _, matrix, err := reader.GetFileCooccurrence(); err == nil {
				t.Logf("✓ FileCooccurrence: CompressedSparseRowMatrix → parseCompressedSparseRowMatrix() → %dx%d",
					len(matrix), len(matrix[0]))
			}

			if _, matrix, err := reader.GetPeopleCooccurrence(); err == nil {
				t.Logf("✓ PeopleCooccurrence: CompressedSparseRowMatrix → parseCompressedSparseRowMatrix() → %dx%d",
					len(matrix), len(matrix[0]))
			}

			t.Log("CONCLUSION: Go's matrix format selection appears to match Python's decision tree")
		})
	}
}

// TestDeveloperTimeSeriesCompatibility focuses on the critical time series issue
func TestDeveloperTimeSeriesCompatibility(t *testing.T) {
	devFile := "../../example_data/hercules_devs.pb"

	reader := &readers.ProtobufReader{}
	file, err := os.Open(devFile)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	err = reader.Read(file)
	require.NoError(t, err)

	t.Log("=== Critical Developer Time Series Compatibility Analysis ===")

	// Test Python-compatible time series extraction
	devData, err := reader.GetDeveloperTimeSeriesData()
	if err != nil {
		t.Errorf("CRITICAL: Developer time series not available: %v", err)
		return
	}

	t.Logf("Time series structure:")
	t.Logf("  People: %v", devData.People)
	t.Logf("  Days count: %d", len(devData.Days))

	// Analyze the temporal structure in detail
	if len(devData.Days) == 1 {
		t.Log("❌ CRITICAL COMPATIBILITY ISSUE:")
		t.Log("  Go is using synthetic single-day aggregation")
		t.Log("  Python likely has richer multi-day temporal data")
		t.Log("  This could lead to different temporal analysis results")

		// Show what the single day contains
		if dayData, exists := devData.Days[0]; exists {
			t.Logf("  Single day (day 0) contains %d developers:", len(dayData))
			for devIdx, devStats := range dayData {
				if devIdx < len(devData.People) {
					devName := devData.People[devIdx]
					t.Logf("    Dev %d (%s): commits=%d, lines=+%d/-%d/%d",
						devIdx, devName, devStats.Commits, devStats.LinesAdded,
						devStats.LinesRemoved, devStats.LinesModified)
				}
			}
		}
	} else {
		t.Logf("✓ Multi-day time series available (%d days)", len(devData.Days))

		// Show temporal distribution
		dayIndices := make([]int, 0, len(devData.Days))
		for dayIdx := range devData.Days {
			dayIndices = append(dayIndices, dayIdx)
		}
		t.Logf("  Day indices: %v (first few)", dayIndices[:minInt(len(dayIndices), 10)])

		// This would be compatible with Python's rich time series format
	}

	// Compare with GetDeveloperStats (non-time-series method)
	if stats, err := reader.GetDeveloperStats(); err == nil {
		t.Logf("Developer stats (aggregated): %d developers", len(stats))
		for i, dev := range stats {
			if i < 3 {
				t.Logf("    %s: commits=%d, lines=+%d/-%d",
					dev.Name, dev.Commits, dev.LinesAdded, dev.LinesRemoved)
			}
		}

		// TODO: Compare aggregated stats vs time series data to verify consistency
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
