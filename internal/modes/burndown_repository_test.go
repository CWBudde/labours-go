package modes

import (
	"testing"

	"labours-go/internal/readers"
)

func TestCombineRepositoryBurndowns(t *testing.T) {
	combined := combineRepositoryBurndowns([]readers.RepositoryBurndown{
		{
			Repository: "repo-a",
			Matrix: [][]int{
				{1, 2},
				{3, 4},
			},
		},
		{
			Repository: "repo-b",
			Matrix: [][]int{
				{10},
				{20, 30},
				{40, 50},
			},
		},
	})

	expected := [][]int{
		{11, 2},
		{23, 34},
		{40, 50},
	}
	if len(combined) != len(expected) {
		t.Fatalf("combined rows = %d, want %d", len(combined), len(expected))
	}
	for i := range expected {
		if len(combined[i]) != len(expected[i]) {
			t.Fatalf("combined row %d columns = %d, want %d", i, len(combined[i]), len(expected[i]))
		}
		for j := range expected[i] {
			if combined[i][j] != expected[i][j] {
				t.Fatalf("combined[%d][%d] = %d, want %d", i, j, combined[i][j], expected[i][j])
			}
		}
	}
}
