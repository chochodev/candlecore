package api

import (
	"candlecore/internal/bot"
	"candlecore/internal/engine"
	"candlecore/internal/exchange"
	"candlecore/internal/strategies"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type fastBacktestRequest struct {
	Symbol         string  `json:"symbol"`
	Timeframe      string  `json:"timeframe"`
	Strategy       string  `json:"strategy"`
	StartTime      int64   `json:"start_time"`
	EndTime        int64   `json:"end_time"`
	InitialBalance float64 `json:"initial_balance"`
}

type sideBreakdown struct {
	LongTrades  int `json:"long_trades"`
	ShortTrades int `json:"short_trades"`
}

type backtestAnalytics struct {
	AvgWin        float64       `json:"avg_win"`
	AvgLoss       float64       `json:"avg_loss"`
	ProfitFactor  float64       `json:"profit_factor"`
	Expectancy    float64       `json:"expectancy"`
	BestTrade     float64       `json:"best_trade"`
	WorstTrade    float64       `json:"worst_trade"`
	AvgHoldMins   float64       `json:"avg_hold_mins"`
	CandlesTotal  int           `json:"candles_total"`
	CandlesWindow int           `json:"candles_window"`
	WarmupCandles int           `json:"warmup_candles"`
	SideBreakdown sideBreakdown `json:"side_breakdown"`
}

type fastBacktestResponse struct {
	Report    *engine.BacktestReport `json:"report"`
	Analytics backtestAnalytics      `json:"analytics"`
	Trades    []bot.Position         `json:"trades"`
}

func (s *Server) handleFastBacktest(c *gin.Context) {
	var req fastBacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Symbol == "" || req.Timeframe == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol and timeframe are required"})
		return
	}
	if req.Strategy == "" {
		req.Strategy = "pulse_scalper"
	}
	if req.InitialBalance <= 0 {
		req.InitialBalance = defaultSimBalance
	}

	timeframe := exchange.Timeframe(req.Timeframe)
	provider := exchange.NewLocalFileProvider(s.dataDir)

	allCandles, err := provider.GetCandles(req.Symbol, timeframe, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var windowCandles []exchange.Candle
	startIndex := -1
	startAt := time.Unix(req.StartTime, 0)
	endAt := time.Unix(req.EndTime, 0)
	for i, cd := range allCandles {
		if req.StartTime > 0 && cd.Timestamp.Before(startAt) {
			continue
		}
		if req.EndTime > 0 && cd.Timestamp.After(endAt) {
			break
		}
		if startIndex == -1 {
			startIndex = i
		}
		windowCandles = append(windowCandles, cd)
	}
	if len(windowCandles) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no candles found for selected range"})
		return
	}

	warmupCount := 200
	warmupStart := startIndex - warmupCount
	if warmupStart < 0 {
		warmupStart = 0
	}
	runCandles := append([]exchange.Candle{}, allCandles[warmupStart:startIndex]...)
	runCandles = append(runCandles, windowCandles...)

	strategy, err := strategies.GetStrategy(req.Strategy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	b := bot.NewBot(strategy, provider, bot.Config{
		Symbol:         req.Symbol,
		Timeframe:      timeframe,
		InitialBalance: req.InitialBalance,
		PositionSize:   10,
	})

	runStarted := time.Now()
	if err := b.RunBacktest(runCandles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	runDuration := time.Since(runStarted)

	rawTrades := b.GetTrades()
	filteredTrades := make([]bot.Position, 0, len(rawTrades))
	engineTrades := make([]engine.Trade, 0, len(rawTrades))
	for _, t := range rawTrades {
		if req.StartTime > 0 && t.OpenedAt.Before(startAt) {
			continue
		}
		if req.EndTime > 0 && t.OpenedAt.After(endAt) {
			continue
		}
		filteredTrades = append(filteredTrades, t)
		engineTrades = append(engineTrades, engine.Trade{PnL: t.RealizedPnL})
	}

	report := engine.CalculateMetrics(req.InitialBalance, engineTrades, b.GetBalanceHistory())
	report.StrategyName = req.Strategy
	report.Symbol = req.Symbol
	report.Timeframe = req.Timeframe
	report.Duration = runDuration
	report.StartDate = windowCandles[0].Timestamp
	report.EndDate = windowCandles[len(windowCandles)-1].Timestamp

	analytics := buildBacktestAnalytics(filteredTrades, len(allCandles), len(windowCandles), len(runCandles)-len(windowCandles))

	c.JSON(http.StatusOK, fastBacktestResponse{
		Report:    report,
		Analytics: analytics,
		Trades:    filteredTrades,
	})
}

func buildBacktestAnalytics(trades []bot.Position, candlesTotal, candlesWindow, warmupCandles int) backtestAnalytics {
	a := backtestAnalytics{
		CandlesTotal:  candlesTotal,
		CandlesWindow: candlesWindow,
		WarmupCandles: warmupCandles,
	}
	if len(trades) == 0 {
		return a
	}

	var winsSum, lossesSum float64
	var wins, losses int
	totalPnL := 0.0
	best := -math.MaxFloat64
	worst := math.MaxFloat64
	totalHoldMins := 0.0

	for _, t := range trades {
		pnl := t.RealizedPnL
		totalPnL += pnl
		if pnl > best {
			best = pnl
		}
		if pnl < worst {
			worst = pnl
		}

		if t.Side == "long" {
			a.SideBreakdown.LongTrades++
		}
		if t.Side == "short" {
			a.SideBreakdown.ShortTrades++
		}

		if pnl > 0 {
			wins++
			winsSum += pnl
		} else if pnl < 0 {
			losses++
			lossesSum += math.Abs(pnl)
		}

		if t.ClosedAt != nil {
			totalHoldMins += t.ClosedAt.Sub(t.OpenedAt).Minutes()
		}
	}

	if wins > 0 {
		a.AvgWin = winsSum / float64(wins)
	}
	if losses > 0 {
		a.AvgLoss = lossesSum / float64(losses)
	}
	if lossesSum > 0 {
		a.ProfitFactor = winsSum / lossesSum
	}
	a.Expectancy = totalPnL / float64(len(trades))
	a.BestTrade = best
	a.WorstTrade = worst
	a.AvgHoldMins = totalHoldMins / float64(len(trades))

	return a
}
