package api

import (
	"bytes"
	"io"
	"strings"
	"time"

	"log-service/internal/config"
	"log-service/internal/model"
	"log-service/internal/service"
)

// mockStorage 内存存储实现，用于 HTTP 层测试
type mockStorage struct {
	files map[string]map[string][]byte // username -> filename -> content
	times map[string]time.Time
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		files: make(map[string]map[string][]byte),
		times: make(map[string]time.Time),
	}
}

func (m *mockStorage) put(username, filename string, data []byte) {
	if _, ok := m.files[username]; !ok {
		m.files[username] = make(map[string][]byte)
	}
	m.files[username][filename] = data
	m.times[username+"/"+filename] = time.Now()
}

func (m *mockStorage) Save(username, filename string, reader io.Reader) (bool, int64, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return false, 0, err
	}
	_, existed := m.files[username][filename]
	m.put(username, filename, data)
	return existed, int64(len(data)), nil
}

func (m *mockStorage) Get(username, filename string) (io.ReadCloser, int64, error) {
	data, ok := m.files[username][filename]
	if !ok {
		return nil, 0, service.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

func (m *mockStorage) Query(username, keyword string) ([]model.FileInfo, error) {
	var result []model.FileInfo
	for name, data := range m.files[username] {
		if keyword != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(keyword)) {
			continue
		}
		result = append(result, model.FileInfo{
			Username:     username,
			Name:         name,
			Size:         int64(len(data)),
			ModifiedTime: m.times[username+"/"+name].Format(time.RFC3339),
		})
	}
	return result, nil
}

func (m *mockStorage) Delete(username, filename string) error {
	if _, ok := m.files[username][filename]; !ok {
		return service.ErrNotFound
	}
	delete(m.files[username], filename)
	return nil
}

func (m *mockStorage) ListAll() ([]model.FileInfo, error) {
	var result []model.FileInfo
	for user := range m.files {
		files, _ := m.Query(user, "")
		result = append(result, files...)
	}
	return result, nil
}

func (m *mockStorage) ListUsers() ([]string, error) {
	var users []string
	for user := range m.files {
		users = append(users, user)
	}
	return users, nil
}

func testConfig(authEnabled bool) *config.Config {
	enabled := authEnabled
	cfg := &config.Config{
		Upload:    config.UploadConfig{MaxSizeMB: 100},
		RateLimit: config.RateLimitConfig{Limit: 10000, IntervalSeconds: 60},
		Auth: config.AuthConfig{
			Enabled:         &enabled,
			Token:           "test-token",
			SessionTTLHours: 1,
		},
	}
	return cfg
}

func newTestHandler() (*Handler, *mockStorage) {
	storage := newMockStorage()
	return NewHandler(storage, nil, testConfig(false)), storage
}

func newAuthTestHandler() (*Handler, *mockStorage) {
	storage := newMockStorage()
	return NewHandler(storage, nil, testConfig(true)), storage
}