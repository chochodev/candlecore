package config

// Config holds application configuration
type Config struct {
	Server   ServerConfig
	Bot      BotConfig
	Exchange ExchangeConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port       string
	DataDir    string
	EnableCORS bool
}

// BotConfig holds bot configuration
type BotConfig struct {
	DryRun         bool    // Dry-run mode (no real trades)
	InitialBalance float64 // Initial balance for dry-run
	DefaultSymbol  string
	DefaultTimeframe string
	DefaultStrategy string
}

// ExchangeConfig holds exchange configuration
type ExchangeConfig struct {
	Name   string
	APIKey string
	Secret string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Port:       "8080",
			DataDir:    "data/historical",
			EnableCORS: true,
		},
		Bot: BotConfig{
			DryRun:         true, // Default to dry-run for safety
			InitialBalance: 10000.0,
			DefaultSymbol:  "bitcoin",
			DefaultTimeframe: "1h",
			DefaultStrategy: "ma_crossover",
		},
		Exchange: ExchangeConfig{
			Name: "local",
		},
	}
}
