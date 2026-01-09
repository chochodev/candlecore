package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"candlecore/internal/api"
	"candlecore/internal/scraper"
	"candlecore/internal/ui"

	"github.com/spf13/cobra"
)

var (
	dataDir string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "candlecore",
	Short: "Candlecore - Algorithmic Crypto Trading Engine",
	Long: `Candlecore is a production-ready algorithmic trading engine for cryptocurrency markets.
	
Features:
  - Backtesting with historical data
  - Paper trading simulation
  - REST API for frontend integration
  - PostgreSQL state persistence
  - Strategy optimization`,
}

// dataCmd represents the data command group
var dataCmd = &cobra.Command{
	Use:   "data",
	Short: "Manage historical market data",
	Long:  "Download, update, and manage cryptocurrency historical data for backtesting.",
}

// scrapeCmd downloads initial historical data
var scrapeCmd = &cobra.Command{
	Use:   "scrape [coin]",
	Short: "Download maximum historical data for a coin",
	Long:  "Downloads up to 365 days of daily OHLCV data from CoinGecko.\n\nSupported coins: bitcoin, ethereum\nLeave empty to scrape all coins.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintBanner()
		ui.PrintSection("DATA SCRAPING")
		
		s := scraper.NewDataScraper(dataDir)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		if len(args) == 0 {
			ui.PrintInfo("Scraping all supported coins...")
			fmt.Println()
			if err := s.ScrapeAll(ctx); err != nil {
				ui.PrintError(fmt.Sprintf("Scraping failed: %v", err))
				os.Exit(1)
			}
		} else {
			coinID := args[0]
			ui.PrintInfo(fmt.Sprintf("Scraping %s data...", coinID))
			fmt.Println()
			if err := s.ScrapeCoin(ctx, coinID); err != nil {
				ui.PrintError(fmt.Sprintf("Scraping failed: %v", err))
				os.Exit(1)
			}
		}
		
		ui.PrintSuccess("Data scraping completed")
		fmt.Println()
	},
}

// updateCmd updates existing data
var updateCmd = &cobra.Command{
	Use:   "update [coin]",
	Short: "Update existing data with latest candles",
	Long:  "Appends new data since last scrape without re-downloading everything.\n\nLeave empty to update all coins.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintBanner()
		ui.PrintSection("DATA UPDATE")
		
		s := scraper.NewDataScraper(dataDir)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		if len(args) == 0 {
			ui.PrintInfo("Updating all coins...")
			fmt.Println()
			if err := s.UpdateAll(ctx); err != nil {
				ui.PrintError(fmt.Sprintf("Update failed: %v", err))
				os.Exit(1)
			}
		} else {
			coinID := args[0]
			ui.PrintInfo(fmt.Sprintf("Updating %s data...", coinID))
			fmt.Println()
			if err := s.UpdateCoin(ctx, coinID); err != nil {
				ui.PrintError(fmt.Sprintf("Update failed: %v", err))
				os.Exit(1)
			}
		}
		
		ui.PrintSuccess("Data update completed")
		fmt.Println()
	},
}

// listCmd shows available data
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available data files",
	Long:  "Shows information about downloaded cryptocurrency data.",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintBanner()
		ui.PrintSection("AVAILABLE DATA")
		
		s := scraper.NewDataScraper(dataDir)
		info, err := s.GetDataInfo()
		if err != nil {
			ui.PrintError(fmt.Sprintf("Failed to get data info: %v", err))
			os.Exit(1)
		}
		
		if len(info) == 0 {
			ui.PrintWarning("No data files found. Run 'candlecore data scrape' to download data.")
			return
		}
		
		fmt.Printf("  %-12s %-10s %-12s %-12s %-10s\n", "Coin", "Candles", "From", "To", "Size")
		fmt.Println("  " + string(make([]byte, 70)))
		
		for coinID, data := range info {
			sizeKB := float64(data.FileSize) / 1024
			fmt.Printf("  %-12s %-10d %-12s %-12s %.2f KB\n",
				coinID,
				data.TotalCandles,
				data.FirstDate.Format("2006-01-02"),
				data.LastDate.Format("2006-01-02"),
				sizeKB,
			)
		}
		
		fmt.Println()
	},
}

// serveCmd starts the API server
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long:  "Starts the HTTP API server for frontend access.\n\nDefault port: 8080",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		
		ui.PrintBanner()
		ui.PrintSection("API SERVER")
		
		ui.PrintInfo(fmt.Sprintf("Starting API server on port %s", port))
		ui.PrintSuccess(fmt.Sprintf("API endpoint: http://localhost:%s/api/v1", port))
		ui.PrintInfo("Press Ctrl+C to stop")
		fmt.Println()
		
		server := api.NewServer(dataDir)
		if err := server.Run(port); err != nil {
			ui.PrintError(fmt.Sprintf("Server failed: %v", err))
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "data/historical", "Directory for storing historical data")
	
	serveCmd.Flags().StringP("port", "p", "8080", "API server port")
	
	dataCmd.AddCommand(scrapeCmd)
	dataCmd.AddCommand(updateCmd)
	dataCmd.AddCommand(listCmd)
	
	rootCmd.AddCommand(dataCmd)
	rootCmd.AddCommand(serveCmd)
}

// Execute runs the CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
