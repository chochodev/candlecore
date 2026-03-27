package engine

import (
	"math"
	"time"
)

// BacktestReport contains performance metrics of a backtest run
type BacktestReport struct {
	StrategyName   string        `json:"strategy_name"`
	Symbol         string        `json:"symbol"`
	Timeframe      string        `json:"timeframe"`
	InitialBalance float64       `json:"initial_balance"`
	FinalBalance   float64       `json:"final_balance"`
	TotalPnL       float64       `json:"total_pnl"`
	TotalPnLPct    float64       `json:"total_pnl_pct"`
	WinRate        float64       `json:"win_rate"`
	TotalTrades    int           `json:"total_trades"`
	WinningTrades  int           `json:"winning_trades"`
	LosingTrades   int           `json:"losing_trades"`
	MaxDrawdown    float64       `json:"max_drawdown"`
	MaxDrawdownPct float64       `json:"max_drawdown_pct"`
	SharpeRatio    float64       `json:"sharpe_ratio"`
	Duration       time.Duration `json:"duration"`
	StartDate      time.Time     `json:"start_date"`
	EndDate        time.Time     `json:"end_date"`
}

// CalculateMetrics computes performance metrics from a list of trades and balance history
func CalculateMetrics(initialBalance float64, trades []Trade, balanceHistory []float64) *BacktestReport {
	report := &BacktestReport{
		InitialBalance: initialBalance,
		TotalTrades:    len(trades),
	}

	if len(balanceHistory) > 0 {
		report.FinalBalance = balanceHistory[len(balanceHistory)-1]
		report.TotalPnL = report.FinalBalance - initialBalance
		report.TotalPnLPct = (report.TotalPnL / initialBalance) * 100
	}

	// Calculate Win Rate
	if report.TotalTrades > 0 {
		for _, t := range trades {
			if t.PnL > 0 {
				report.WinningTrades++
			} else {
				report.LosingTrades++
			}
		}
		report.WinRate = (float64(report.WinningTrades) / float64(report.TotalTrades)) * 100
	}

	// Calculate Max Drawdown
	peak := initialBalance
	maxDD := 0.0
	for _, balance := range balanceHistory {
		if balance > peak {
			peak = balance
		}
		dd := peak - balance
		if dd > maxDD {
			maxDD = dd
		}
	}
	report.MaxDrawdown = maxDD
	report.MaxDrawdownPct = (maxDD / peak) * 100

	// Calculate Sharpe Ratio (Daily-based simplified version)
	// We'll calculate it based on returns between steps
	if len(balanceHistory) > 1 {
		returns := make([]float64, len(balanceHistory)-1)
		for i := 1; i < len(balanceHistory); i++ {
			returns[i-1] = (balanceHistory[i] - balanceHistory[i-1]) / balanceHistory[i-1]
		}

		mean := 0.0
		for _, r := range returns {
			mean += r
		}
		mean /= float64(len(returns))

		variance := 0.0
		for _, r := range returns {
			variance += math.Pow(r-mean, 2)
		}
		variance /= float64(len(returns))
		stdDev := math.Sqrt(variance)

		if stdDev > 0 {
			// Annualize (assuming daily data, otherwise timeframe adjustment is needed)
			// For simplicity, we just provide the ratio of observed steps
			report.SharpeRatio = mean / stdDev * math.Sqrt(252) // Assuming 252 trading days
		}
	}

	return report
}
