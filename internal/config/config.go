package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	Upload    UploadConfig    `yaml:"upload"`
	RateLimit RateLimitConfig `yaml:"ratelimit"`
	Auth      AuthConfig      `yaml:"auth"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port         int `yaml:"port"`
	ReadTimeout  int `yaml:"read_timeout"`
	WriteTimeout int `yaml:"write_timeout"`
	IdleTimeout  int `yaml:"idle_timeout"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Path string `yaml:"path"`
	// MaxAgeSeconds 保留秒数，0 表示不清理，未配置默认 30 天
	MaxAgeSeconds *int `yaml:"max_age_seconds"`
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxSizeMB int `yaml:"max_size_mb"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Limit           int `yaml:"limit"`
	IntervalSeconds int `yaml:"interval_seconds"`
}

// AuthConfig 鉴权配置
type AuthConfig struct {
	Enabled *bool  `yaml:"enabled"`
	Token   string `yaml:"token"`
	// SessionSecret 会话签名密钥，为空时由 token 派生
	SessionSecret   string `yaml:"session_secret"`
	SessionTTLHours int    `yaml:"session_ttl_hours"`
}

// Load 从指定路径加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	cfg.setDefaults()

	if err := cfg.validateBasic(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	return &cfg, nil
}

// LoadDefault 加载默认路径配置，支持环境变量覆盖
func LoadDefault() (*Config, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "configs/config.yaml"
	}

	cfg, err := Load(path)
	if err != nil {
		return nil, err
	}

	// 密钥优先从环境变量读，避免写进仓库
	if token := os.Getenv("AUTH_TOKEN"); token != "" {
		cfg.Auth.Token = token
	}
	if secret := os.Getenv("SESSION_SECRET"); secret != "" {
		cfg.Auth.SessionSecret = secret
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}
	return cfg, nil
}

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
	if c.Upload.MaxSizeMB == 0 {
		c.Upload.MaxSizeMB = 100
	}
	if c.RateLimit.Limit == 0 {
		c.RateLimit.Limit = 100
	}
	if c.RateLimit.IntervalSeconds == 0 {
		c.RateLimit.IntervalSeconds = 60
	}
	if c.Auth.Enabled == nil {
		enabled := true
		c.Auth.Enabled = &enabled
	}
	if c.Auth.SessionTTLHours == 0 {
		c.Auth.SessionTTLHours = 12
	}
}

func (c *Config) validateBasic() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("端口必须在 1-65535 之间")
	}
	if c.Upload.MaxSizeMB <= 0 {
		return fmt.Errorf("上传大小限制必须大于 0")
	}
	if c.RateLimit.Limit <= 0 {
		return fmt.Errorf("限流限制必须大于 0")
	}
	if c.Storage.MaxAgeSeconds != nil && *c.Storage.MaxAgeSeconds < 0 {
		return fmt.Errorf("保留时间不能为负数，0 表示不清理")
	}
	return nil
}

// validate 完整校验，含鉴权口令（环境变量注入后再执行）
func (c *Config) validate() error {
	if err := c.validateBasic(); err != nil {
		return err
	}
	if c.AuthEnabled() && c.Auth.Token == "" {
		return fmt.Errorf("已启用鉴权但未配置 token，请设置 AUTH_TOKEN 环境变量或 configs/config.yaml 中的 auth.token")
	}
	return nil
}

// AuthEnabled 是否启用鉴权
func (c *Config) AuthEnabled() bool {
	if c.Auth.Enabled == nil {
		return true
	}
	return *c.Auth.Enabled
}

// GetMaxAge 文件保留时长，0 表示不清理
func (c *Config) GetMaxAge() time.Duration {
	if c.Storage.MaxAgeSeconds == nil {
		return 30 * 24 * time.Hour
	}
	if *c.Storage.MaxAgeSeconds <= 0 {
		return 0
	}
	return time.Duration(*c.Storage.MaxAgeSeconds) * time.Second
}

// GetMaxAgeSeconds 保留秒数，供清理器使用
func (c *Config) GetMaxAgeSeconds() int {
	return int(c.GetMaxAge().Seconds())
}

// GetReadTimeout 读取超时
func (c *Config) GetReadTimeout() time.Duration {
	return time.Duration(c.Server.ReadTimeout) * time.Second
}

// GetWriteTimeout 写入超时
func (c *Config) GetWriteTimeout() time.Duration {
	return time.Duration(c.Server.WriteTimeout) * time.Second
}

// GetIdleTimeout 空闲超时
func (c *Config) GetIdleTimeout() time.Duration {
	return time.Duration(c.Server.IdleTimeout) * time.Second
}

// GetMaxUploadSize 最大上传字节数
func (c *Config) GetMaxUploadSize() int64 {
	return int64(c.Upload.MaxSizeMB) * 1024 * 1024
}

// GetRateLimitInterval 限流时间窗口
func (c *Config) GetRateLimitInterval() time.Duration {
	return time.Duration(c.RateLimit.IntervalSeconds) * time.Second
}

// GetSessionTTL 会话有效期
func (c *Config) GetSessionTTL() time.Duration {
	return time.Duration(c.Auth.SessionTTLHours) * time.Hour
}

// GetSessionSecret 会话签名密钥，优先配置，其次由 token 派生
func (c *Config) GetSessionSecret() string {
	if c.Auth.SessionSecret != "" {
		return c.Auth.SessionSecret
	}
	return "session::" + c.Auth.Token
}