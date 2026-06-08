package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构体
type Config struct {
	Server    ServerConfig    `yaml:"server"`    // 服务器配置
	Storage   StorageConfig   `yaml:"storage"`   // 存储配置
	Upload    UploadConfig    `yaml:"upload"`    // 上传配置
	RateLimit RateLimitConfig `yaml:"ratelimit"` // 限流配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int `yaml:"port"`          // 监听端口
	ReadTimeout  int `yaml:"read_timeout"`  // 读取超时（秒）
	WriteTimeout int `yaml:"write_timeout"` // 写入超时（秒）
	IdleTimeout  int `yaml:"idle_timeout"`  // 空闲超时（秒）
}

// StorageConfig 存储配置
type StorageConfig struct {
	Path          string `yaml:"path"`            // 存储路径
	MaxAgeSeconds int    `yaml:"max_age_seconds"` // 最大保留秒数
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxSizeMB int `yaml:"max_size_mb"` // 最大文件大小（MB）
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Limit            int `yaml:"limit"`             // 请求数限制
	IntervalSeconds  int `yaml:"interval_seconds"`  // 时间窗口（秒）
}

// Load 从指定路径加载配置文件
func Load(path string) (*Config, error) {
	// 读取配置文件内容
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	cfg.setDefaults()

	// 验证配置合法性
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// LoadDefault 加载默认路径的配置文件
func LoadDefault() (*Config, error) {
	// 优先从环境变量获取配置文件路径
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "configs/config.yaml"
	}
	return Load(path)
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 120
	}
	if c.Storage.Path == "" {
		c.Storage.Path = "./data"
	}
	if c.Storage.MaxAgeSeconds == 0 {
		c.Storage.MaxAgeSeconds = 2592000 // 默认 30 天
	}
	if c.Upload.MaxSizeMB == 0 {
		c.Upload.MaxSizeMB = 100
	}
	if c.RateLimit.Limit == 0 {
		c.RateLimit.Limit = 100
	}
	if c.RateLimit.IntervalSeconds == 0 {
		c.RateLimit.IntervalSeconds = 60
	}
}

// validate 验证配置合法性
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("端口必须在 1-65535 之间")
	}
	if c.Storage.MaxAgeSeconds <= 0 {
		return fmt.Errorf("保留时间必须大于 0")
	}
	if c.Upload.MaxSizeMB <= 0 {
		return fmt.Errorf("上传大小限制必须大于 0")
	}
	if c.RateLimit.Limit <= 0 {
		return fmt.Errorf("限流限制必须大于 0")
	}
	return nil
}

// GetReadTimeout 获取读取超时时间
func (c *Config) GetReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeout) * time.Second
}

// GetWriteTimeout 获取写入超时时间
func (c *Config) GetWriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeout) * time.Second
}

// GetIdleTimeout 获取空闲超时时间
func (c *Config) GetIdleTimeout() time.Duration {
	return time.Duration(c.Server.IdleTimeout) * time.Second
}

// GetMaxUploadSize 获取最大上传大小（字节）
func (c *Config) GetMaxUploadSize() int64 {
	return int64(c.Upload.MaxSizeMB) * 1024 * 1024
}

// GetRateLimitInterval 获取限流时间窗口
func (c *Config) GetRateLimitInterval() time.Duration {
	return time.Duration(c.RateLimit.IntervalSeconds) * time.Second
}
