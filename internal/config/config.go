package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Targets []Target `mapstructure:"targets"`
	Timeout int      `mapstructure:"timeout"` // seconds
	Workers int      `mapstructure:"workers"` // concurrent workers
	LogFile string   `mapstructure:"log_file"`
}

type Target struct {
	Name    string `mapstructure:"name"`
	BaseURL string `mapstructure:"base_url"` // Optional, defaults to api.anthropic.com
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"` // e.g., claude-3-5-haiku-20241022
}

// Load reads configuration from file and environment variables
func Load(configPath string) (*Config, error) {
	// Load .env file if it exists (ignore error if not found)
	_ = godotenv.Load()

	// Set defaults
	viper.SetDefault("timeout", 30)
	viper.SetDefault("workers", 5)
	viper.SetDefault("log_file", "debug.log")

	// Enable environment variable support
	viper.SetEnvPrefix("APIHEALTH")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read config file if provided
	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("reading config file: %w", err)
			}
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Expand environment variables in target configurations
	for i := range cfg.Targets {
		cfg.Targets[i].APIKey = os.ExpandEnv(cfg.Targets[i].APIKey)
		cfg.Targets[i].BaseURL = os.ExpandEnv(cfg.Targets[i].BaseURL)

		// Set default base URL if not provided
		if cfg.Targets[i].BaseURL == "" {
			cfg.Targets[i].BaseURL = "https://api.anthropic.com"
		}

		// Remove trailing slash from base URL
		cfg.Targets[i].BaseURL = strings.TrimSuffix(cfg.Targets[i].BaseURL, "/")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Targets) == 0 {
		return fmt.Errorf("no targets configured")
	}

	for i, target := range c.Targets {
		if target.Name == "" {
			return fmt.Errorf("target %d: name is required", i)
		}
		if target.APIKey == "" {
			return fmt.Errorf("target %s: api_key is required", target.Name)
		}
		if target.Model == "" {
			return fmt.Errorf("target %s: model is required", target.Name)
		}
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.Workers <= 0 {
		return fmt.Errorf("workers must be positive")
	}

	return nil
}