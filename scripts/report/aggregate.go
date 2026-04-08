// Optional changelog deduplication and scorecard logic for scripts/report.
// To disable: stop importing these from main.go (or delete this file) and assign report.Benchmarks without DedupeBenchmarks.
package main

import (
	"candlecore/internal/engine"
	"math"
	"sort"
	"strings"
	"time"
)

// canonicalAliases maps raw symbol strings (any case) to a single canonical asset id for reporting.
// Extend this map when new duplicate CSV prefixes appear.
var canonicalAliases = map[string]string{
	"bitcoin":  "btc",
	"btc":      "btc",
	"ethereum": "eth",
	"eth":      "eth",
	"solana":   "sol",
	"sol":      "sol",
}

// CanonicalSymbol returns a stable lowercase asset id for deduplication and scorecard rows.
func CanonicalSymbol(symbol string) string {
	s := strings.ToLower(strings.TrimSpace(symbol))
	if c, ok := canonicalAliases[s]; ok {
		return c
	}
	return s
}

func reportSpan(r *engine.BacktestReport) time.Duration {
	if r == nil {
		return 0
	}
	if r.EndDate.IsZero() || r.StartDate.IsZero() {
		return 0
	}
	d := r.EndDate.Sub(r.StartDate)
	if d < 0 {
		return 0
	}
	return d
}

// pickBetter chooses the more informative duplicate: longer history span, then more trades, then higher abs Sharpe.
func pickBetter(a, b *engine.BacktestReport) *engine.BacktestReport {
	sa, sb := reportSpan(a), reportSpan(b)
	if sa != sb {
		if sa > sb {
			return a
		}
		return b
	}
	if a.TotalTrades != b.TotalTrades {
		if a.TotalTrades > b.TotalTrades {
			return a
		}
		return b
	}
	if math.Abs(a.SharpeRatio) != math.Abs(b.SharpeRatio) {
		if math.Abs(a.SharpeRatio) > math.Abs(b.SharpeRatio) {
			return a
		}
		return b
	}
	return a
}

// DedupeBenchmarks keeps one row per (canonical symbol, timeframe). Symbol on the row is rewritten to canonical.
func DedupeBenchmarks(reports []*engine.BacktestReport) []*engine.BacktestReport {
	type key struct {
		sym string
		tf  string
	}
	best := make(map[key]*engine.BacktestReport)

	for _, r := range reports {
		if r == nil {
			continue
		}
		canonical := CanonicalSymbol(r.Symbol)
		k := key{sym: canonical, tf: r.Timeframe}
		cp := *r
		cp.Symbol = canonical

		existing, ok := best[k]
		if !ok {
			best[k] = &cp
			continue
		}
		chosen := pickBetter(existing, &cp)
		if chosen == &cp {
			best[k] = &cp
		}
	}

	out := make([]*engine.BacktestReport, 0, len(best))
	for _, v := range best {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Symbol != out[j].Symbol {
			return out[i].Symbol < out[j].Symbol
		}
		return out[i].Timeframe < out[j].Timeframe
	})
	return out
}

// ScorecardRow is one line in the one-page aggregate (unique asset + timeframe).
type ScorecardRow struct {
	Symbol         string  `json:"symbol"`
	Timeframe      string  `json:"timeframe"`
	TotalPnLPct    float64 `json:"total_pnl_pct"`
	WinRate        float64 `json:"win_rate"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	MaxDrawdownPct float64 `json:"max_drawdown_pct"`
	TotalTrades    int     `json:"total_trades"`
}

// ScorecardTotals summarizes all deduped pairs.
type ScorecardTotals struct {
	UniquePairs           int     `json:"unique_pairs"`
	PairsPositivePnLPct   int     `json:"pairs_with_positive_total_pnl_pct"`
	AvgTotalPnLPct        float64 `json:"avg_total_pnl_pct_across_pairs"`
	AvgSharpeRatio        float64 `json:"avg_sharpe_ratio_across_pairs"`
}

// ScorecardSummary is the "true aggregate" view after deduplication.
type ScorecardSummary struct {
	StrategyName string          `json:"strategy_name"`
	Rows         []ScorecardRow  `json:"rows"`
	Totals       ScorecardTotals `json:"totals"`
}

// BuildScorecard builds a compact table and rollups from deduped benchmarks.
func BuildScorecard(strategyName string, deduped []*engine.BacktestReport) ScorecardSummary {
	rows := make([]ScorecardRow, 0, len(deduped))
	var sumPnLPct, sumSharpe float64
	positive := 0

	for _, r := range deduped {
		if r == nil {
			continue
		}
		rows = append(rows, ScorecardRow{
			Symbol:         r.Symbol,
			Timeframe:      r.Timeframe,
			TotalPnLPct:    r.TotalPnLPct,
			WinRate:        r.WinRate,
			SharpeRatio:    r.SharpeRatio,
			MaxDrawdownPct: r.MaxDrawdownPct,
			TotalTrades:    r.TotalTrades,
		})
		sumPnLPct += r.TotalPnLPct
		sumSharpe += r.SharpeRatio
		if r.TotalPnLPct > 0 {
			positive++
		}
	}

	n := len(rows)
	totals := ScorecardTotals{
		UniquePairs:         n,
		PairsPositivePnLPct: positive,
	}
	if n > 0 {
		totals.AvgTotalPnLPct = sumPnLPct / float64(n)
		totals.AvgSharpeRatio = sumSharpe / float64(n)
	}

	return ScorecardSummary{
		StrategyName: strategyName,
		Rows:         rows,
		Totals:       totals,
	}
}
