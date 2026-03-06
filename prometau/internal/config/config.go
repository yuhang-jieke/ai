// Package config provides configuration management with Viper and Nacos support.
// Priority: Nacos > Environment Variables > Config File > Default Values
package config

// Config is the top-level configuration structure.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	Nacos    NacosConfig    `mapstructure:"nacos"`
	Storage  StorageConfig  `mapstructure:"storage"`
}

// ServerConfig contains HTTP server configuration.
type ServerConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"read_timeout"`  // in seconds
	WriteTimeout int    `mapstructure:"write_timeout"` // in seconds
}

// DatabaseConfig contains MySQL database configuration.
type DatabaseConfig struct {
	Driver          string `mapstructure:"driver"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // in seconds
}

// RedisConfig contains Redis configuration.
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LogConfig contains logging configuration.
type LogConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // json or text
	Output     string `mapstructure:"output"`      // stdout or file
	FileName   string `mapstructure:"file_name"`   // log file path
	MaxSize    int    `mapstructure:"max_size"`    // max size in MB
	MaxBackups int    `mapstructure:"max_backups"` // max number of old log files
	MaxAge     int    `mapstructure:"max_age"`     // max days to retain
}

// NacosConfig contains Nacos configuration center settings.
type NacosConfig struct {
	ServerAddr string `mapstructure:"server_addr"` // Nacos server address (host:port)
	Namespace  string `mapstructure:"namespace"`   // Namespace ID
	DataID     string `mapstructure:"data_id"`     // Configuration Data ID
	Group      string `mapstructure:"group"`       // Configuration Group
	Username   string `mapstructure:"username"`    // Authentication username
	Password   string `mapstructure:"password"`    // Authentication password
}

// StorageConfig contains object storage configuration.
type StorageConfig struct {
	Type   string       `mapstructure:"type"`   // Storage type: minio, oss, rustfs
	MinIO  MinIOConfig  `mapstructure:"minio"`  // MinIO configuration
	OSS    OSSConfig    `mapstructure:"oss"`    // Aliyun OSS configuration
	RustFS RustFSConfig `mapstructure:"rustfs"` // RustFS configuration
}

// MinIOConfig contains MinIO configuration.
type MinIOConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	UseSSL          bool   `mapstructure:"use_ssl"`
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
}

// OSSConfig contains Aliyun OSS configuration.
type OSSConfig struct {
	Endpoint        string `mapstructure:"endpoint"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
}

// RustFSConfig contains RustFS configuration.
type RustFSConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Token    string `mapstructure:"token"`
	Bucket   string `mapstructure:"bucket"`
}

// DefaultConfig returns configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Database: DatabaseConfig{
			Driver:          "mysql",
			Host:            "115.190.57.118",
			Port:            3306,
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600,
		},
		Redis: RedisConfig{
			Host: "115.190.57.118",
			Port: 6379,
			DB:   0,
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
		},
		Nacos: NacosConfig{
			ServerAddr: "115.190.57.118:8848",
			Namespace:  "public",
			DataID:     "demo-config",
			Group:      "DEFAULT_GROUP",
		},
		Storage: StorageConfig{
			Type: "minio",
			MinIO: MinIOConfig{
				Endpoint:        "115.190.57.118:9000",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UseSSL:          false,
				Bucket:          "yuhang",
				Region:          "",
			},
			OSS: OSSConfig{
				Endpoint:        "oss-cn-shanghai.aliyuncs.com",
				AccessKeyID:     "LTAI5tSYLWEhPs3Vjh4kTvGR",
				AccessKeySecret: "R8R1Clr66RQiQZtAuPPiCV3steCtUG",
				Bucket:          "byh666",
				Region:          "cn-shanghai",
			},
		},
	}
}
