package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHerculesReportWithLocalLaboursStrict(t *testing.T) {
	if os.Getenv("LABOURS_GO_HERCULES_REPORT") != "1" {
		t.Skip("Hercules report integration is opt-in; set LABOURS_GO_HERCULES_REPORT=1")
	}

	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("Failed to resolve repository root: %v", err)
	}

	herculesBin := os.Getenv("HERCULES_BIN")
	if herculesBin == "" {
		herculesBin = filepath.Join(repoRoot, "..", "hercules", "hercules")
	}
	if info, err := os.Stat(herculesBin); err != nil || info.IsDir() {
		t.Skipf("Hercules binary not available: %s", herculesBin)
	}

	fixtureRepo := os.Getenv("HERCULES_FIXTURE_REPO")
	if fixtureRepo == "" {
		fixtureRepo = filepath.Join(repoRoot, "..", "hercules", "cmd", "hercules", "test_data", "hercules.siva")
	}
	if _, err := os.Stat(fixtureRepo); err != nil {
		t.Skipf("Hercules fixture repository not available: %s", fixtureRepo)
	}

	tmpDir := t.TempDir()
	laboursBin := filepath.Join(tmpDir, "labours")
	build := exec.Command("go", "build", "-o", laboursBin, repoRoot)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build local labours binary: %v\n%s", err, output)
	}

	reportDir := filepath.Join(tmpDir, "report")
	report := exec.Command(
		herculesBin,
		"report",
		"--labours-cmd", laboursBin,
		"--strict",
		"-o", reportDir,
		fixtureRepo,
	)
	if output, err := report.CombinedOutput(); err != nil {
		t.Fatalf("Hercules report failed: %v\n%s", err, output)
	}

	indexPath := filepath.Join(reportDir, "index.html")
	index, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("Report index was not generated at %s: %v", indexPath, err)
	}
	if strings.Contains(strings.ToLower(string(index)), "failed") {
		t.Fatalf("Report index contains a failure marker")
	}

	charts, err := filepath.Glob(filepath.Join(reportDir, "charts", "*"))
	if err != nil {
		t.Fatalf("Failed to list report charts: %v", err)
	}
	if len(charts) < 10 {
		t.Fatalf("Expected at least 10 report chart assets, got %d", len(charts))
	}
}
