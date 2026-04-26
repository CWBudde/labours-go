package modes

import (
	"fmt"
	"math"
	"path/filepath"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"labours-go/internal/graphics"
	"labours-go/internal/readers"
)

// SentimentResult represents sentiment analysis for a developer or file
type SentimentResult struct {
	Entity   string
	Type     string // "developer" or "language"
	Positive float64
	Neutral  float64
	Negative float64
	Score    float64 // Overall sentiment score (-1 to 1)
}

// Sentiment generates sentiment analysis based on available repository data
// This analyzes patterns in developer activity and language usage to infer sentiment trends
func Sentiment(reader readers.Reader, output string) error {
	fmt.Println("Analyzing repository sentiment patterns...")

	// Collect sentiment results from different data sources
	var sentimentResults []SentimentResult
	if sentimentReader, ok := reader.(readers.SentimentReader); ok {
		ticks, err := sentimentReader.GetSentimentByTick()
		if err == nil && len(ticks) > 0 {
			sentimentResults = append(sentimentResults, sentimentResultsFromTicks(ticks)...)
		}
		if err != nil {
			fmt.Printf("Warning: Could not read collected sentiment data: %v\n", err)
		}
	}

	if len(sentimentResults) == 0 {
		// Fall back to the older heuristic path for fixtures without CommentSentimentResults.
		devResults, err := analyzeDeveloperSentiment(reader)
		if err != nil {
			fmt.Printf("Warning: Could not analyze developer sentiment: %v\n", err)
		} else {
			sentimentResults = append(sentimentResults, devResults...)
		}

		langResults, err := analyzeLanguageSentiment(reader)
		if err != nil {
			fmt.Printf("Warning: Could not analyze language sentiment: %v\n", err)
		} else {
			sentimentResults = append(sentimentResults, langResults...)
		}
	}

	if len(sentimentResults) == 0 {
		return fmt.Errorf("no sentiment data available - ensure the input contains developer stats or language stats")
	}

	// Generate visualizations
	if err := plotSentimentOverview(sentimentResults, output); err != nil {
		return fmt.Errorf("failed to generate sentiment overview: %v", err)
	}

	if err := plotSentimentByType(sentimentResults, output); err != nil {
		return fmt.Errorf("failed to generate sentiment by type: %v", err)
	}

	// Print summary
	printSentimentSummary(sentimentResults)

	fmt.Printf("Sentiment analysis completed. Analyzed %d entities.\n", len(sentimentResults))
	return nil
}

func sentimentResultsFromTicks(ticks map[int]readers.SentimentTick) []SentimentResult {
	keys := make([]int, 0, len(ticks))
	for tick := range ticks {
		keys = append(keys, tick)
	}
	sort.Ints(keys)

	results := make([]SentimentResult, 0, len(keys))
	for _, tick := range keys {
		value := float64(ticks[tick].Value)
		if !isFinite(value) {
			continue
		}
		score := clamp(value, -1, 1)
		positive := 0.0
		negative := 0.0
		if score > 0 {
			positive = score
		} else if score < 0 {
			negative = -score
		}
		results = append(results, normalizeSentimentResult(SentimentResult{
			Entity:   fmt.Sprintf("tick %d", tick),
			Type:     "tick",
			Positive: positive,
			Neutral:  1 - math.Abs(score),
			Negative: negative,
			Score:    score,
		}))
	}
	return results
}

