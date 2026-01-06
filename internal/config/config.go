package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	// Trading configuration
	InitialBalance float64 `yaml:"initial_balance"`
	TakerFee       float64 `yaml:"taker_fee"`  // e.g., 0.001 for 0.1%
	MakerFee       float64 `yaml:"maker_fee"`  // e.g., 0.0005 for 0.05%
	SlippageBps    float64 `yaml:"slippage_bps"` // basis points, e.g., 5 for 0.05%

	// Data configuration
	DataSource     string `yaml:"data_source"` // Path to candle data file
	StateDirectory string `yaml:"state_directory"` // Where to save/load state

	// Logging
	LogLevel string `yaml:"log_level"` // debug, info, warn, error

	// Strategy configuration
	Strategy StrategyConfig `yaml:"strategy"`
}

// StrategyConfig holds strategy-specific parameters
type StrategyConfig struct {
	Name         string  `yaml:"name"`
	FastPeriod   int     `yaml:"fast_period"`
	SlowPeriod   int     `yaml:"slow_period"`
	PositionSize float64 `yaml:"position_size"` // How much to invest per trade
}

// Load reads configuration from a YAML file with environment variable overrides
func Load(path string) (*Config, error) {
	// Set defaults
	cfg := &Config{
		InitialBalance: 10000.0,
		TakerFee:       0.001,
		MakerFee:       0.0005,
		SlippageBps:    5.0,
		DataSource:     "data/candles.csv",
		StateDirectory: ".state",
		LogLevel:       "info",
		Strategy: StrategyConfig{
			Name:         "simple_ma",
			FastPeriod:   10,
			SlowPeriod:   30,
			PositionSize: 1000.0,
		},
	}

	// Try to load from file
	if _, err := os.Stat(path); err == nil {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables if present
	if val := os.Getenv("CANDLECORE_INITIAL_BALANCE"); val != "" {
		var balance float64
		if _, err := fmt.Sscanf(val, "%f", &balance); err == nil {
			cfg.InitialBalance = balance
		}
	}

	if val := os.Getenv("CANDLECORE_LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}

	if val := os.Getenv("CANDLECORE_DATA_SOURCE"); val != "" {
		cfg.DataSource = val
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.InitialBalance <= 0 {
		return fmt.Errorf("initial_balance must be positive")
	}

	if c.TakerFee < 0 || c.TakerFee > 1 {
		return fmt.Errorf("taker_fee must be between 0 and 1")
	}

	if c.MakerFee < 0 || c.MakerFee > 1 {
		return fmt.Errorf("maker_fee must be between 0 and 1")
	}

	if c.SlippageBps < 0 {
		return fmt.Errorf("slippage_bps must be non-negative")
	}

	if c.Strategy.FastPeriod <= 0 || c.Strategy.SlowPeriod <= 0 {
		return fmt.Errorf("strategy periods must be positive")
	}

	if c.Strategy.FastPeriod >= c.Strategy.SlowPeriod {
		return fmt.Errorf("fast_period must be less than slow_period")
	}

	return nil
}
