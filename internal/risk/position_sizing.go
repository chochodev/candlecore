package risk

// PositionSizer calculates trade size based on risk parameters
type PositionSizer struct {
	MaxPositionPercent float64 // Max % of portfolio per position (e.g., 0.10 = 10%)
	MaxRiskPercent     float64 // Max % to risk per trade (e.g., 0.02 = 2%)
}

// NewPositionSizer creates a new position sizer
func NewPositionSizer(maxPositionPercent, maxRiskPercent float64) *PositionSizer {
	return &PositionSizer{
		MaxPositionPercent: maxPositionPercent,
		MaxRiskPercent:     maxRiskPercent,
	}
}

// CalculateStakeAmount calculates how much to invest in a trade
func (ps *PositionSizer) CalculateStakeAmount(portfolioValue float64, stoplossPercent float64, fixedStakeAmount float64) float64 {
	// If fixed stake amount is set, use it (but respect max position %)
	if fixedStakeAmount > 0 {
		maxAllowed := portfolioValue * ps.MaxPositionPercent
		if fixedStakeAmount > maxAllowed {
			return maxAllowed
		}
		return fixedStakeAmount
	}
	
	// Risk-based position sizing
	// Formula: stake = (portfolio * max_risk%) / |stoploss%|
	if stoplossPercent < 0 {
		stoplossPercent = -stoplossPercent
	}
	
	if stoplossPercent == 0 {
		stoplossPercent = 0.05 // Default 5% if not set
	}
	
	riskAmount := portfolioValue * ps.MaxRiskPercent
	stakeAmount := riskAmount / stoplossPercent
	
	// Ensure we don't exceed max position size
	maxAllowed := portfolioValue * ps.MaxPositionPercent
	if stakeAmount > maxAllowed {
		stakeAmount = maxAllowed
	}
	
	return stakeAmount
}

// CalculatePositionSize calculates how many units to buy
func (ps *PositionSizer) CalculatePositionSize(stakeAmount, currentPrice float64) float64 {
	if currentPrice == 0 {
		return 0
	}
	return stakeAmount / currentPrice
}
