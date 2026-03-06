package config

import (
	"github.com/spf13/viper"
)

// Option is a functional option for configuring the Manager.
type Option func(*Manager)

// WithConfigFile sets the configuration file path.
func WithConfigFile(path string) Option {
	return func(m *Manager) {
		m.configFile = path
	}
}

// WithNacos enables Nacos configuration center.
func WithNacos(cfg NacosConfig) Option {
	return func(m *Manager) {
		m.nacosConfig = &cfg
	}
}

// WithEnvPrefix sets the environment variable prefix.
// Environment variables should be in format: PREFIX_KEY (e.g., PROMETAU_DATABASE_HOST)
func WithEnvPrefix(prefix string) Option {
	return func(m *Manager) {
		m.envPrefix = prefix
	}
}

// WithViper sets a custom viper instance.
// Useful for testing or advanced configuration.
func WithViper(v *viper.Viper) Option {
	return func(m *Manager) {
		m.v = v
	}
}

// WithDefaults sets default configuration values.
func WithDefaults(cfg *Config) Option {
	return func(m *Manager) {
		m.defaults = cfg
	}
}
