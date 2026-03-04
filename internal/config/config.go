package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds all harness configuration.
type Config struct {
	Backend    string        `toml:"backend"`
	Model      string        `toml:"model"`
	BudgetUSD  float64       `toml:"budget_usd"`
	MaxRetries int           `toml:"max_retries"`
	MaxTurns   int           `toml:"max_turns"`
	Timeout    time.Duration `toml:"-"`
	TimeoutStr string        `toml:"timeout"`
	VerifyCmd  string        `toml:"verify_cmd"`

	Review ReviewConfig `toml:"review"`
}

// ReviewConfig holds review-specific settings.
type ReviewConfig struct {
	Model     string  `toml:"model"`
	BudgetUSD float64 `toml:"budget_usd"`
}

// configFile is the raw TOML structure.
type configFile struct {
	Defaults struct {
		Backend    string  `toml:"backend"`
		Model      string  `toml:"model"`
		BudgetUSD  float64 `toml:"budget_usd"`
		MaxRetries int     `toml:"max_retries"`
		MaxTurns   int     `toml:"max_turns"`
		Timeout    string  `toml:"timeout"`
		VerifyCmd  string  `toml:"verify_cmd"`
	} `toml:"defaults"`
	Review ReviewConfig `toml:"review"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Backend:    "claude",
		Model:      "opus",
		BudgetUSD:  5.00,
		MaxRetries: 3,
		MaxTurns:   50,
		Timeout:    10 * time.Minute,
		TimeoutStr: "10m",
		VerifyCmd:  "just verify",
		Review: ReviewConfig{
			Model:     "sonnet",
			BudgetUSD: 2.00,
		},
	}
}

// Load loads configuration from files and environment, applying overrides.
// Priority: CLI flags (applied by caller) > env vars > ./harness.toml > ~/.config/harness/config.toml
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Load global config
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".config", "harness", "config.toml")
		_ = loadFromFile(cfg, globalPath)
	}

	// Load local config (overrides global)
	_ = loadFromFile(cfg, "harness.toml")

	// Environment variables (override files)
	applyEnv(cfg)

	return cfg, nil
}

func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var f configFile
	if err := toml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	if f.Defaults.Backend != "" {
		cfg.Backend = f.Defaults.Backend
	}
	if f.Defaults.Model != "" {
		cfg.Model = f.Defaults.Model
	}
	if f.Defaults.BudgetUSD > 0 {
		cfg.BudgetUSD = f.Defaults.BudgetUSD
	}
	if f.Defaults.MaxRetries > 0 {
		cfg.MaxRetries = f.Defaults.MaxRetries
	}
	if f.Defaults.MaxTurns > 0 {
		cfg.MaxTurns = f.Defaults.MaxTurns
	}
	if f.Defaults.Timeout != "" {
		if d, err := time.ParseDuration(f.Defaults.Timeout); err == nil {
			cfg.Timeout = d
			cfg.TimeoutStr = f.Defaults.Timeout
		}
	}
	if f.Defaults.VerifyCmd != "" {
		cfg.VerifyCmd = f.Defaults.VerifyCmd
	}
	if f.Review.Model != "" {
		cfg.Review.Model = f.Review.Model
	}
	if f.Review.BudgetUSD > 0 {
		cfg.Review.BudgetUSD = f.Review.BudgetUSD
	}

	return nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("HARNESS_BACKEND"); v != "" {
		cfg.Backend = v
	}
	if v := os.Getenv("HARNESS_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("HARNESS_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeout = d
			cfg.TimeoutStr = v
		}
	}
	if v := os.Getenv("HARNESS_VERIFY_CMD"); v != "" {
		cfg.VerifyCmd = v
	}
}
