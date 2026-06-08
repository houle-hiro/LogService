package service

import (
	"bytes"
	"io"
	"log-service/internal/model"
	"time"
)

// mockStorage 是一个用于测试的内存存储实现
type mockStorage struct {
	files map[string][]byte
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		files: make(map[string][]byte),
	}
}

func (m *mockStorage) Save(filename string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	m.files[filename] = data
	return nil
}

func (m *mockStorage) Get(filename string) (io.ReadCloser, int64, error) {
	data, ok := m.files[filename]
	if !ok {
		return nil, 0, io.EOF
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}

func (m *mockStorage) Query(prefix string) ([]model.FileInfo, error) {
	var result []model.FileInfo
	for name, data := range m.files {
		if len(prefix) == 0 || len(name) >= len(prefix) && name[:len(prefix)] == prefix {
			result = append(result, model.FileInfo{
				Name: name,
				Size: int64(len(data)),
			})
		}
	}
	return result, nil
}

func (m *mockStorage) Delete(filename string) error {
	delete(m.files, filename)
	return nil
}

func (m *mockStorage) ListAll() ([]model.FileInfo, error) {
	var result []model.FileInfo
	for name, data := range m.files {
		result = append(result, model.FileInfo{
			Name:         name,
			Size:         int64(len(data)),
			ModifiedTime: time.Now().Add(-1 * time.Hour).Format(time.RFC3339), // 默认 1 小时前
		})
	}
	return result, nil
}