// analyzeDeveloperSentiment analyzes sentiment based on developer activity patterns
func analyzeDeveloperSentiment(reader readers.Reader) ([]SentimentResult, error) {
	devStats, err := reader.GetDeveloperStats()
	if err != nil || len(devStats) == 0 {
		return nil, fmt.Errorf("no developer statistics available")
	}

	var results []SentimentResult

	// Calculate total metrics for normalization
	var totalCommits, totalAdded, totalRemoved int
	for _, dev := range devStats {
		totalCommits += dev.Commits
		totalAdded += dev.LinesAdded
		totalRemoved += dev.LinesRemoved
	}
	_ = totalAdded
	_ = totalRemoved
	if totalCommits <= 0 {
		totalCommits = 1
	}

	for _, dev := range devStats {
		// Calculate activity ratios
		commitRatio := float64(dev.Commits) / float64(totalCommits)
		changeRatio := float64(dev.LinesAdded) / float64(dev.LinesAdded+dev.LinesRemoved+1)

		// Sentiment heuristics based on activity patterns:
		// - High commit activity = positive engagement
		// - Balanced add/remove ratio = positive code quality focus
		// - Many languages = positive exploration/learning
		// - Many files touched = positive collaboration

		positive := commitRatio*0.3 + changeRatio*0.3 +
			float64(len(dev.Languages))/10.0*0.2 +
			float64(dev.FilesTouched)/100.0*0.2

		// Factors that might indicate negative sentiment:
		// - Very high removal ratio (frustration/cleanup)
		// - Very low commit activity (disengagement)
		negative := 0.0
		if dev.LinesRemoved > dev.LinesAdded*2 {
			negative += 0.3 // High removal ratio
		}
		if commitRatio < 0.01 && len(devStats) > 5 {
			negative += 0.2 // Very low activity
		}

		neutral := 1.0 - positive - negative
		if neutral < 0 {
			// Normalize if we went over 1.0
			total := positive + negative
			positive = positive / total
			negative = negative / total
			neutral = 0.0
		}

		score := positive - negative // Overall sentiment score

		results = append(results, normalizeSentimentResult(SentimentResult{
			Entity:   dev.Name,
			Type:     "developer",
			Positive: positive,
			Neutral:  neutral,
			Negative: negative,
			Score:    score,
		}))
	}

	return results, nil
}

// analyzeLanguageSentiment analyzes sentiment based on language usage patterns
func analyzeLanguageSentiment(reader readers.Reader) ([]SentimentResult, error) {
	langStats, err := reader.GetLanguageStats()
	if err != nil || len(langStats) == 0 {
		return nil, fmt.Errorf("no language statistics available")
	}

	var results []SentimentResult

	// Calculate total lines for normalization
	totalLines := 0
	for _, lang := range langStats {
		totalLines += lang.Lines
	}
	if totalLines <= 0 {
		return nil, fmt.Errorf("no non-zero language statistics available")
	}

	// Language sentiment heuristics based on common perceptions and usage patterns
	languageSentimentMap := map[string]float64{
		"Go":         0.7,  // Modern, loved language
		"Python":     0.6,  // Popular, versatile
		"JavaScript": 0.2,  // Mixed feelings, necessary evil
		"TypeScript": 0.5,  // Improvement over JS
		"Rust":       0.8,  // Loved by developers
		"Java":       0.1,  // Corporate, verbose
		"C++":        0.0,  // Complex, powerful but hard
		"C":          0.3,  // Respected but challenging
		"Kotlin":     0.6,  // Modern improvement over Java
		"Swift":      0.5,  // Apple ecosystem
		"PHP":        -0.2, // Often criticized
		"Ruby":       0.4,  // Elegant but less popular
		"Shell":      0.1,  // Necessary but limited
		"HTML":       0.2,  // Markup, not programming
		"CSS":        0.1,  // Styling challenges
		"SQL":        0.3,  // Necessary data tool
		"Dockerfile": 0.4,  // Modern deployment
		"YAML":       0.2,  // Configuration
		"JSON":       0.3,  // Data format
		"Markdown":   0.5,  // Documentation friendly
	}

	for _, lang := range langStats {
		usage := float64(lang.Lines) / float64(totalLines)

		// Get base sentiment for this language (default neutral)
		baseSentiment := languageSentimentMap[lang.Language]
		if _, exists := languageSentimentMap[lang.Language]; !exists {
			baseSentiment = 0.0 // Neutral for unknown languages
		}

		// Weight sentiment by usage - more used languages have more impact
		weightedSentiment := baseSentiment * usage * 2.0 // Amplify for visibility

		var positive, negative, neutral float64
		if weightedSentiment > 0 {
			positive = weightedSentiment
			negative = 0.0
		} else if weightedSentiment < 0 {
			positive = 0.0
			negative = -weightedSentiment
		}

		neutral = 1.0 - positive - negative
		if neutral < 0 {
			neutral = 0.0
		}

		results = append(results, normalizeSentimentResult(SentimentResult{
			Entity:   lang.Language,
			Type:     "language",
			Positive: positive,
			Neutral:  neutral,
			Negative: negative,
			Score:    weightedSentiment,
		}))
	}

	return results, nil
}

