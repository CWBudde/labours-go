package main

import (
	"fmt"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"labours-go/internal/pb"
)

// This utility creates sample test data files for testing
func main() {
	testDir := "test/testdata"

	// Ensure testdata directory exists
	err := os.MkdirAll(testDir, 0o755)
	if err != nil {
		fmt.Printf("Failed to create testdata directory: %v\n", err)
		os.Exit(1)
	}

	// Create simple burndown data
	fmt.Println("Creating simple burndown test data...")
	simpleBurndown := generateSimpleBurndownData()
	simpleData, err := proto.Marshal(simpleBurndown)
	if err != nil {
		fmt.Printf("Failed to serialize simple data: %v\n", err)
		os.Exit(1)
	}

	simplePath := filepath.Join(testDir, "simple_burndown.pb")
	err = os.WriteFile(simplePath, simpleData, 0o644)
	if err != nil {
		fmt.Printf("Failed to write simple data: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s (%d bytes)\n", simplePath, len(simpleData))

	// Create realistic burndown data
	fmt.Println("Creating realistic burndown test data...")
	realisticBurndown := generateRealisticBurndownData()
	realisticData, err := proto.Marshal(realisticBurndown)
	if err != nil {
		fmt.Printf("Failed to serialize realistic data: %v\n", err)
		os.Exit(1)
	}

	realisticPath := filepath.Join(testDir, "realistic_burndown.pb")
	err = os.WriteFile(realisticPath, realisticData, 0o644)
	if err != nil {
		fmt.Printf("Failed to write realistic data: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s (%d bytes)\n", realisticPath, len(realisticData))

	// Create a README for the test data
	readmePath := filepath.Join(testDir, "README.md")
	readme := `# Test Data Files

This directory contains sample hercules protobuf files for testing labours-go.

## Files

- **simple_burndown.pb**: Small-scale test data with sample project data
- **realistic_burndown.pb**: Large-scale test data with more comprehensive metrics
  
## Generated Data Characteristics

### Simple Burndown Data
- Basic project burndown matrix
- Simple file and people data
- Metadata with timestamps

### Realistic Burndown Data  
- More comprehensive burndown analysis
- Multiple developers and files
- Extended time range

## Usage in Tests

These files are used by:
- Unit tests for reader functionality
- Integration tests for end-to-end workflows
- Visual regression tests for chart consistency
- Performance benchmarks

## Regeneration

To regenerate this test data, run:
` + "```bash" + `
go run test/create_sample_data.go
` + "```" + `
`

	err = os.WriteFile(readmePath, []byte(readme), 0o644)
	if err != nil {
		fmt.Printf("Failed to write README: %v\n", err)
	} else {
		fmt.Printf("Created %s\n", readmePath)
	}

	fmt.Println("Sample test data creation completed successfully!")
}

func generateSimpleBurndownData() *pb.AnalysisResults {
	// Create burndown analysis data
	burndownData := &pb.BurndownAnalysisResults{
		Project: &pb.BurndownSparseMatrix{
			Name:            "test-project",
			NumberOfRows:    2,
			NumberOfColumns: 10,
			Rows: []*pb.BurndownSparseMatrixRow{
				{Columns: generateUint32Data(10)},
				{Columns: generateUint32Data(10)},
			},
		},
		Files: []*pb.BurndownSparseMatrix{
			{
				Name:            "main.go",
				NumberOfRows:    1,
				NumberOfColumns: 10,
				Rows:            []*pb.BurndownSparseMatrixRow{{Columns: generateUint32Data(10)}},
			},
			{
				Name:            "utils.go",
				NumberOfRows:    1,
				NumberOfColumns: 10,
				Rows:            []*pb.BurndownSparseMatrixRow{{Columns: generateUint32Data(10)}},
			},
			{
				Name:            "handler.go",
				NumberOfRows:    1,
				NumberOfColumns: 10,
				Rows:            []*pb.BurndownSparseMatrixRow{{Columns: generateUint32Data(10)}},
			},
		},
		People: []*pb.BurndownSparseMatrix{
			{
				Name:            "Alice",
				NumberOfRows:    1,
				NumberOfColumns: 10,
				Rows:            []*pb.BurndownSparseMatrixRow{{Columns: generateUint32Data(10)}},
			},
			{
				Name:            "Bob",
				NumberOfRows:    1,
				NumberOfColumns: 10,
				Rows:            []*pb.BurndownSparseMatrixRow{{Columns: generateUint32Data(10)}},
			},
			{
				Name:            "Charlie",
				NumberOfRows:    1,
				NumberOfColumns: 10,
				Rows:            []*pb.BurndownSparseMatrixRow{{Columns: generateUint32Data(10)}},
			},
		},
		FilesOwnership: []*pb.FilesOwnership{
			{
				Value: map[int32]int32{
					0: 0, // main.go -> Alice
					1: 1, // utils.go -> Bob
					2: 2, // handler.go -> Charlie
				},
			},
		},
		Granularity: 1,
		Sampling:    86400, // 1 day in seconds
	}

	// Marshal burndown data to bytes
	burndownBytes, err := proto.Marshal(burndownData)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal burndown data: %v", err))
	}

	return &pb.AnalysisResults{
		Header: &pb.Metadata{
			Repository:    "test-repository",
			BeginUnixTime: 1640995200, // 2022-01-01
			EndUnixTime:   1672531200, // 2023-01-01
			Version:       1,
		},
		Contents: map[string][]byte{
			"Burndown": burndownBytes,
		},
	}
}

func generateRealisticBurndownData() *pb.AnalysisResults {
	// Create realistic burndown analysis data with more files and people
	burndownData := &pb.BurndownAnalysisResults{
		Project: &pb.BurndownSparseMatrix{
			Name:            "realistic-project",
			NumberOfRows:    5,
			NumberOfColumns: 50,
			Rows: []*pb.BurndownSparseMatrixRow{
				{Columns: generateUint32Data(50)},
				{Columns: generateUint32Data(50)},
				{Columns: generateUint32Data(50)},
				{Columns: generateUint32Data(50)},
				{Columns: generateUint32Data(50)},
			},
		},
		Files:  generateMultipleFileBurndown(10),
		People: generateMultiplePeopleBurndown(5),
		FilesOwnership: []*pb.FilesOwnership{
			{
				Value: generateRealisticFileOwnership(10),
			},
		},
		Granularity: 1,
		Sampling:    86400, // 1 day in seconds
	}

	// Marshal burndown data to bytes
	burndownBytes, err := proto.Marshal(burndownData)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal burndown data: %v", err))
	}

	return &pb.AnalysisResults{
		Header: &pb.Metadata{
			Repository:    "realistic-repository",
			BeginUnixTime: 1640995200,
			EndUnixTime:   1672531200,
			Version:       1,
		},
		Contents: map[string][]byte{
			"Burndown": burndownBytes,
		},
	}
}

