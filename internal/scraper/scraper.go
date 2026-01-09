package scraper

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"candlecore/internal/engine"
	"candlecore/internal/fetcher"
)

// DataScraper handles historical data scraping and storage
type DataScraper struct {
	dataDir      string
	coinGecko    *fetcher.CoinGeckoFetcher
	rateLimit    time.Duration
	maxHistorical int
}

// NewDataScraper creates a new data scraper
func NewDataScraper(dataDir string) *DataScraper {
	return &DataScraper{
		dataDir:      dataDir,
		coinGecko:    fetcher.NewCoinGeckoFetcher(),
		rateLimit:    time.Second * 3,
		maxHistorical: 365,
	}
}

// SupportedCoins returns list of supported cryptocurrencies
func (s *DataScraper) SupportedCoins() []string {
	return []string{"bitcoin", "ethereum"}
}

// ScrapeCoin fetches maximum historical data for a coin
func (s *DataScraper) ScrapeCoin(ctx context.Context, coinID string) error {
	fmt.Printf("Scraping %s data (max history: %d days)...\n", coinID, s.maxHistorical)
	
	candles, err := s.coinGecko.FetchCandles(ctx, coinID, s.maxHistorical)
	if err != nil {
		return fmt.Errorf("failed to fetch %s data: %w", coinID, err)
	}
	
	if len(candles) == 0 {
		return fmt.Errorf("no data received for %s", coinID)
	}
	
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s_daily.csv", coinID))
	
	if err := s.writeCandles(filename, candles); err != nil {
		return fmt.Errorf("failed to write %s data: %w", coinID, err)
	}
	
	fmt.Printf("Successfully scraped %d candles for %s\n", len(candles), coinID)
	fmt.Printf("Date range: %s to %s\n", 
		candles[0].Timestamp.Format("2006-01-02"),
		candles[len(candles)-1].Timestamp.Format("2006-01-02"),
	)
	fmt.Printf("Saved to: %s\n\n", filename)
	
	return nil
}

// UpdateCoin appends new data since last scrape
func (s *DataScraper) UpdateCoin(ctx context.Context, coinID string) error {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s_daily.csv", coinID))
	
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return s.ScrapeCoin(ctx, coinID)
	}
	
	existingCandles, err := s.readCandles(filename)
	if err != nil {
		return fmt.Errorf("failed to read existing data: %w", err)
	}
	
	if len(existingCandles) == 0 {
		return s.ScrapeCoin(ctx, coinID)
	}
	
	lastTimestamp := existingCandles[len(existingCandles)-1].Timestamp
	daysSince := int(time.Since(lastTimestamp).Hours() / 24)
	
	if daysSince <= 1 {
		fmt.Printf("%s data is up to date (last: %s)\n", coinID, lastTimestamp.Format("2006-01-02"))
		return nil
	}
	
	fmt.Printf("Updating %s data (fetching last %d days)...\n", coinID, daysSince+1)
	
	newCandles, err := s.coinGecko.FetchCandlesSince(ctx, coinID, lastTimestamp.Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("failed to fetch new data: %w", err)
	}
	
	if len(newCandles) == 0 {
		fmt.Printf("No new data available for %s\n", coinID)
		return nil
	}
	
	allCandles := s.mergeCandles(existingCandles, newCandles)
	
	if err := s.writeCandles(filename, allCandles); err != nil {
		return fmt.Errorf("failed to write updated data: %w", err)
	}
	
	fmt.Printf("Added %d new candles for %s\n", len(newCandles), coinID)
	fmt.Printf("Total candles: %d (from %s to %s)\n\n",
		len(allCandles),
		allCandles[0].Timestamp.Format("2006-01-02"),
		allCandles[len(allCandles)-1].Timestamp.Format("2006-01-02"),
	)
	
	return nil
}

// ScrapeAll fetches data for all supported coins
func (s *DataScraper) ScrapeAll(ctx context.Context) error {
	coins := s.SupportedCoins()
	
	for i, coinID := range coins {
		if err := s.ScrapeCoin(ctx, coinID); err != nil {
			return err
		}
		
		if i < len(coins)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.rateLimit):
			}
		}
	}
	
	return nil
}

