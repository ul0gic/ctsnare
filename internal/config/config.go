// Package config handles configuration loading and defaults for ctsnare.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ul0gic/ctsnare/internal/domain"
)

// CTLogConfig defines the URL and human-readable name for a single CT log.
type CTLogConfig struct {
	URL  string `toml:"url"`
	Name string `toml:"name"`
}

// Config holds all configurable values for ctsnare.
type Config struct {
	CTLogs         []CTLogConfig              `toml:"ct_logs"`
	DefaultProfile string                     `toml:"default_profile"`
	BatchSize      int                        `toml:"batch_size"`
	PollInterval   time.Duration              `toml:"poll_interval"`
	DBPath         string                     `toml:"db_path"`
	CustomProfiles map[string]domain.Profile  `toml:"custom_profiles"`
	SkipSuffixes   []string                   `toml:"skip_suffixes"`
}

// DefaultConfig returns a Config with sensible production defaults.
func DefaultConfig() *Config {
	return &Config{
		CTLogs: []CTLogConfig{
			{
				URL:  "https://ct.googleapis.com/logs/us1/argon2025h1",
				Name: "Google Argon 2025h1",
			},
			{
				URL:  "https://ct.googleapis.com/logs/us1/argon2025h2",
				Name: "Google Argon 2025h2",
			},
			{
				URL:  "https://ct.googleapis.com/logs/eu1/xenon2025h1",
				Name: "Google Xenon 2025h1",
			},
		},
		DefaultProfile: "all",
		BatchSize:      256,
		PollInterval:   5 * time.Second,
		DBPath:         defaultDBPath(),
		CustomProfiles: make(map[string]domain.Profile),
		SkipSuffixes:   defaultSkipSuffixes(),
	}
}

// Load reads a TOML config file and returns a Config with defaults applied
// for any missing values. If the file does not exist, it returns the default
// config without error. An empty path also returns defaults.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	applyDefaults(cfg)

	return cfg, nil
}

// MergeFlags applies CLI flag overrides to the config. Zero values are
// treated as "not set" and do not override.
func MergeFlags(cfg *Config, dbPath string, batchSize int, pollInterval time.Duration) {
	if dbPath != "" {
		cfg.DBPath = dbPath
	}
	if batchSize > 0 {
		cfg.BatchSize = batchSize
	}
	if pollInterval > 0 {
		cfg.PollInterval = pollInterval
	}
}

// applyDefaults fills in zero-valued fields with sensible defaults.
func applyDefaults(cfg *Config) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 256
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Second
	}
	if cfg.DBPath == "" {
		cfg.DBPath = defaultDBPath()
	}
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = "all"
	}
	if len(cfg.CTLogs) == 0 {
		cfg.CTLogs = DefaultConfig().CTLogs
	}
	if cfg.CustomProfiles == nil {
		cfg.CustomProfiles = make(map[string]domain.Profile)
	}
	if len(cfg.SkipSuffixes) == 0 {
		cfg.SkipSuffixes = defaultSkipSuffixes()
	}
}

// defaultDBPath returns the XDG-compliant database path.
func defaultDBPath() string {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "ctsnare.db"
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "ctsnare", "ctsnare.db")
}

// defaultSkipSuffixes returns the default list of domain suffixes to skip
// during scoring. These are infrastructure domains that generate noise.
func defaultSkipSuffixes() []string {
	return []string{
		"cloudflaressl.com",
		"amazonaws.com",
		"herokuapp.com",
		"azurewebsites.net",
		"googleusercontent.com",
		"fastly.net",
		"akamaiedge.net",
		"cloudfront.net",
		"github.io",
		"gitlab.io",
		"netlify.app",
		"vercel.app",
		"firebaseapp.com",
		"appspot.com",
		"trafficmanager.net",
		"azure-api.net",
	}
}
