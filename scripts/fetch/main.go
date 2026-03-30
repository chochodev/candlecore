package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Config
var symbols = []string{"BTC", "ETH", "SOL"}
var timeframes = []string{"d", "1h", "15m"}

func main() {
	fmt.Println("🚀 Candlecore Historical Data Fetcher")
	
	for _, symbol := range symbols {
		for _, tf := range timeframes {
			url := fmt.Sprintf("https://www.cryptodatadownload.com/cdd/Binance_%sUSDT_%s.csv", symbol, tf)
			tempFile := fmt.Sprintf("data/historical/temp_%s_%s.csv", symbol, tf)
			
			// Set target filename
			suffix := tf
			if tf == "d" { suffix = "1d" }
			targetFile := fmt.Sprintf("data/historical/%s_%s.csv", strings.ToLower(symbol), suffix)

			fmt.Printf("📥 Downloading %s (%s)... ", symbol, tf)
			if err := downloadFile(url, tempFile); err != nil {
				fmt.Printf("❌ Failed: %v\n", err)
				continue
			}
			fmt.Println("✅")

			fmt.Printf("🔄 Transforming to %s... ", targetFile)
			if err := transform(tempFile, targetFile); err != nil {
				fmt.Printf("❌ Failed: %v\n", err)
			} else {
				fmt.Println("✅ Done")
			}

			// Clean up temp file
			os.Remove(tempFile)
		}
	}
	fmt.Println("\n✨ Data update complete!")
}

func downloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func transform(inputPath, outputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	
	// Skip 1st line (URL)
	_, _ = reader.Read()
	
	// Read header
	header, err := reader.Read()
	if err != nil {
		return err
	}

	dateIdx, openIdx, highIdx, lowIdx, closeIdx, volumeIdx := -1, -1, -1, -1, -1, -1
	for i, col := range header {
		name := strings.ToLower(strings.TrimSpace(col))
		if name == "date" { dateIdx = i }
		if name == "open" { openIdx = i }
		if name == "high" { highIdx = i }
		if name == "low" { lowIdx = i }
		if name == "close" { closeIdx = i }
		if strings.HasPrefix(name, "volume") && !strings.Contains(name, "usdt") { volumeIdx = i }
	}

	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	writer := csv.NewWriter(out)
	defer writer.Flush()
	writer.Write([]string{"timestamp", "open", "high", "low", "close", "volume"})

	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		if dateIdx >= len(rec) { continue }
		
		t, err := parseDate(rec[dateIdx])
		if err != nil { continue }

		writer.Write([]string{
			t.Format(time.RFC3339),
			rec[openIdx],
			rec[highIdx],
			rec[lowIdx],
			rec[closeIdx],
			rec[volumeIdx],
		})
	}
	return nil
}

func parseDate(dateStr string) (time.Time, error) {
	if strings.Contains(dateStr, ":") {
		// Try full date-time format
		return time.Parse("2006-01-02 15:04:05", dateStr)
	}
	// Try daily date format
	return time.Parse("2006-01-02", dateStr)
}