func generateSequentialData(count int) []int64 {
	data := make([]int64, count)
	for i := 0; i < count; i++ {
		data[i] = int64(1000 - i*5) // Decreasing pattern
		if data[i] < 0 {
			data[i] = 0
		}
	}
	return data
}

func generateSequentialIndices(count int) []int32 {
	indices := make([]int32, count)
	for i := 0; i < count; i++ {
		indices[i] = int32(i % 50) // Cycle through columns
	}
	return indices
}

func generateIndptr(rows, cols int) []int64 {
	indptr := make([]int64, rows+1)
	for i := 0; i <= rows; i++ {
		indptr[i] = int64(i * cols)
	}
	return indptr
}

func generateFileOwnership(numFiles int) map[string]int32 {
	ownership := make(map[string]int32)
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file_%d.go", i)
		ownership[filename] = int32(i % 5) // Assign to one of 5 people
	}
	return ownership
}

func generateFileNames(count int) []string {
	names := make([]string, count)
	for i := 0; i < count; i++ {
		names[i] = fmt.Sprintf("file_%d.go", i)
	}
	return names
}

// New helper functions for correct protobuf structure

func generateUint32Data(count int) []uint32 {
	data := make([]uint32, count)
	for i := 0; i < count; i++ {
		data[i] = uint32(1000 - i*5) // Decreasing pattern
		if data[i] > 10000 {         // Prevent overflow
			data[i] = 0
		}
	}
	return data
}

func generateMultipleFileBurndown(count int) []*pb.BurndownSparseMatrix {
	files := make([]*pb.BurndownSparseMatrix, count)
	for i := 0; i < count; i++ {
		files[i] = &pb.BurndownSparseMatrix{
			Name:            fmt.Sprintf("file_%d.go", i),
			NumberOfRows:    2,
			NumberOfColumns: 50,
			Rows: []*pb.BurndownSparseMatrixRow{
				{Columns: generateUint32Data(50)},
				{Columns: generateUint32Data(50)},
			},
		}
	}
	return files
}

func generateMultiplePeopleBurndown(count int) []*pb.BurndownSparseMatrix {
	names := []string{"Alice", "Bob", "Charlie", "Diana", "Eve"}
	people := make([]*pb.BurndownSparseMatrix, count)
	for i := 0; i < count; i++ {
		name := names[i%len(names)]
		people[i] = &pb.BurndownSparseMatrix{
			Name:            name,
			NumberOfRows:    2,
			NumberOfColumns: 50,
			Rows: []*pb.BurndownSparseMatrixRow{
				{Columns: generateUint32Data(50)},
				{Columns: generateUint32Data(50)},
			},
		}
	}
	return people
}

func generateRealisticFileOwnership(numFiles int) map[int32]int32 {
	ownership := make(map[int32]int32)
	for i := 0; i < numFiles; i++ {
		ownership[int32(i)] = int32(i % 5) // Assign to one of 5 people
	}
	return ownership
}
