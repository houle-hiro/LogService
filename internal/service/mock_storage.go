package service

import (
	"bytes"
	"io"
	"time"

	"log-service/internal/model"
)

// mockStorage 内存存储实现，用于服务层测试
type mockStorage struct {
	files map[string]map[string][]byte // username -> filename -> content
	times map[string]time.Time          // username/filename -> 修改时间
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		files: make(map[string]map[string][]byte),
		times: make(map[string]time.Time),
	}
}

func (m *mockStorage) maybeInit(username string) {
	if _, ok := m.files[username]; !ok {
		m.files[username] = make(map[string][]byte)
	}
}

// put 直接写入文件并设置修改时间
func (m *mockStorage) put(username, filename string, data []byte, modTime time.Time) {
	m.maybeInit(username)
	m.files[username][filename] = data
	m.times[username+"/"+filename] = modTime
}

func (m *mockStorage) Save(username, filename string, reader io.Reader) (bool, int64, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return false, 0, err
	}
	_, existed := m.files[username][filename]
	m.put(username, filename, data, time.Now())
	return existed, int64(len(data)), nil
}

func (m *mockStorage) Get(username, filename string) (io.ReadCloser, int64, error) {
	data, ok := m.files[username][filename]
	if !ok {
		return nil, 0, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

func (m *mockStorage) Query(username, keyword string) ([]model.FileInfo, error) {
	var result []model.FileInfo
	for name, data := range m.files[username] {
		if keyword != "" && !containsFold(name, keyword) {
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
		return ErrNotFound
	}
	delete(m.files[username], filename)
	delete(m.times, username+"/"+filename)
	return nil
}

func (m *mockStorage) ListAll() ([]model.FileInfo, error) {
	var result []model.FileInfo
	for username := range m.files {
		files, _ := m.Query(username, "")
		result = append(result, files...)
	}
	return result, nil
}

func (m *mockStorage) ListUsers() ([]string, error) {
	var users []string
	for username := range m.files {
		users = append(users, username)
	}
	return users, nil
}

func containsFold(s, sub string) bool {
	if sub == "" {
		return true
	}
	s, sub = lower(s), lower(sub)
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func lower(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}