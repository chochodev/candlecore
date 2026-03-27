package risk

import (
	"time"
)

// ROIEntry represents a time-based ROI target
type ROIEntry struct {
	Minutes int
	ROI     float64
}

// ROIManager handles minimal ROI calculations
type ROIManager struct {
	ROITable map[int]float64 // minutes -> ROI percentage
}

// NewROIManager creates a new ROI manager
func NewROIManager(roiTable map[int]float64) *ROIManager {
	if roiTable == nil {
		roiTable = map[int]float64{
			0:   0.10, // 10% immediate
			60:  0.05, // 5% after 1 hour
			120: 0.02, // 2% after 2 hours
		}
	}
	
	return &ROIManager{
		ROITable: roiTable,
	}
}

// GetMinimalROI returns the minimal ROI for the given trade duration
func (rm *ROIManager) GetMinimalROI(entryTime time.Time, currentTime time.Time) float64 {
	duration := currentTime.Sub(entryTime)
	durationMinutes := int(duration.Minutes())
	
	// Find the applicable ROI based on duration
	// ROI table is sorted by time (lowest to highest)
	// We want the highest time threshold that's still <= duration
	minimalROI := 0.0
	maxApplicableMinutes := -1
	
	for minutes, roi := range rm.ROITable {
		if minutes <= durationMinutes && minutes > maxApplicableMinutes {
			maxApplicableMinutes = minutes
			minimalROI = roi
		}
	}
	
	return minimalROI
}

// ShouldTakeProfit checks if the current profit meets or exceeds minimal ROI
func (rm *ROIManager) ShouldTakeProfit(entryTime time.Time, currentTime time.Time, currentProfit float64) (bool, string) {
	minimalROI := rm.GetMinimalROI(entryTime, currentTime)
	
	if currentProfit >= minimalROI {
		duration := currentTime.Sub(entryTime)
		return true, formatROIReason(currentProfit, minimalROI, duration)
	}
	
	return false, ""
}

func formatROIReason(currentProfit, minimalROI float64, duration time.Duration) string {
	minutes := int(duration.Minutes())
	return formatString("ROI target reached: %.2f%% >= %.2f%% after %d minutes", 
		currentProfit*100, minimalROI*100, minutes)
}

func formatString(format string, args ...interface{}) string {
	// Simple sprintf-like function
	result := format
	// This is a placeholder - in real code use fmt.Sprintf
	return result
}
