package scraper

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"candlecore/internal/engine"
)

// BinanceBulkScraper downloads historical data from Binance public data
type BinanceBulkScraper struct {
	dataDir    string
	client     *http.Client
	baseURL    string
}

// NewBinanceBulkScraper creates scraper for Binance bulk data
func NewBinanceBulkScraper(dataDir string) *BinanceBulkScraper {
	return &BinanceBulkScraper{
		dataDir: dataDir,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		baseURL: "https://data.binance.vision/data/spot/daily/klines",
	}
}

// SupportedPairs returns tradeable pairs
func (s *BinanceBulkScraper) SupportedPairs() map[string]string {
	return map[string]string{
		"BTCUSDT": "bitcoin",
		"ETHUSDT": "ethereum",
	}
}

// ScrapeFullHistory downloads all available historical data
func (s *BinanceBulkScraper) ScrapeFullHistory(ctx context.Context, symbol, interval string, startYear, endYear int) error {
	coinID, ok := s.SupportedPairs()[symbol]
	if !ok {
		return fmt.Errorf("unsupported symbol: %s", symbol)
	}
	
	if !s.ValidInterval(interval) {
		return fmt.Errorf("unsupported interval: %s (use: 5m, 15m, 1h, 4h, 1d)", interval)
	}
	
	fmt.Printf("Downloading %s %s data from %d to %d...\n", symbol, interval, startYear, endYear)
	fmt.Println("This may take several minutes depending on data size.")
	fmt.Println()
	
	allCandles := make([]engine.Candle, 0)
	totalMonths := 0
	successMonths := 0
	
	for year := startYear; year <= endYear; year++ {
		startMonth := 1
		endMonth := 12
		
		if year == endYear && time.Now().Year() == endYear {
			endMonth = int(time.Now().Month())
		}
		
		for month := startMonth; month <= endMonth; month++ {
			totalMonths++
			
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			
			candles, err := s.downloadMonth(ctx, symbol, interval, year, month)
			if err != nil {
				fmt.Printf("  ⚠ Skipped %s %d-%02d: %v\n", symbol, year, month, err)
				continue
			}
			
			if len(candles) > 0 {
				allCandles = append(allCandles, candles...)
				successMonths++
				fmt.Printf("  ✓ Downloaded %s %d-%02d (%d candles)\n", symbol, year, month, len(candles))
			}
		}
	}
	
	if len(allCandles) == 0 {
		return fmt.Errorf("no data downloaded for %s", symbol)
	}
	
	fmt.Printf("\nTotal: %d/%d months downloaded (%d candles)\n", successMonths, totalMonths, len(allCandles))
	
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s_%s.csv", coinID, interval))
	if err := s.writeCandles(filename, allCandles); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	fmt.Printf("Saved to: %s\n", filename)
	fmt.Printf("Date range: %s to %s\n\n",
		allCandles[0].Timestamp.Format("2006-01-02"),
		allCandles[len(allCandles)-1].Timestamp.Format("2006-01-02"),
	)
	
	return nil
}

// ValidInterval checks if interval is supported
func (s *BinanceBulkScraper) ValidInterval(interval string) bool {
	valid := map[string]bool{
		"5m":  true,
		"15m": true,
		"1h":  true,
		"4h":  true,
		"1d":  true,
	}
	return valid[interval]
}

// downloadMonth fetches one month of data
func (s *BinanceBulkScraper) downloadMonth(ctx context.Context, symbol, interval string, year, month int) ([]engine.Candle, error) {
	url := fmt.Sprintf("%s/%s/%s/%s-%s-%d-%02d.zip", s.baseURL, symbol, interval, symbol, interval, year, month)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("data not available")
	}
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	
	tmpFile, err := os.CreateTemp("", "binance-*.zip")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return nil, err
	}
	
	return s.extractCandles(tmpFile.Name())
}

// extractCandles extracts and parses candles from zip
func (s *BinanceBulkScraper) extractCandles(zipPath string) ([]engine.Candle, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	
	if len(r.File) == 0 {
		return nil, fmt.Errorf("empty zip file")
	}
	
	file := r.File[0]
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	
	reader := csv.NewReader(rc)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	
	candles := make([]engine.Candle, 0, len(records))
	
	for _, record := range records {
		if len(record) < 11 {
			continue
		}
		
		openTime, _ := strconv.ParseInt(record[0], 10, 64)
		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		close, _ := strconv.ParseFloat(record[4], 64)
		volume, _ := strconv.ParseFloat(record[5], 64)
		
		candles = append(candles, engine.Candle{
			Timestamp: time.UnixMilli(openTime),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}
	
	return candles, nil
}

// writeCandles saves candles to CSV
func (s *BinanceBulkScraper) writeCandles(filename string, candles []engine.Candle) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return err
	}
	
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	writer.Write([]string{"timestamp", "open", "high", "low", "close", "volume"})
	
	for _, c := range candles {
		record := []string{
			c.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.8f", c.Open),
			fmt.Sprintf("%.8f", c.High),
			fmt.Sprintf("%.8f", c.Low),
			fmt.Sprintf("%.8f", c.Close),
			fmt.Sprintf("%.8f", c.Volume),
		}
		writer.Write(record)
	}
	
	return nil
}
