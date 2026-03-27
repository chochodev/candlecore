package risk

// StoplossType defines the type of stoploss
type StoplossType string

const (
	StoplossFixed    StoplossType = "fixed"
	StoplossTrailing StoplossType = "trailing"
	StoplossCustom   StoplossType = "custom"
)

// StoplossManager handles stoploss calculations and updates
type StoplossManager struct {
	Type           StoplossType
	StoplossPercent float64 // e.g., -0.05 for -5%
	TrailingDelta   float64 // Trailing stop distance
}

// NewStoplossManager creates a new stoploss manager
func NewStoplossManager(stoplossPercent float64, trailing bool, trailingDelta float64) *StoplossManager {
	sType := StoplossFixed
	if trailing {
		sType = StoplossTrailing
	}
	
	return &StoplossManager{
		Type:           sType,
		StoplossPercent: stoplossPercent,
		TrailingDelta:   trailingDelta,
	}
}

// CalculateStoploss calculates the initial stoploss price
func (sm *StoplossManager) CalculateStoploss(entryPrice float64) float64 {
	return entryPrice * (1 + sm.StoplossPercent)
}

// UpdateTrailingStop updates the stoploss for a trailing stop
func (sm *StoplossManager) UpdateTrailingStop(currentPrice float64, currentStoploss float64, side string) float64 {
	if sm.Type != StoplossTrailing {
		return currentStoploss
	}
	
	if side == "long" {
		// For long positions, stoploss follows price up
		newStoploss := currentPrice * (1 - sm.TrailingDelta)
		if newStoploss > currentStoploss {
			return newStoploss
		}
	} else {
		// For short positions, stoploss follows price down
		newStoploss := currentPrice * (1 + sm.TrailingDelta)
		if newStoploss < currentStoploss {
			return newStoploss
		}
	}
	
	return currentStoploss
}

// ShouldStoploss checks if stoploss should be triggered
func (sm *StoplossManager) ShouldStoploss(currentPrice float64, stoplossPrice float64, side string) bool {
	if side == "long" {
		return currentPrice <= stoplossPrice
	}
	// For short positions
	return currentPrice >= stoplossPrice
}

// CalculateProfit calculates current profit/loss percentage
func CalculateProfit(entryPrice, currentPrice float64, side string) float64 {
	if side == "long" {
		return (currentPrice - entryPrice) / entryPrice
	}
	// For short positions
	return (entryPrice - currentPrice) / entryPrice
}
