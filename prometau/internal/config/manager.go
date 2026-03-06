package config

import (
	"errors"
	"log/slog"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

// DefaultEnvPrefix is the default prefix for environment variables.
const DefaultEnvPrefix = "PROMETAU"

// ErrNotLoaded is returned when trying to access config before Load() is called.
var ErrNotLoaded = errors.New("configuration not loaded")

// Manager manages application configuration from multiple sources.
// Priority: Nacos > Environment Variables > Config File > Default Values
type Manager struct {
	mu sync.RWMutex

	v           *viper.Viper
	config      *Config
	configFile  string
	envPrefix   string
	defaults    *Config
	nacosConfig *NacosConfig
	nacosClient *NacosClient
	watcher     *Watcher
	loaded      bool
}

// NewManager creates a new configuration manager with the given options.
func NewManager(opts ...Option) *Manager {
	m := &Manager{
		v:          viper.New(),
		envPrefix:  DefaultEnvPrefix,
		defaults:   DefaultConfig(),
		configFile: "configs/config.yaml",
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// Load loads configuration from all configured sources.
// Order: Defaults -> Config File -> Environment Variables -> Nacos
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set default values
	m.setDefaults()

	// Load from config file
	if m.configFile != "" {
		m.v.SetConfigFile(m.configFile)
		if err := m.v.ReadInConfig(); err != nil {
			if !errors.Is(err, viper.ConfigFileNotFoundError{}) {
				slog.Warn("failed to read config file", "path", m.configFile, "error", err)
			}
			// Continue without config file - use defaults and env vars
		} else {
			slog.Info("loaded config file", "path", m.configFile)
		}
	}

	// Bind environment variables
	m.v.SetEnvPrefix(m.envPrefix)
	m.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	m.v.AutomaticEnv()

	// Load from Nacos if configured
	if m.nacosConfig != nil {
		if err := m.loadFromNacos(); err != nil {
			slog.Error("failed to load from nacos", "error", err)
			// Continue with local config - don't fail the application
		}
	}

	// Unmarshal into config struct
	m.config = &Config{}
	if err := m.v.Unmarshal(m.config); err != nil {
		return err
	}

	m.loaded = true
	return nil
}

// setDefaults sets default configuration values.
func (m *Manager) setDefaults() {
	if m.defaults == nil {
		m.defaults = DefaultConfig()
	}

	m.v.SetDefault("server.host", m.defaults.Server.Host)
	m.v.SetDefault("server.port", m.defaults.Server.Port)
	m.v.SetDefault("server.read_timeout", m.defaults.Server.ReadTimeout)
	m.v.SetDefault("server.write_timeout", m.defaults.Server.WriteTimeout)

	m.v.SetDefault("database.driver", m.defaults.Database.Driver)
	m.v.SetDefault("database.host", m.defaults.Database.Host)
	m.v.SetDefault("database.port", m.defaults.Database.Port)
	m.v.SetDefault("database.max_open_conns", m.defaults.Database.MaxOpenConns)
	m.v.SetDefault("database.max_idle_conns", m.defaults.Database.MaxIdleConns)
	m.v.SetDefault("database.conn_max_lifetime", m.defaults.Database.ConnMaxLifetime)

	m.v.SetDefault("redis.host", m.defaults.Redis.Host)
	m.v.SetDefault("redis.port", m.defaults.Redis.Port)
	m.v.SetDefault("redis.db", m.defaults.Redis.DB)

	m.v.SetDefault("log.level", m.defaults.Log.Level)
	m.v.SetDefault("log.format", m.defaults.Log.Format)
	m.v.SetDefault("log.output", m.defaults.Log.Output)
	m.v.SetDefault("log.max_size", m.defaults.Log.MaxSize)
	m.v.SetDefault("log.max_backups", m.defaults.Log.MaxBackups)
	m.v.SetDefault("log.max_age", m.defaults.Log.MaxAge)

	m.v.SetDefault("nacos.server_addr", m.defaults.Nacos.ServerAddr)
	m.v.SetDefault("nacos.namespace", m.defaults.Nacos.Namespace)
	m.v.SetDefault("nacos.data_id", m.defaults.Nacos.DataID)
	m.v.SetDefault("nacos.group", m.defaults.Nacos.Group)

	// Storage defaults
	m.v.SetDefault("storage.type", m.defaults.Storage.Type)
	m.v.SetDefault("storage.minio.endpoint", "115.190.57.118:9000")
	m.v.SetDefault("storage.minio.access_key_id", m.defaults.Storage.MinIO.AccessKeyID)
	m.v.SetDefault("storage.minio.secret_access_key", m.defaults.Storage.MinIO.SecretAccessKey)
	m.v.SetDefault("storage.minio.bucket", m.defaults.Storage.MinIO.Bucket)
	m.v.SetDefault("storage.oss.endpoint", "oss-cn-shanghai.aliyuncs.com")
	m.v.SetDefault("storage.oss.access_key_id", m.defaults.Storage.OSS.AccessKeyID)
	m.v.SetDefault("storage.oss.access_key_secret", m.defaults.Storage.OSS.AccessKeySecret)
	m.v.SetDefault("storage.oss.bucket", m.defaults.Storage.OSS.Bucket)
	m.v.SetDefault("storage.oss.region", m.defaults.Storage.OSS.Region)
}

// loadFromNacos loads configuration from Nacos configuration center.
func (m *Manager) loadFromNacos() error {
	client, err := NewNacosClient(*m.nacosConfig)
	if err != nil {
		return err
	}

	content, err := client.GetConfig()
	if err != nil {
		return err
	}

	m.nacosClient = client

	// Merge Nacos config (highest priority)
	if err := m.v.MergeConfig(strings.NewReader(content)); err != nil {
		return err
	}

	slog.Info("loaded config from nacos",
		"server", m.nacosConfig.ServerAddr,
		"data_id", m.nacosConfig.DataID,
		"group", m.nacosConfig.Group)

	return nil
}

// Get returns the current configuration.
// Returns ErrNotLoaded if Load() hasn't been called.
func (m *Manager) Get() (*Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return nil, ErrNotLoaded
	}
	return m.config, nil
}

// GetKey returns a specific configuration value by key.
func (m *Manager) GetKey(key string) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return nil, ErrNotLoaded
	}
	return m.v.Get(key), nil
}

