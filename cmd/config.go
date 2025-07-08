package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	APIKey string `json:"api_key"`
}

// getConfigDir returns the directory where config files should be stored
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}
	
	configDir := filepath.Join(homeDir, ".hyperbolic")
	return configDir, nil
}

// getConfigPath returns the full path to the config file
func getConfigPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}
	
	return filepath.Join(configDir, "config.json"), nil
}

// SaveConfig saves the configuration to disk
func SaveConfig(config *Config) error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}
	
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	
	// Write to file with restricted permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}
	
	return nil
}

// LoadConfig loads the configuration from disk
func LoadConfig() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, err
	}
	
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found - please run 'hyperbolic auth' first")
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}
	
	// Unmarshal JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}
	
	return &config, nil
}

// GetAPIKey returns the stored API key
func GetAPIKey() (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}
	
	if config.APIKey == "" {
		return "", fmt.Errorf("no API key found in config - please run 'hyperbolic auth' first")
	}
	
	return config.APIKey, nil
} 