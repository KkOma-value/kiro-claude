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
	Server   ServerConfig   `yaml:"server"`
	Runtime  RuntimeConfig  `yaml:"runtime"`
	Kiro     KiroConfig     `yaml:"kiro"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
}

// ServerConfig represents server settings
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// RuntimeConfig represents backend runtime settings.
type RuntimeConfig struct {
	Mode                   string `yaml:"mode"`
	UpstreamEndpoint       string `yaml:"upstream_endpoint"`
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

// SecurityConfig represents proxy auth settings
type SecurityConfig struct {
	ProxyAPIKey string `yaml:"proxy_api_key"` // empty = no auth required
}

// LoggingConfig represents logging settings
type LoggingConfig struct {
	Level     string `yaml:"level"`      // "debug", "info", "warn", "error"
	Format    string `yaml:"format"`     // "json", "text"
	DebugDump bool   `yaml:"debug_dump"` // dump request/response to log
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
			UpstreamEndpoint:       "https://prod.us-east-1.codewhisperer.desktop.kiro.dev",
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
		Security: SecurityConfig{
			ProxyAPIKey: "",
		},
		Logging: LoggingConfig{
			Level:     "info",
			Format:    "text",
			DebugDump: false,
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
	if endpoint := os.Getenv("KIRO_UPSTREAM_ENDPOINT"); endpoint != "" {
		config.Runtime.UpstreamEndpoint = endpoint
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
	if apiKey := os.Getenv("KIRO_PROXY_API_KEY"); apiKey != "" {
		config.Security.ProxyAPIKey = apiKey
	}
	if debugDump := os.Getenv("KIRO_DEBUG_DUMP"); debugDump != "" {
		config.Logging.DebugDump = strings.EqualFold(debugDump, "true") || debugDump == "1"
	}
}
