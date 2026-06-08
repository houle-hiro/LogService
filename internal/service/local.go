package service

import (
	"fmt"
	"io"
	"log-service/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type LocalStorage struct {
	basePath string
	mu       sync.RWMutex
}

func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	return &LocalStorage{basePath: basePath}, nil
}

// extractName 从文件名中提取名称部分（第一个下划线之前）
func extractName(filename string) string {
	idx := strings.Index(filename, "_")
	if idx <= 0 {
		return ""
	}
	return filename[:idx]
}

func (s *LocalStorage) Save(filename string, reader io.Reader) error {
	name := extractName(filename)
	if name == "" {
		return fmt.Errorf("无法从文件名中提取名称: %s", filename)
	}

	// 创建子目录
	subDir := filepath.Join(s.basePath, name)
	if err := os.MkdirAll(subDir, 0755); err != nil {
		return fmt.Errorf("创建子目录失败: %w", err)
	}

	filePath := filepath.Join(subDir, filename)

	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	if err != nil {
		os.Remove(filePath)
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

func (s *LocalStorage) Get(filename string) (io.ReadCloser, int64, error) {
	name := extractName(filename)
	if name == "" {
		return nil, 0, fmt.Errorf("无法从文件名中提取名称: %s", filename)
	}

	filePath := filepath.Join(s.basePath, name, filename)

	s.mu.RLock()
	defer s.mu.RUnlock()

	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, fmt.Errorf("打开文件失败: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return file, info.Size(), nil
}

func (s *LocalStorage) Query(prefix string) ([]model.FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var files []model.FileInfo

	// 读取所有子目录
	subDirs, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	for _, subDir := range subDirs {
		if !subDir.IsDir() {
			continue
		}

		// 读取子目录中的文件
		subPath := filepath.Join(s.basePath, subDir.Name())
		entries, err := os.ReadDir(subPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			if !strings.HasPrefix(name, prefix) {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			files = append(files, model.FileInfo{
				Name:         name,
				Size:         info.Size(),
				ModifiedTime: info.ModTime().Format(time.RFC3339),
			})
		}
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModifiedTime > files[j].ModifiedTime
	})

	return files, nil
}

func (s *LocalStorage) Delete(filename string) error {
	name := extractName(filename)
	if name == "" {
		return fmt.Errorf("无法从文件名中提取名称: %s", filename)
	}

	filePath := filepath.Join(s.basePath, name, filename)

	s.mu.Lock()
	defer s.mu.Unlock()

	return os.Remove(filePath)
}

func (s *LocalStorage) ListAll() ([]model.FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var files []model.FileInfo

	// 读取所有子目录
	subDirs, err := os.ReadDir(s.basePath)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	for _, subDir := range subDirs {
		if !subDir.IsDir() {
			continue
		}

		// 读取子目录中的文件
		subPath := filepath.Join(s.basePath, subDir.Name())
		entries, err := os.ReadDir(subPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			files = append(files, model.FileInfo{
				Name:         entry.Name(),
				Size:         info.Size(),
				ModifiedTime: info.ModTime().Format(time.RFC3339),
			})
		}
	}

	return files, nil
}
