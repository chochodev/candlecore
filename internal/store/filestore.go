package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"candlecore/internal/engine"
)

// FileStore implements StateStore using JSON files
// This allows the engine to resume from where it left off
type FileStore struct {
	directory string
}

// NewFileStore creates a new file-based state store
func NewFileStore(directory string) (*FileStore, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	return &FileStore{
		directory: directory,
	}, nil
}

// SaveState persists the broker state to disk
func (s *FileStore) SaveState(broker engine.Broker) error {
	account := broker.GetAccount()

	// Serialize account state to JSON
	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to file
	statePath := filepath.Join(s.directory, "account.json")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadState restores the broker state from disk
func (s *FileStore) LoadState(broker engine.Broker) error {
	statePath := filepath.Join(s.directory, "account.json")

	// Check if state file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return fmt.Errorf("state file does not exist")
	}

	// Read state file
	data, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Deserialize account state
	var account engine.Account
	if err := json.Unmarshal(data, &account); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Note: In a real implementation, you'd need to restore the broker's
	// internal state. For PaperBroker, this would mean setting balance,
	// positions, etc. This would require the broker to expose a SetState
	// method or similar. For now, this is a simplified version.

	// TODO: Implement broker state restoration
	// This would require extending the Broker interface with a SetState method

	return nil
}
