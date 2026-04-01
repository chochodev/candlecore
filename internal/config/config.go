package config

import (
	"candlecore/internal/exchange"
	"encoding/json"
	"os"
	"path/filepath"
)

// BotProfile represents the persistent configuration for a Pulse instance
type BotProfile struct {
	Symbol      string                 `json:"symbol"`
	Timeframe   exchange.Timeframe      `json:"timeframe"`
	Strategy    string                 `json:"strategy"`
	ReplayMode  bool                   `json:"replay_mode"`
	DryRun      bool                   `json:"dry_run"`
	ReplaySpeed float64                `json:"replay_speed"`
	Parameters  map[string]interface{} `json:"parameters"`
	IsRunning   bool                   `json:"is_running"`
}

// ConfigManager handles JSON persistence on disk
type ConfigManager struct {
	path string
}

func NewConfigManager(basePath string) *ConfigManager {
	os.MkdirAll(basePath, 0755)
	return &ConfigManager{path: filepath.Join(basePath, "bot_profile.json")}
}

// Save persists the current profile to disk
func (m *ConfigManager) Save(profile BotProfile) error {
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}

// Load retrieves the profile from disk
func (m *ConfigManager) Load() (*BotProfile, error) {
	if _, err := os.Stat(m.path); os.IsNotExist(err) {
		return nil, nil // First time setup
	}
	data, err := os.ReadFile(m.path)
	if err != nil {
		return nil, err
	}
	var profile BotProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}
