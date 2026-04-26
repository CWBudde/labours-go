package modes

import (
	"os"
	"path/filepath"
	"testing"

	"labours-go/internal/readers"
)

type refactoringProxyTestReader struct {
	*NoDataReader
}

func (r *refactoringProxyTestReader) GetRefactoringProxy() (*readers.RefactoringProxyData, error) {
	return &readers.RefactoringProxyData{
		Threshold: 0.3,
		Ticks: []readers.RefactoringProxyTick{
			{RefactoringRate: 0.1, TotalChanges: 10},
			{RefactoringRate: 0.5, IsRefactoring: true, TotalChanges: 20},
		},
	}, nil
}

func TestRefactoringProxyWritesChart(t *testing.T) {
	output := filepath.Join(t.TempDir(), "refactoring-proxy.png")
	err := RefactoringProxy(&refactoringProxyTestReader{NoDataReader: &NoDataReader{}}, output)
	if err != nil {
		t.Fatalf("RefactoringProxy() unexpected error: %v", err)
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("Expected refactoring proxy chart at %s: %v", output, err)
	}
}