// Bind binds the configuration to the given struct pointer.
func (m *Manager) Bind(val interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.loaded {
		return ErrNotLoaded
	}
	return m.v.Unmarshal(val)
}

// Watch registers a callback to be called when configuration changes.
// The callback is called in a separate goroutine.
func (m *Manager) Watch(callback func()) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watcher == nil {
		m.watcher = NewWatcher()
	}

	m.watcher.AddCallback(callback)

	// Start Nacos listener if configured
	if m.nacosClient != nil && m.nacosConfig != nil {
		go m.nacosClient.ListenConfig(m.handleNacosChange)
	}

	return nil
}

// handleNacosChange handles configuration changes from Nacos.
func (m *Manager) handleNacosChange(namespace, group, dataID, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Merge new config
	if err := m.v.MergeConfig(strings.NewReader(content)); err != nil {
		slog.Error("failed to merge nacos config", "error", err)
		return
	}

	// Re-unmarshal
	m.config = &Config{}
	if err := m.v.Unmarshal(m.config); err != nil {
		slog.Error("failed to unmarshal config", "error", err)
		return
	}

	slog.Info("configuration updated from nacos")

	// Trigger callbacks
	if m.watcher != nil {
		m.watcher.TriggerCallbacks()
	}
}

// Close closes the configuration manager and releases resources.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.nacosClient != nil {
		return m.nacosClient.Close()
	}
	return nil
}

// Viper returns the underlying viper instance for advanced usage.
func (m *Manager) Viper() *viper.Viper {
	return m.v
}
