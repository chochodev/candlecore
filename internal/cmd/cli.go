package cmd

import (
	"candlecore/internal/api"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dataDir string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "candlecore",
	Short: "Candlecore - Algorithmic Crypto Trading Bot",
	Long: `Candlecore is a production-ready algorithmic trading bot for cryptocurrency markets.

Features:
  - Real-time WebSocket streaming
  - Bot control & configuration
  - Historical replay & backtesting
  - Multiple trading strategies
  - Technical indicators`,
}

// serveCmd starts the API server
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  "Starts the REST API and WebSocket server for bot control and frontend integration.",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetString("port")
		
		fmt.Printf("Starting Candlecore API Server on port %s...\n", port)
		fmt.Printf("Data directory: %s\n", dataDir)
		fmt.Println()
		
		server := api.NewServer(dataDir)
		
		if err := server.Run(port); err != nil {
			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "data/historical", "Directory for storing historical data")
	
	serveCmd.Flags().StringP("port", "p", "8080", "Port to run the server on")
	
	rootCmd.AddCommand(serveCmd)
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
