package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestLaboursReferenceRecipesIncludeBurndownProject(t *testing.T) {
	recipe, ok := findLaboursReferenceRecipe("burndown_project")
	if !ok {
		t.Fatal("missing burndown_project recipe")
	}
	if recipe.Input != filepath.Join("test", "testdata", "hercules", "report_default.pb") {
		t.Fatalf("input = %q, want report fixture", recipe.Input)
	}
	if recipe.Mode != "burndown-project" {
		t.Fatalf("mode = %q, want burndown-project", recipe.Mode)
	}
	if recipe.OutputIsDir {
		t.Fatal("burndown_project should render to the requested output file")
	}
}

func TestBuildEntryComparesDisplayedComposite(t *testing.T) {
	dir := t.TempDir()
	transparentWhite := filepath.Join(dir, "transparent-white.png")
	opaqueWhite := filepath.Join(dir, "opaque-white.png")

	writeTestPNG(t, transparentWhite, color.RGBA{R: 255, G: 255, B: 255, A: 0})
	writeTestPNG(t, opaqueWhite, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	entry, err := buildEntry("plots", "reference", "background", transparentWhite, opaqueWhite)
	if err != nil {
		t.Fatalf("build entry: %v", err)
	}
	if entry.RMSE != 0 || entry.AvgDiff != 0 || entry.MaxDiff != 0 || entry.DiffPixels != 0 {
		t.Fatalf("transparent white and opaque white should compare equal after display compositing: %+v", entry)
	}
}

func writeTestPNG(t *testing.T, path string, c color.RGBA) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test png: %v", err)
	}
	defer file.Close()
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode test png: %v", err)
	}
}