func normalizeSentimentResult(result SentimentResult) SentimentResult {
	result.Positive = nonNegativeFinite(result.Positive)
	result.Neutral = nonNegativeFinite(result.Neutral)
	result.Negative = nonNegativeFinite(result.Negative)
	if !isFinite(result.Score) {
		result.Score = 0
	}

	total := result.Positive + result.Neutral + result.Negative
	switch {
	case total == 0:
		result.Neutral = 1
	case total > 1:
		result.Positive /= total
		result.Neutral /= total
		result.Negative /= total
	}
	return result
}

func nonNegativeFinite(value float64) float64 {
	if !isFinite(value) || value < 0 {
		return 0
	}
	return value
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

// plotSentimentOverview creates a stacked bar chart showing overall sentiment distribution
func plotSentimentOverview(results []SentimentResult, output string) error {
	// Sort by sentiment score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	p := plot.New()
	p.Title.Text = "Repository Sentiment Analysis Overview"
	p.X.Label.Text = "Entities (Developers & Languages)"
	p.Y.Label.Text = "Sentiment Distribution"

	// Prepare data for stacked bars
	entities := make([]string, len(results))
	positiveVals := make(plotter.Values, len(results))
	neutralVals := make(plotter.Values, len(results))
	negativeVals := make(plotter.Values, len(results))

	for i, result := range results {
		result = normalizeSentimentResult(result)
		entities[i] = fmt.Sprintf("%s (%s)", result.Entity, result.Type)
		positiveVals[i] = result.Positive
		neutralVals[i] = result.Neutral
		negativeVals[i] = result.Negative
	}

	// Create stacked bars
	positiveBars, err := plotter.NewBarChart(positiveVals, vg.Points(40))
	if err != nil {
		return fmt.Errorf("failed to create positive bars: %v", err)
	}
	positiveBars.Color = graphics.ColorPalette[2] // Green

	neutralBars, err := plotter.NewBarChart(neutralVals, vg.Points(40))
	if err != nil {
		return fmt.Errorf("failed to create neutral bars: %v", err)
	}
	neutralBars.Color = graphics.ColorPalette[7] // Gray
	neutralBars.StackOn(positiveBars)

	negativeBars, err := plotter.NewBarChart(negativeVals, vg.Points(40))
	if err != nil {
		return fmt.Errorf("failed to create negative bars: %v", err)
	}
	negativeBars.Color = graphics.ColorPalette[3] // Red
	negativeBars.StackOn(neutralBars)

	p.Add(positiveBars, neutralBars, negativeBars)
	p.NominalX(entities...)

	// Rotate labels if many entities
	if len(results) > 8 {
		p.X.Tick.Label.Rotation = 0.785398 // 45 degrees
		p.X.Tick.Label.XAlign = -0.5
		p.X.Tick.Label.YAlign = -0.5
	}

	// Add legend
	p.Legend.Add("Positive", positiveBars)
	p.Legend.Add("Neutral", neutralBars)
	p.Legend.Add("Negative", negativeBars)
	p.Legend.Top = true

	// Save plots
	outputFile := filepath.Join(output, "sentiment-overview.png")
	if err := p.Save(16*vg.Inch, 8*vg.Inch, outputFile); err != nil {
		return fmt.Errorf("failed to save sentiment overview: %v", err)
	}

	svgFile := filepath.Join(output, "sentiment-overview.svg")
	if err := p.Save(16*vg.Inch, 8*vg.Inch, svgFile); err != nil {
		return fmt.Errorf("failed to save sentiment overview SVG: %v", err)
	}

	fmt.Printf("Sentiment overview charts saved to %s and %s\n", outputFile, svgFile)
	return nil
}

// plotSentimentByType creates separate charts for developers and languages
func plotSentimentByType(results []SentimentResult, output string) error {
	// Separate by type
	var developers, languages []SentimentResult
	for _, result := range results {
		if result.Type == "developer" {
			developers = append(developers, result)
		} else if result.Type == "language" {
			languages = append(languages, result)
		}
	}

	// Plot developer sentiment if available
	if len(developers) > 0 {
		if err := plotSentimentForType(developers, "Developer Sentiment Analysis", output, "sentiment-developers"); err != nil {
			return fmt.Errorf("failed to plot developer sentiment: %v", err)
		}
	}

	// Plot language sentiment if available
	if len(languages) > 0 {
		if err := plotSentimentForType(languages, "Language Sentiment Analysis", output, "sentiment-languages"); err != nil {
			return fmt.Errorf("failed to plot language sentiment: %v", err)
		}
	}

	return nil
}

// plotSentimentForType creates a sentiment chart for a specific type
func plotSentimentForType(results []SentimentResult, title, output, filename string) error {
	// Sort by sentiment score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	p := plot.New()
	p.Title.Text = title
	p.X.Label.Text = "Entities"
	p.Y.Label.Text = "Sentiment Score"

	// Create scatter plot of sentiment scores
	pts := make(plotter.XYs, len(results))
	for i, result := range results {
		result = normalizeSentimentResult(result)
		pts[i].X = float64(i)
		pts[i].Y = result.Score
	}

	scatter, err := plotter.NewScatter(pts)
	if err != nil {
		return fmt.Errorf("failed to create scatter plot: %v", err)
	}
	scatter.GlyphStyle.Color = graphics.ColorPalette[0]
	scatter.GlyphStyle.Radius = vg.Points(4)

	p.Add(scatter)

	// Set X-axis labels
	names := make([]string, len(results))
	for i, result := range results {
		names[i] = result.Entity
	}
	p.NominalX(names...)

	// Rotate labels if many entities
	if len(results) > 6 {
		p.X.Tick.Label.Rotation = 0.785398
		p.X.Tick.Label.XAlign = -0.5
		p.X.Tick.Label.YAlign = -0.5
	}

	// Add horizontal line at y=0 for neutral sentiment
	line := plotter.NewFunction(func(x float64) float64 { return 0 })
	line.Color = graphics.ColorPalette[4]
	line.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
	p.Add(line)

	// Save plots
	outputFile := filepath.Join(output, filename+".png")
	if err := p.Save(14*vg.Inch, 8*vg.Inch, outputFile); err != nil {
		return fmt.Errorf("failed to save %s: %v", filename, err)
	}

	svgFile := filepath.Join(output, filename+".svg")
	if err := p.Save(14*vg.Inch, 8*vg.Inch, svgFile); err != nil {
		return fmt.Errorf("failed to save %s SVG: %v", filename, err)
	}

	fmt.Printf("%s charts saved to %s and %s\n", title, outputFile, svgFile)
	return nil
}

// printSentimentSummary prints a text summary of sentiment analysis
func printSentimentSummary(results []SentimentResult) {
	fmt.Println("\nSentiment Analysis Summary:")
	fmt.Println("===========================")

	// Calculate overall statistics
	var totalPositive, totalNeutral, totalNegative float64
	var devCount, langCount int

	for _, result := range results {
		result = normalizeSentimentResult(result)
		totalPositive += result.Positive
		totalNeutral += result.Neutral
		totalNegative += result.Negative

		if result.Type == "developer" {
			devCount++
		} else {
			langCount++
		}
	}

	avgPositive := totalPositive / float64(len(results))
	avgNeutral := totalNeutral / float64(len(results))
	avgNegative := totalNegative / float64(len(results))

	fmt.Printf("Overall Sentiment Distribution:\n")
	fmt.Printf("  Positive: %.1f%%\n", avgPositive*100)
	fmt.Printf("  Neutral:  %.1f%%\n", avgNeutral*100)
	fmt.Printf("  Negative: %.1f%%\n", avgNegative*100)
	fmt.Printf("\nAnalyzed: %d developers, %d languages\n", devCount, langCount)

	// Show top positive and negative entities
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	fmt.Println("\nMost Positive Entities:")
	for i, result := range results {
		if i >= 5 || result.Score <= 0 {
			break
		}
		fmt.Printf("  %d. %s (%s) - Score: %.3f\n", i+1, result.Entity, result.Type, result.Score)
	}

	fmt.Println("\nMost Negative Entities:")
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score < results[j].Score
	})
	for i, result := range results {
		if i >= 5 || result.Score >= 0 {
			break
		}
		fmt.Printf("  %d. %s (%s) - Score: %.3f\n", i+1, result.Entity, result.Type, result.Score)
	}
}
