package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		expect func(t *testing.T, m *Manager)
	}{
		{
			name: "default manager",
			opts: nil,
			expect: func(t *testing.T, m *Manager) {
				if m.configFile != "configs/config.yaml" {
					t.Errorf("expected default config file, got %s", m.configFile)
				}
				if m.envPrefix != "PROMETAU" {
					t.Errorf("expected default env prefix, got %s", m.envPrefix)
				}
			},
		},
		{
			name: "custom config file",
			opts: []Option{WithConfigFile("test.yaml")},
			expect: func(t *testing.T, m *Manager) {
				if m.configFile != "test.yaml" {
					t.Errorf("expected test.yaml, got %s", m.configFile)
				}
			},
		},
		{
			name: "custom env prefix",
			opts: []Option{WithEnvPrefix("TESTAPP")},
			expect: func(t *testing.T, m *Manager) {
				if m.envPrefix != "TESTAPP" {
					t.Errorf("expected TESTAPP, got %s", m.envPrefix)
				}
			},
		},
		{
			name: "custom viper instance",
			opts: []Option{WithViper(viper.New())},
			expect: func(t *testing.T, m *Manager) {
				if m.v == nil {
					t.Error("expected viper instance to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.opts...)
			tt.expect(t, m)
		})
	}
}

func TestLoadLocalConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  host: "0.0.0.0"
  port: 9090
database:
  host: "db.example.com"
  port: 3307
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := NewManager(WithConfigFile(configPath))
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	cfg, err := m.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("expected server port 9090, got %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "db.example.com" {
		t.Errorf("expected database host db.example.com, got %s", cfg.Database.Host)
	}
	if cfg.Database.Port != 3307 {
		t.Errorf("expected database port 3307, got %d", cfg.Database.Port)
	}
}

func TestEnvPriority(t *testing.T) {
	// Set environment variable
	os.Setenv("PROMETAU_SERVER_PORT", "9999")
	os.Setenv("PROMETAU_DATABASE_HOST", "env-db.example.com")
	defer func() {
		os.Unsetenv("PROMETAU_SERVER_PORT")
		os.Unsetenv("PROMETAU_DATABASE_HOST")
	}()

	// Create a temporary config file with different values
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  port: 8080
database:
  host: "file-db.example.com"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := NewManager(WithConfigFile(configPath))
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	cfg, err := m.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Environment variables should override config file
	if cfg.Server.Port != 9999 {
		t.Errorf("expected server port 9999 (from env), got %d", cfg.Server.Port)
	}
	if cfg.Database.Host != "env-db.example.com" {
		t.Errorf("expected database host env-db.example.com (from env), got %s", cfg.Database.Host)
	}
}

func TestDefaultValues(t *testing.T) {
	// Create a temporary config file with minimal content
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  port: 8080
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := NewManager(WithConfigFile(configPath))
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	cfg, err := m.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Check default values are applied
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected default server host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Database.Driver != "mysql" {
		t.Errorf("expected default database driver mysql, got %s", cfg.Database.Driver)
	}
	if cfg.Database.Port != 3306 {
		t.Errorf("expected default database port 3306, got %d", cfg.Database.Port)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("expected default redis port 6379, got %d", cfg.Redis.Port)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Log.Level)
	}
}

func TestGetBeforeLoad(t *testing.T) {
	m := NewManager()

	_, err := m.Get()
	if err != ErrNotLoaded {
		t.Errorf("expected ErrNotLoaded, got %v", err)
	}
}

func TestGetKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  port: 8080
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := NewManager(WithConfigFile(configPath))
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	val, err := m.GetKey("server.port")
	if err != nil {
		t.Fatalf("GetKey() failed: %v", err)
	}

	if val != 8080 {
		t.Errorf("expected server.port 8080, got %v", val)
	}
}

func TestBind(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
server:
  host: "192.168.1.1"
  port: 8080
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := NewManager(WithConfigFile(configPath))
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	type PartialConfig struct {
		Server ServerConfig `mapstructure:"server"`
	}

	var partial PartialConfig
	if err := m.Bind(&partial); err != nil {
		t.Fatalf("Bind() failed: %v", err)
	}

	if partial.Server.Host != "192.168.1.1" {
		t.Errorf("expected server host 192.168.1.1, got %s", partial.Server.Host)
	}
	if partial.Server.Port != 8080 {
		t.Errorf("expected server port 8080, got %d", partial.Server.Port)
	}
}

func TestMissingConfigFile(t *testing.T) {
	m := NewManager(WithConfigFile("nonexistent.yaml"))

	// Should not fail - just use defaults
	if err := m.Load(); err != nil {
		t.Errorf("Load() should not fail with missing file: %v", err)
	}

	cfg, err := m.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	// Should have default values
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
}

func TestNacosConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	configContent := `
nacos:
  server_addr: "nacos.example.com:8848"
  namespace: "test-namespace"
  data_id: "test-app.yaml"
  group: "TEST_GROUP"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	m := NewManager(WithConfigFile(configPath))
	if err := m.Load(); err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	cfg, err := m.Get()
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if cfg.Nacos.ServerAddr != "nacos.example.com:8848" {
		t.Errorf("expected nacos server addr, got %s", cfg.Nacos.ServerAddr)
	}
	if cfg.Nacos.Namespace != "test-namespace" {
		t.Errorf("expected nacos namespace, got %s", cfg.Nacos.Namespace)
	}
}
