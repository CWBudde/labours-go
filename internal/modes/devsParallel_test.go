package modes

import (
	"errors"
	"testing"

	"labours-go/internal/readers"
)

func TestDevsParallelRequiresPeopleBurndownByDefault(t *testing.T) {
	err := DevsParallel(&NoDataReader{}, t.TempDir(), 20, false)
	if !errors.Is(err, readers.ErrAnalysisMissing) {
		t.Fatalf("Expected missing-analysis error when people burndown is absent, got %v", err)
	}
}

func TestFilterPeopleBurndownByActivityRespectsMaxPeople(t *testing.T) {
	people := []readers.PeopleBurndown{
		{Person: "low", Matrix: [][]int{{1}}},
		{Person: "high", Matrix: [][]int{{10}}},
		{Person: "middle", Matrix: [][]int{{5}}},
	}

	filtered := filterPeopleBurndownByActivity(people, 2)
	if len(filtered) != 2 {
		t.Fatalf("Expected 2 people after filtering, got %d", len(filtered))
	}
	if filtered[0].Person != "high" || filtered[1].Person != "middle" {
		t.Fatalf("Unexpected filter order: %#v", filtered)
	}
}
