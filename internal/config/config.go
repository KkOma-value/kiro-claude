package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Runtime RuntimeConfig `yaml:"runtime"`
	Kiro    KiroConfig    `yaml:"kiro"`
	Proxy   ProxyConfig   `yaml:"proxy"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig represents server settings
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// RuntimeConfig represents backend runtime settings.
type RuntimeConfig struct {
	Mode                   string `yaml:"mode"`
	MockScenario           string `yaml:"mock_scenario"`
	UpstreamEndpoint       string `yaml:"upstream_endpoint"`
	AllowStartWithoutToken bool   `yaml:"allow_start_without_token"`
}

// KiroConfig represents Kiro-specific settings
type KiroConfig struct {
	CacheDir           string `yaml:"cache_dir"`
	EnableMultiAccount bool   `yaml:"enable_multi_account"`
	RefreshInterval    int    `yaml:"refresh_interval"` // seconds
}

// ProxyConfig represents proxy settings
type ProxyConfig struct {
	HTTPProxy   string `yaml:"http_proxy"`
	HTTPSProxy  string `yaml:"https_proxy"`
	SOCKS5Proxy string `yaml:"socks5_proxy"`
}

// LoggingConfig represents logging settings
type LoggingConfig struct {
	Level  string `yaml:"level"`  // "debug", "info", "warn", "error"
	Format string `yaml:"format"` // "json", "text"
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8000,
		},
		Runtime: RuntimeConfig{
			Mode:                   "kiro-live",
			MockScenario:           "default",
			UpstreamEndpoint:       "",
			AllowStartWithoutToken: true,
		},
		Kiro: KiroConfig{
			CacheDir:           "~/.aws/sso/cache",
			EnableMultiAccount: true,
			RefreshInterval:    60,
		},
		Proxy: ProxyConfig{
			HTTPProxy:   "",
			HTTPSProxy:  "",
			SOCKS5Proxy: "",
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// LoadConfig loads configuration from a YAML file, with environment variable overrides
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// Try to load from file if provided
	if configPath != "" {
		// Expand ~ to home directory
		if strings.HasPrefix(configPath, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to expand ~: %w", err)
			}
			configPath = filepath.Join(home, configPath[1:])
		}

		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// If file doesn't exist, just use defaults + env overrides
	}

	// Apply environment variable overrides
	applyEnvOverrides(config)

	return config, nil
}

// applyEnvOverrides applies environment variable overrides to the config
func applyEnvOverrides(config *Config) {
	if host := os.Getenv("KIRO_CLAUDE_HOST"); host != "" {
		config.Server.Host = host
	}
	if port := os.Getenv("KIRO_CLAUDE_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &config.Server.Port)
	}
	if cacheDir := os.Getenv("KIRO_CACHE_DIR"); cacheDir != "" {
		config.Kiro.CacheDir = cacheDir
	}
	if mode := os.Getenv("KIRO_CLAUDE_MODE"); mode != "" {
		config.Runtime.Mode = mode
	}
	if scenario := os.Getenv("KIRO_CLAUDE_MOCK_SCENARIO"); scenario != "" {
		config.Runtime.MockScenario = scenario
	}
	if endpoint := os.Getenv("KIRO_UPSTREAM_ENDPOINT"); endpoint != "" {
		config.Runtime.UpstreamEndpoint = endpoint
	}
	if allow := os.Getenv("KIRO_ALLOW_START_WITHOUT_TOKEN"); allow != "" {
		config.Runtime.AllowStartWithoutToken = strings.EqualFold(allow, "true") || allow == "1"
	}
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Logging.Level = level
	}
	if httpProxy := os.Getenv("HTTP_PROXY"); httpProxy != "" {
		config.Proxy.HTTPProxy = httpProxy
	}
	if httpsProxy := os.Getenv("HTTPS_PROXY"); httpsProxy != "" {
		config.Proxy.HTTPSProxy = httpsProxy
	}
}
