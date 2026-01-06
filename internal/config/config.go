package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
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

	// Database configuration (optional, for PostgreSQL state store)
	Database DatabaseConfig `yaml:"database"`

	// Logging
	LogLevel string `yaml:"log_level"` // debug, info, warn, error

	// Strategy configuration
	Strategy StrategyConfig `yaml:"strategy"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Enabled  bool   `yaml:"enabled"`  // Use database instead of file store
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"` // disable, require, verify-ca, verify-full
	AccountID int64 `yaml:"account_id"` // Which account ID to use
}

// StrategyConfig holds strategy-specific parameters
type StrategyConfig struct {
	Name         string  `yaml:"name"`
	FastPeriod   int     `yaml:"fast_period"`
	SlowPeriod   int     `yaml:"slow_period"`
	PositionSize float64 `yaml:"position_size"` // How much to invest per trade
}

// Load reads configuration from a YAML file with environment variable overrides
// It first attempts to load a .env file if present
func Load(path string) (*Config, error) {
	// Try to load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using defaults and config.yaml")
	} else {
		fmt.Println("Loaded configuration from .env file")
	}

	// Set defaults
	cfg := &Config{
		InitialBalance: 10000.0,
		TakerFee:       0.001,
		MakerFee:       0.0005,
		SlippageBps:    5.0,
		DataSource:     "data/candles.csv",
		StateDirectory: ".state",
		LogLevel:       "info",
		Database: DatabaseConfig{
			Enabled:   false,
			Host:      "localhost",
			Port:      5432,
			User:      "candlecore",
			Password:  "",
			DBName:    "candlecore",
			SSLMode:   "disable",
			AccountID: 1,
		},
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

	// Override with environment variables
	applyEnvOverrides(cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(cfg *Config) {
	// Trading settings
	if val := os.Getenv("CANDLECORE_INITIAL_BALANCE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.InitialBalance = f
		}
	}

	if val := os.Getenv("CANDLECORE_TAKER_FEE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.TakerFee = f
		}
	}

	if val := os.Getenv("CANDLECORE_MAKER_FEE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.MakerFee = f
		}
	}

	if val := os.Getenv("CANDLECORE_SLIPPAGE_BPS"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.SlippageBps = f
		}
	}

	// Data and state
	if val := os.Getenv("CANDLECORE_DATA_SOURCE"); val != "" {
		cfg.DataSource = val
	}

	if val := os.Getenv("CANDLECORE_STATE_DIR"); val != "" {
		cfg.StateDirectory = val
	}

	// Logging
	if val := os.Getenv("CANDLECORE_LOG_LEVEL"); val != "" {
		cfg.LogLevel = val
	}

	// Database settings
	if val := os.Getenv("CANDLECORE_DB_ENABLED"); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			cfg.Database.Enabled = b
		}
	}

	if val := os.Getenv("CANDLECORE_DB_HOST"); val != "" {
		cfg.Database.Host = val
	}

	if val := os.Getenv("CANDLECORE_DB_PORT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Database.Port = i
		}
	}

	if val := os.Getenv("CANDLECORE_DB_USER"); val != "" {
		cfg.Database.User = val
	}

	if val := os.Getenv("CANDLECORE_DB_PASSWORD"); val != "" {
		cfg.Database.Password = val
	}

	if val := os.Getenv("CANDLECORE_DB_NAME"); val != "" {
		cfg.Database.DBName = val
	}

	if val := os.Getenv("CANDLECORE_DB_SSLMODE"); val != "" {
		cfg.Database.SSLMode = val
	}

	if val := os.Getenv("CANDLECORE_DB_ACCOUNT_ID"); val != "" {
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			cfg.Database.AccountID = i
		}
	}

	// Strategy settings
	if val := os.Getenv("CANDLECORE_STRATEGY_NAME"); val != "" {
		cfg.Strategy.Name = val
	}

	if val := os.Getenv("CANDLECORE_STRATEGY_FAST_PERIOD"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Strategy.FastPeriod = i
		}
	}

	if val := os.Getenv("CANDLECORE_STRATEGY_SLOW_PERIOD"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			cfg.Strategy.SlowPeriod = i
		}
	}

	if val := os.Getenv("CANDLECORE_STRATEGY_POSITION_SIZE"); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			cfg.Strategy.PositionSize = f
		}
	}
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

	// Validate database config if enabled
	if c.Database.Enabled {
		if c.Database.Host == "" {
			return fmt.Errorf("database host is required when database is enabled")
		}
		if c.Database.Port <= 0 || c.Database.Port > 65535 {
			return fmt.Errorf("database port must be between 1 and 65535")
		}
		if c.Database.User == "" {
			return fmt.Errorf("database user is required when database is enabled")
		}
		if c.Database.DBName == "" {
			return fmt.Errorf("database name is required when database is enabled")
		}
		if c.Database.AccountID <= 0 {
			return fmt.Errorf("database account_id must be positive")
		}
	}

	return nil
}

// GetDatabaseConnectionString builds a PostgreSQL connection string
func (c *Config) GetDatabaseConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}
