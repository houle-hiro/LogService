package service

import (
	"io"
	"log-service/internal/model"
	"testing"
	"time"
)

func TestCleanupClean(t *testing.T) {
	storage := newMockStorage()

	// 添加一个过期文件
	storage.files["old_file.zip"] = []byte("old content")

	// 添加一个新文件
	storage.files["new_file.zip"] = []byte("new content")

	// 创建清理器，设置过期时间为负数（这样所有文件都会过期）
	cleanup := &Cleanup{
		storage:  storage,
		maxAge:   -1 * time.Hour, // 负数表示所有文件都过期
		interval: 1 * time.Hour,
		stopChan: make(chan struct{}),
	}

	// 执行清理
	cleanup.clean()

	// 验证文件已被删除
	if len(storage.files) != 0 {
		t.Errorf("期望所有文件被删除，剩余 %d", len(storage.files))
	}
}

func TestCleanupCleanNotExpired(t *testing.T) {
	storage := newMockStorage()

	// 添加文件
	storage.files["file1.zip"] = []byte("content")

	// 创建清理器，设置很长的过期时间
	cleanup := &Cleanup{
		storage:  storage,
		maxAge:   24 * time.Hour,
		interval: 1 * time.Hour,
		stopChan: make(chan struct{}),
	}

	// 执行清理
	cleanup.clean()

	// 验证文件未被删除
	if len(storage.files) != 1 {
		t.Error("未过期文件不应被删除")
	}
}

func TestCleanupStartStop(t *testing.T) {
	storage := newMockStorage()
	cleanup := NewCleanup(storage, 3600) // 1 小时

	// 启动清理
	cleanup.Start()

	// 给 goroutine 一点启动时间
	time.Sleep(10 * time.Millisecond)

	// 停止清理
	cleanup.Stop()

	// 再次停止应该不会 panic
	cleanup.Stop()
}

func TestCleanupListAllError(t *testing.T) {
	// 使用一个会返回错误的存储
	errorStorage := &errorMockStorage{}
	cleanup := &Cleanup{
		storage:  errorStorage,
		maxAge:   24 * time.Hour,
		interval: 1 * time.Hour,
		stopChan: make(chan struct{}),
	}

	// 执行清理，不应 panic
	cleanup.clean()
}

// errorMockStorage 是一个总是返回错误的存储实现
type errorMockStorage struct{}

func (m *errorMockStorage) Save(filename string, reader io.Reader) error {
	return io.EOF
}

func (m *errorMockStorage) Get(filename string) (io.ReadCloser, int64, error) {
	return nil, 0, io.EOF
}

func (m *errorMockStorage) Query(prefix string) ([]model.FileInfo, error) {
	return nil, io.EOF
}

func (m *errorMockStorage) Delete(filename string) error {
	return io.EOF
}

func (m *errorMockStorage) ListAll() ([]model.FileInfo, error) {
	return nil, io.EOF
}