// UpdateAll updates all existing coin data
func (s *DataScraper) UpdateAll(ctx context.Context) error {
	coins := s.SupportedCoins()
	
	for i, coinID := range coins {
		if err := s.UpdateCoin(ctx, coinID); err != nil {
			fmt.Printf("Warning: failed to update %s: %v\n", coinID, err)
			continue
		}
		
		if i < len(coins)-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(s.rateLimit):
			}
		}
	}
	
	return nil
}

// GetCoinData loads candle data for a coin
func (s *DataScraper) GetCoinData(coinID string) ([]engine.Candle, error) {
	filename := filepath.Join(s.dataDir, fmt.Sprintf("%s_daily.csv", coinID))
	return s.readCandles(filename)
}

// writeCandles writes candles to CSV file
func (s *DataScraper) writeCandles(filename string, candles []engine.Candle) error {
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
	
	if err := writer.Write([]string{"timestamp", "open", "high", "low", "close", "volume"}); err != nil {
		return err
	}
	
	for _, c := range candles {
		record := []string{
			c.Timestamp.Format(time.RFC3339),
			fmt.Sprintf("%.8f", c.Open),
			fmt.Sprintf("%.8f", c.High),
			fmt.Sprintf("%.8f", c.Low),
			fmt.Sprintf("%.8f", c.Close),
			fmt.Sprintf("%.8f", c.Volume),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	
	return nil
}

// readCandles reads candles from CSV file
func (s *DataScraper) readCandles(filename string) ([]engine.Candle, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	
	if len(records) <= 1 {
		return []engine.Candle{}, nil
	}
	
	candles := make([]engine.Candle, 0, len(records)-1)
	for i, record := range records {
		if i == 0 {
			continue
		}
		
		if len(record) < 6 {
			continue
		}
		
		timestamp, err := time.Parse(time.RFC3339, record[0])
		if err != nil {
			continue
		}
		
		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		close, _ := strconv.ParseFloat(record[4], 64)
		volume, _ := strconv.ParseFloat(record[5], 64)
		
		candles = append(candles, engine.Candle{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}
	
	return candles, nil
}

// mergeCandles combines existing and new candles, removing duplicates
func (s *DataScraper) mergeCandles(existing, new []engine.Candle) []engine.Candle {
	candleMap := make(map[int64]engine.Candle)
	
	for _, c := range existing {
		candleMap[c.Timestamp.Unix()] = c
	}
	
	for _, c := range new {
		candleMap[c.Timestamp.Unix()] = c
	}
	
	merged := make([]engine.Candle, 0, len(candleMap))
	for _, c := range candleMap {
		merged = append(merged, c)
	}
	
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Timestamp.Before(merged[j].Timestamp)
	})
	
	return merged
}

// GetDataInfo returns information about existing data files
func (s *DataScraper) GetDataInfo() (map[string]DataInfo, error) {
	info := make(map[string]DataInfo)
	
	for _, coinID := range s.SupportedCoins() {
		filename := filepath.Join(s.dataDir, fmt.Sprintf("%s_daily.csv", coinID))
		
		candles, err := s.readCandles(filename)
		if err != nil {
			continue
		}
		
		if len(candles) == 0 {
			continue
		}
		
		stat, _ := os.Stat(filename)
		
		info[coinID] = DataInfo{
			CoinID:      coinID,
			TotalCandles: len(candles),
			FirstDate:   candles[0].Timestamp,
			LastDate:    candles[len(candles)-1].Timestamp,
			FileSize:    stat.Size(),
			FilePath:    filename,
		}
	}
	
	return info, nil
}

// DataInfo contains information about a data file
type DataInfo struct {
	CoinID       string
	TotalCandles int
	FirstDate    time.Time
	LastDate     time.Time
	FileSize     int64
	FilePath     string
}
