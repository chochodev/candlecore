package research

import (
	"fmt"
	"sort"
)

// ConditionStats holds raw results for a specific set of criteria
type ConditionStats struct {
	Condition  string
	Samples    int
	TPCount    int
	SLCount    int
	TimeExpiries int
	TotalPnL   float64
	WinRate    float64
	Expectancy float64 // PnL per trade
}

// AnalyzeResults scans the dataset for profitable clusters
func AnalyzeResults(data []ResearchRow) []ConditionStats {
	groups := make(map[string]*ConditionStats)

	for _, row := range data {
		// HARDENED KEY: The Full Context Signature
		key := fmt.Sprintf("%-7s|%-8s|%-10s|%-8s", 
			row.Features.H1Trend, 
			row.Features.VolSpike, 
			row.Features.BreakOfStructure,
			row.Features.PullbackDepth)
		
		if _, exists := groups[key]; !exists {
			groups[key] = &ConditionStats{Condition: key}
		}

		stat := groups[key]
		stat.Samples++
		
		if row.Labels.HitBarrier == "tp" {
			stat.TPCount++
			stat.TotalPnL += 2.0 
		} else if row.Labels.HitBarrier == "sl" {
			stat.SLCount++
			stat.TotalPnL -= 1.0 
		} else {
			stat.TimeExpiries++
		}
	}

	// Calculate Final Stats
	var results []ConditionStats
	for _, s := range groups {
		if s.Samples < 30 { continue } // Higher threshold for higher significance
		
		validSamples := s.Samples - s.TimeExpiries
		if validSamples > 0 {
			s.WinRate = (float64(s.TPCount) / float64(validSamples)) * 100
		}
		s.Expectancy = s.TotalPnL / float64(s.Samples)
		
		if s.Expectancy > 0.10 {
			results = append(results, *s)
		}
	}

	// Sort by Expectancy
	sort.Slice(results, func(i, j int) bool {
		return results[i].Expectancy > results[j].Expectancy
	})

	return results
}

// PrintReport outputs the ranking
func PrintReport(stats []ConditionStats) {
	fmt.Printf("\n--- STATISTICAL EDGE REPORT ---\n")
	fmt.Printf("%-35s | %-8s | %-8s | %-10s\n", "CONDITION CLUSTER", "SAMPLES", "WIN RATE", "EXPECTANCY")
	fmt.Printf("----------------------------------------------------------------------\n")
	for _, s := range stats {
		if s.Expectancy <= 0 { continue }
		fmt.Printf("%-35s | %-8d | %-8.1f%% | %-10.2f\n", s.Condition, s.Samples, s.WinRate, s.Expectancy)
	}
	fmt.Println("----------------------------------------------------------------------")
}
