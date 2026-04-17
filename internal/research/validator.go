package research

import (
	"fmt"
)

// ValidateEdge performs a 70/30 Walk-Forward analysis
func ValidateEdge(data []ResearchRow) {
	total := len(data)
	splitIdx := int(float64(total) * 0.7)
	
	trainData := data[:splitIdx]
	testData := data[splitIdx:]

	fmt.Printf("\n--- WALK-FORWARD VALIDATION REPORT ---\n")
	fmt.Printf("Train Set Size: %d | Test Set Size: %d\n", len(trainData), len(testData))
	
	// 1. Get stats from Train
	trainStats := AnalyzeResults(trainData)
	fmt.Println("\nTop Clusters from Training Set:")
	PrintReport(trainStats[:3]) // Top 3

	// 2. Stress Test the #1 Edge on unseen Test data
	if len(trainStats) > 0 {
		topEdgeKey := trainStats[0].Condition
		
		fmt.Printf("\nStress Testing TOP EDGE on UNSEEN DATA: %s\n", topEdgeKey)
		testResults := FilterByCondition(testData, topEdgeKey)
		
		fmt.Printf("Test Samples: %d\n", testResults.Samples)
		fmt.Printf("Train WinRate: %.1f%%  -->  Test WinRate: %.1f%%\n", trainStats[0].WinRate, testResults.WinRate)
		fmt.Printf("Train Exp: %.2f  -->  Test Exp: %.2f\n", trainStats[0].Expectancy, testResults.Expectancy)
		
		drift := ((testResults.Expectancy - trainStats[0].Expectancy) / trainStats[0].Expectancy) * 100
		fmt.Printf("Expectancy Drift: %.1f%%\n", drift)
		
		if drift < -30 {
			fmt.Println("\n[REJECTED] - Performance collapsed out-of-sample. Overfitting detected.")
		} else {
			fmt.Println("\n[PASSED] - Edge holds on unseen data. Ready for codification.")
		}
	}
}

// FilterByCondition isolates a specific cluster in a dataset
func FilterByCondition(data []ResearchRow, key string) ConditionStats {
	stats := &ConditionStats{Condition: key}
	for _, row := range data {
		// Re-generate key - MUST MATCH analyzer.go exactly
		rowKey := fmt.Sprintf("%-7s|%-8s|%-10s|%-8s", 
			row.Features.H1Trend, 
			row.Features.VolSpike, 
			row.Features.BreakOfStructure,
			row.Features.PullbackDepth)
		
		if rowKey == key {
			stats.Samples++
			if row.Labels.HitBarrier == "tp" {
				stats.TPCount++
				stats.TotalPnL += 2.0
			} else if row.Labels.HitBarrier == "sl" {
				stats.SLCount++
				stats.TotalPnL -= 1.0
			} else {
				stats.TimeExpiries++
			}
		}
	}
	
	validSamples := stats.Samples - stats.TimeExpiries
	if validSamples > 0 {
		stats.WinRate = (float64(stats.TPCount) / float64(validSamples)) * 100
	}
	if stats.Samples > 0 {
		stats.Expectancy = stats.TotalPnL / float64(stats.Samples)
	}
	
	return *stats
}
