package config

import (
	"fmt"
	"net"
	"strconv"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// NacosClientInterface defines the interface for Nacos client operations.
// This allows for easier testing and mocking.
type NacosClientInterface interface {
	GetConfig() (string, error)
	ListenConfig(callback func(namespace, group, dataID, content string)) error
	CancelListenConfig() error
	Close() error
}

// NacosClient wraps the Nacos SDK client for configuration management.
type NacosClient struct {
	client   config_client.IConfigClient
	config   NacosConfig
	listener func(namespace, group, dataID, content string)
}

// NewNacosClient creates a new Nacos client.
func NewNacosClient(cfg NacosConfig) (*NacosClient, error) {
	// Parse server address
	host, port := parseServerAddr(cfg.ServerAddr)

	// Create server config
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr: host,
			Port:   port,
		},
	}

	// Create client config
	clientConfig := constant.ClientConfig{
		NamespaceId: cfg.Namespace,
	}

	// Add authentication if provided
	if cfg.Username != "" && cfg.Password != "" {
		clientConfig.Username = cfg.Username
		clientConfig.Password = cfg.Password
	}

	// Create config client
	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create nacos client: %w", err)
	}

	return &NacosClient{
		client: client,
		config: cfg,
	}, nil
}

// parseServerAddr parses server address in format "host" or "host:port".
func parseServerAddr(addr string) (string, uint64) {
	// Default port
	const defaultPort = 8848

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// No port specified, use default
		return addr, defaultPort
	}

	portNum, err := strconv.ParseUint(port, 10, 64)
	if err != nil {
		return host, defaultPort
	}

	return host, portNum
}

// GetConfig retrieves configuration from Nacos.
func (c *NacosClient) GetConfig() (string, error) {
	content, err := c.client.GetConfig(vo.ConfigParam{
		DataId: c.config.DataID,
		Group:  c.config.Group,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get config from nacos: %w", err)
	}
	return content, nil
}

// ListenConfig registers a listener for configuration changes.
func (c *NacosClient) ListenConfig(callback func(namespace, group, dataID, content string)) error {
	c.listener = callback

	err := c.client.ListenConfig(vo.ConfigParam{
		DataId: c.config.DataID,
		Group:  c.config.Group,
		OnChange: func(namespace, group, dataID, content string) {
			if c.listener != nil {
				c.listener(namespace, group, dataID, content)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("failed to listen config: %w", err)
	}

	return nil
}

// CancelListenConfig stops listening for configuration changes.
func (c *NacosClient) CancelListenConfig() error {
	return c.client.CancelListenConfig(vo.ConfigParam{
		DataId: c.config.DataID,
		Group:  c.config.Group,
	})
}

// Close closes the Nacos client and releases resources.
func (c *NacosClient) Close() error {
	if c.client != nil {
		return c.CancelListenConfig()
	}
	return nil
}

// MockNacosClient is a mock implementation for testing.
type MockNacosClient struct {
	ConfigContent string
	ConfigError   error
	Closed        bool
}

// GetConfig returns the mock config content.
func (m *MockNacosClient) GetConfig() (string, error) {
	if m.ConfigError != nil {
		return "", m.ConfigError
	}
	return m.ConfigContent, nil
}

// ListenConfig does nothing in mock.
func (m *MockNacosClient) ListenConfig(callback func(namespace, group, dataID, content string)) error {
	return nil
}

// CancelListenConfig does nothing in mock.
func (m *MockNacosClient) CancelListenConfig() error {
	return nil
}

// Close marks the mock as closed.
func (m *MockNacosClient) Close() error {
	m.Closed = true
	return nil
}

// Ensure NacosClient implements NacosClientInterface
var _ NacosClientInterface = (*NacosClient)(nil)

// Ensure MockNacosClient implements NacosClientInterface
var _ NacosClientInterface = (*MockNacosClient)(nil)
