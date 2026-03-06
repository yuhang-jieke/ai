package config

import (
	"errors"
	"testing"
)

func TestMockNacosClient_GetConfig(t *testing.T) {
	tests := []struct {
		name    string
		mock    *MockNacosClient
		want    string
		wantErr bool
	}{
		{
			name: "returns config content",
			mock: &MockNacosClient{ConfigContent: "test: value"},
			want: "test: value",
		},
		{
			name:    "returns error",
			mock:    &MockNacosClient{ConfigError: errors.New("connection failed")},
			wantErr: true,
		},
		{
			name: "empty content",
			mock: &MockNacosClient{ConfigContent: ""},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.mock.GetConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockNacosClient_Close(t *testing.T) {
	mock := &MockNacosClient{}
	if mock.Closed {
		t.Error("mock should not be closed initially")
	}

	if err := mock.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !mock.Closed {
		t.Error("mock should be closed after Close()")
	}
}

func TestMockNacosClient_ListenConfig(t *testing.T) {
	mock := &MockNacosClient{}

	// ListenConfig should not fail in mock
	err := mock.ListenConfig(func(namespace, group, dataID, content string) {})
	if err != nil {
		t.Errorf("ListenConfig() error = %v", err)
	}
}

func TestMockNacosClient_CancelListenConfig(t *testing.T) {
	mock := &MockNacosClient{}

	// CancelListenConfig should not fail in mock
	err := mock.CancelListenConfig()
	if err != nil {
		t.Errorf("CancelListenConfig() error = %v", err)
	}
}

func TestNacosConfigStruct(t *testing.T) {
	cfg := NacosConfig{
		ServerAddr: "localhost:8848",
		Namespace:  "test-ns",
		DataID:     "app.yaml",
		Group:      "DEFAULT_GROUP",
		Username:   "admin",
		Password:   "secret",
	}

	if cfg.ServerAddr != "localhost:8848" {
		t.Errorf("ServerAddr = %v, want localhost:8848", cfg.ServerAddr)
	}
	if cfg.Namespace != "test-ns" {
		t.Errorf("Namespace = %v, want test-ns", cfg.Namespace)
	}
	if cfg.DataID != "app.yaml" {
		t.Errorf("DataID = %v, want app.yaml", cfg.DataID)
	}
	if cfg.Group != "DEFAULT_GROUP" {
		t.Errorf("Group = %v, want DEFAULT_GROUP", cfg.Group)
	}
	if cfg.Username != "admin" {
		t.Errorf("Username = %v, want admin", cfg.Username)
	}
	if cfg.Password != "secret" {
		t.Errorf("Password = %v, want secret", cfg.Password)
	}
}

func TestNacosClientInterface(t *testing.T) {
	// Ensure MockNacosClient implements NacosClientInterface
	var _ NacosClientInterface = (*MockNacosClient)(nil)
}
