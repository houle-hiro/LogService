package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// 创建临时配置文件
	content := `
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
`
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	// 加载配置
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置值
	if cfg.Server.Port != 9090 {
		t.Errorf("端口期望 9090，实际 %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 60 {
		t.Errorf("读取超时期望 60，实际 %d", cfg.Server.ReadTimeout)
	}
	if cfg.Storage.Path != "./test_data" {
		t.Errorf("存储路径期望 ./test_data，实际 %s", cfg.Storage.Path)
	}
	if cfg.Storage.MaxAgeSeconds != 604800 {
		t.Errorf("保留时间期望 604800，实际 %d", cfg.Storage.MaxAgeSeconds)
	}
	if cfg.Upload.MaxSizeMB != 50 {
		t.Errorf("上传大小期望 50，实际 %d", cfg.Upload.MaxSizeMB)
	}
	if cfg.RateLimit.Limit != 200 {
		t.Errorf("限流限制期望 200，实际 %d", cfg.RateLimit.Limit)
	}
}

func TestLoadDefault(t *testing.T) {
	// 创建临时配置文件到默认路径
	content := `
server:
  port: 8080
storage:
  path: "./data"
`
	tmpDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := tmpDir + "/config.yaml"
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}

	// 设置环境变量
	os.Setenv("CONFIG_PATH", configPath)
	defer os.Unsetenv("CONFIG_PATH")

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("加载默认配置失败: %v", err)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("端口期望 8080，实际 %d", cfg.Server.Port)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Error("期望加载不存在的文件时返回错误")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("invalid: yaml: content: ["); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Error("期望解析无效 YAML 时返回错误")
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	if cfg.Server.Port != 8080 {
		t.Errorf("默认端口期望 8080，实际 %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 30 {
		t.Errorf("默认读取超时期望 30，实际 %d", cfg.Server.ReadTimeout)
	}
	if cfg.Storage.Path != "./data" {
		t.Errorf("默认存储路径期望 ./data，实际 %s", cfg.Storage.Path)
	}
	if cfg.Storage.MaxAgeSeconds != 2592000 {
		t.Errorf("默认保留时间期望 2592000，实际 %d", cfg.Storage.MaxAgeSeconds)
	}
	if cfg.Upload.MaxSizeMB != 100 {
		t.Errorf("默认上传大小期望 100，实际 %d", cfg.Upload.MaxSizeMB)
	}
	if cfg.RateLimit.Limit != 100 {
		t.Errorf("默认限流期望 100，实际 %d", cfg.RateLimit.Limit)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "有效配置",
			cfg: Config{
				Server:    ServerConfig{Port: 8080},
				Storage:   StorageConfig{MaxAgeSeconds: 2592000},
				Upload:    UploadConfig{MaxSizeMB: 100},
				RateLimit: RateLimitConfig{Limit: 100},
			},
			wantErr: false,
		},
		{
			name: "无效端口",
			cfg: Config{
				Server:    ServerConfig{Port: 0},
				Storage:   StorageConfig{MaxAgeSeconds: 2592000},
				Upload:    UploadConfig{MaxSizeMB: 100},
				RateLimit: RateLimitConfig{Limit: 100},
			},
			wantErr: true,
		},
		{
			name: "端口过大",
			cfg: Config{
				Server:    ServerConfig{Port: 99999},
				Storage:   StorageConfig{MaxAgeSeconds: 2592000},
				Upload:    UploadConfig{MaxSizeMB: 100},
				RateLimit: RateLimitConfig{Limit: 100},
			},
			wantErr: true,
		},
		{
			name: "无效保留时间",
			cfg: Config{
				Server:    ServerConfig{Port: 8080},
				Storage:   StorageConfig{MaxAgeSeconds: 0},
				Upload:    UploadConfig{MaxSizeMB: 100},
				RateLimit: RateLimitConfig{Limit: 100},
			},
			wantErr: true,
		},
		{
			name: "无效上传大小",
			cfg: Config{
				Server:    ServerConfig{Port: 8080},
				Storage:   StorageConfig{MaxAgeSeconds: 2592000},
				Upload:    UploadConfig{MaxSizeMB: 0},
				RateLimit: RateLimitConfig{Limit: 100},
			},
			wantErr: true,
		},
		{
			name: "无效限流",
			cfg: Config{
				Server:    ServerConfig{Port: 8080},
				Storage:   StorageConfig{MaxAgeSeconds: 2592000},
				Upload:    UploadConfig{MaxSizeMB: 100},
				RateLimit: RateLimitConfig{Limit: 0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDuration(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			ReadTimeout:  30,
			WriteTimeout: 60,
			IdleTimeout:  120,
		},
		RateLimit: RateLimitConfig{
			IntervalSeconds: 60,
		},
	}

	if cfg.GetReadTimeout() != 30*1e9 {
		t.Error("GetReadTimeout 返回值不正确")
	}
	if cfg.GetWriteTimeout() != 60*1e9 {
		t.Error("GetWriteTimeout 返回值不正确")
	}
	if cfg.GetIdleTimeout() != 120*1e9 {
		t.Error("GetIdleTimeout 返回值不正确")
	}
	if cfg.GetRateLimitInterval() != 60*1e9 {
		t.Error("GetRateLimitInterval 返回值不正确")
	}
}

func TestGetMaxUploadSize(t *testing.T) {
	cfg := &Config{
		Upload: UploadConfig{MaxSizeMB: 100},
	}

	expected := int64(100 * 1024 * 1024)
	if cfg.GetMaxUploadSize() != expected {
		t.Errorf("GetMaxUploadSize 期望 %d，实际 %d", expected, cfg.GetMaxUploadSize())
	}
}
