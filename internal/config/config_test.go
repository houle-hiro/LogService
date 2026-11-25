package config

import (
	"os"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

func TestLoad(t *testing.T) {
	path := writeTempConfig(t, `
server:
  port: 9090
  read_timeout: 60
  write_timeout: 60
  idle_timeout: 300
storage:
  path: "./test_data"
  max_age_seconds: 604800
upload:
  max_size_mb: 50
ratelimit:
  limit: 200
  interval_seconds: 30
auth:
  enabled: false
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("端口期望 9090，实际 %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 60 {
		t.Errorf("读取超时期望 60，实际 %d", cfg.Server.ReadTimeout)
	}
	if cfg.Storage.Path != "./test_data" {
		t.Errorf("存储路径期望 ./test_data，实际 %s", cfg.Storage.Path)
	}
	if cfg.GetMaxAge() != 604800*time.Second {
		t.Errorf("保留时间期望 604800s，实际 %s", cfg.GetMaxAge())
	}
	if cfg.Upload.MaxSizeMB != 50 {
		t.Errorf("上传大小期望 50，实际 %d", cfg.Upload.MaxSizeMB)
	}
	if cfg.RateLimit.Limit != 200 {
		t.Errorf("限流限制期望 200，实际 %d", cfg.RateLimit.Limit)
	}
	if cfg.AuthEnabled() {
		t.Error("应关闭鉴权")
	}
}

func TestLoadDefaultWithEnvToken(t *testing.T) {
	path := writeTempConfig(t, `
server:
  port: 8080
storage:
  path: "./data"
`)

	t.Setenv("CONFIG_PATH", path)
	t.Setenv("AUTH_TOKEN", "env-token")

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("加载默认配置失败: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("端口期望 8080，实际 %d", cfg.Server.Port)
	}
	if cfg.Auth.Token != "env-token" {
		t.Errorf("token 应来自环境变量，实际 %q", cfg.Auth.Token)
	}
	if !cfg.AuthEnabled() {
		t.Error("默认应启用鉴权")
	}
}

func TestLoadMissingToken(t *testing.T) {
	path := writeTempConfig(t, `
server:
  port: 8080
storage:
  path: "./data"
auth:
  enabled: true
`)
	t.Setenv("CONFIG_PATH", path)
	t.Setenv("AUTH_TOKEN", "")
	if _, err := LoadDefault(); err == nil {
		t.Error("启用鉴权但无 token 时应报错")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	if _, err := Load("/nonexistent/config.yaml"); err == nil {
		t.Error("期望加载不存在的文件时返回错误")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := writeTempConfig(t, "invalid: yaml: content: [")
	if _, err := Load(path); err == nil {
		t.Error("期望解析无效 YAML 时返回错误")
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	if cfg.Server.Port != 8080 {
		t.Errorf("默认端口期望 8080，实际 %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 3600 {
		t.Errorf("默认读取超时期望 3600，实际 %d", cfg.Server.ReadTimeout)
	}
	if cfg.Storage.Path != "./data" {
		t.Errorf("默认存储路径期望 ./data，实际 %s", cfg.Storage.Path)
	}
	if cfg.GetMaxAge() != 30*24*time.Hour {
		t.Errorf("默认保留时间期望 30 天，实际 %s", cfg.GetMaxAge())
	}
	if cfg.Upload.MaxSizeMB != 100 {
		t.Errorf("默认上传大小期望 100，实际 %d", cfg.Upload.MaxSizeMB)
	}
	if cfg.RateLimit.Limit != 100 {
		t.Errorf("默认限流期望 100，实际 %d", cfg.RateLimit.Limit)
	}
	if !cfg.AuthEnabled() {
		t.Error("默认应启用鉴权")
	}
	if cfg.Auth.SessionTTLHours != 12 {
		t.Errorf("默认会话有效期期望 12 小时，实际 %d", cfg.Auth.SessionTTLHours)
	}
}

func TestMaxAge(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want time.Duration
	}{
		{"未配置默认 30 天", Config{}, 30 * 24 * time.Hour},
		{"0 表示不清理", Config{Storage: StorageConfig{MaxAgeSeconds: intPtr(0)}}, 0},
		{"自定义", Config{Storage: StorageConfig{MaxAgeSeconds: intPtr(3600)}}, time.Hour},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.cfg.GetMaxAge(); got != c.want {
				t.Errorf("GetMaxAge() = %s, want %s", got, c.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	enabled := boolPtr(true)
	disabled := boolPtr(false)
	base := func() Config {
		return Config{
			Server:    ServerConfig{Port: 8080},
			Upload:    UploadConfig{MaxSizeMB: 100},
			RateLimit: RateLimitConfig{Limit: 100},
			Auth:      AuthConfig{Enabled: enabled, Token: "t"},
		}
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"有效配置", func(c *Config) {}, false},
		{"端口为 0", func(c *Config) { c.Server.Port = 0 }, true},
		{"端口过大", func(c *Config) { c.Server.Port = 99999 }, true},
		{"负数保留时间", func(c *Config) { c.Storage.MaxAgeSeconds = intPtr(-1) }, true},
		{"保留时间为 0 合法", func(c *Config) { c.Storage.MaxAgeSeconds = intPtr(0) }, false},
		{"上传大小为 0", func(c *Config) { c.Upload.MaxSizeMB = 0 }, true},
		{"限流为 0", func(c *Config) { c.RateLimit.Limit = 0 }, true},
		{"启用鉴权但无 token", func(c *Config) { c.Auth.Token = "" }, true},
		{"关闭鉴权无 token", func(c *Config) { c.Auth.Enabled = disabled }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base()
			tt.mutate(&cfg)
			err := cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{ReadTimeout: 30, WriteTimeout: 60, IdleTimeout: 120},
		RateLimit: RateLimitConfig{IntervalSeconds: 60},
		Auth: AuthConfig{SessionTTLHours: 12},
	}

	if cfg.GetReadTimeout() != 30*time.Second {
		t.Error("GetReadTimeout 返回值不正确")
	}
	if cfg.GetWriteTimeout() != 60*time.Second {
		t.Error("GetWriteTimeout 返回值不正确")
	}
	if cfg.GetIdleTimeout() != 120*time.Second {
		t.Error("GetIdleTimeout 返回值不正确")
	}
	if cfg.GetRateLimitInterval() != 60*time.Second {
		t.Error("GetRateLimitInterval 返回值不正确")
	}
	if cfg.GetSessionTTL() != 12*time.Hour {
		t.Error("GetSessionTTL 返回值不正确")
	}
}

func TestGetMaxUploadSize(t *testing.T) {
	cfg := &Config{Upload: UploadConfig{MaxSizeMB: 100}}
	expected := int64(100 * 1024 * 1024)
	if cfg.GetMaxUploadSize() != expected {
		t.Errorf("GetMaxUploadSize 期望 %d，实际 %d", expected, cfg.GetMaxUploadSize())
	}
}

func TestGetSessionSecret(t *testing.T) {
	cfg := &Config{Auth: AuthConfig{Token: "abc"}}
	if cfg.GetSessionSecret() == "" {
		t.Error("未配置 session_secret 时应由 token 派生")
	}
	cfg.Auth.SessionSecret = "fixed"
	if cfg.GetSessionSecret() != "fixed" {
		t.Error("应优先使用配置的 session_secret")
	}
}