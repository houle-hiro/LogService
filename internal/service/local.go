package service

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"log-service/internal/model"
)

// LocalStorage 本地磁盘存储，目录结构 data/{username}/{filename}
type LocalStorage struct {
	basePath string

	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// NewLocalStorage 创建本地存储
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("创建存储目录失败: %w", err)
	}
	return &LocalStorage{
		basePath: basePath,
		locks:    make(map[string]*sync.Mutex),
	}, nil
}

// userLock 取某个用户的写锁，避免同名文件并发写
func (s *LocalStorage) userLock(username string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.locks[username]
	if !ok {
		lock = &sync.Mutex{}
		s.locks[username] = lock
	}
	return lock
}

// userDir 返回用户目录并确保不越界
func (s *LocalStorage) userDir(username string) (string, string, error) {
	name, err := NormalizeUsername(username)
	if err != nil {
		return "", "", err
	}
	dir, err := SafeJoin(s.basePath, name)
	if err != nil {
		return "", "", err
	}
	return name, dir, nil
}

// Save 上传文件。先写临时文件再 rename，避免读到半截文件。
func (s *LocalStorage) Save(username, filename string, reader io.Reader) (bool, int64, error) {
	if err := ValidateFilename(filename); err != nil {
		return false, 0, err
	}
	name, dir, err := s.userDir(username)
	if err != nil {
		return false, 0, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, 0, fmt.Errorf("创建用户目录失败: %w", err)
	}

	lock := s.userLock(name)
	lock.Lock()
	defer lock.Unlock()

	target, err := SafeJoin(dir, filename)
	if err != nil {
		return false, 0, err
	}

	overwritten := false
	if info, statErr := os.Stat(target); statErr == nil && !info.IsDir() {
		overwritten = true
	}

	tmp, err := os.CreateTemp(dir, tempFilePrefix+"*")
	if err != nil {
		return false, 0, fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	size, copyErr := io.Copy(tmp, reader)
	closeErr := tmp.Close()
	if copyErr != nil {
		return false, 0, fmt.Errorf("写入文件失败: %w", copyErr)
	}
	if closeErr != nil {
		return false, 0, fmt.Errorf("写入文件失败: %w", closeErr)
	}
	if err := os.Chmod(tmpPath, 0644); err != nil {
		return false, 0, fmt.Errorf("设置文件权限失败: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		return false, 0, fmt.Errorf("保存文件失败: %w", err)
	}

	return overwritten, size, nil
}

// Get 打开文件
func (s *LocalStorage) Get(username, filename string) (io.ReadCloser, int64, error) {
	if err := ValidateFilename(filename); err != nil {
		return nil, 0, err
	}
	_, dir, err := s.userDir(username)
	if err != nil {
		return nil, 0, err
	}
	target, err := SafeJoin(dir, filename)
	if err != nil {
		return nil, 0, err
	}

	file, err := os.Open(target)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, fmt.Errorf("打开文件失败: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("获取文件信息失败: %w", err)
	}
	if info.IsDir() {
		file.Close()
		return nil, 0, ErrNotFound
	}
	return file, info.Size(), nil
}

// Query 列出某用户的文件
func (s *LocalStorage) Query(username, keyword string) ([]model.FileInfo, error) {
	name, dir, err := s.userDir(username)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.FileInfo{}, nil
		}
		return nil, fmt.Errorf("读取用户目录失败: %w", err)
	}

	keyword = strings.ToLower(strings.TrimSpace(keyword))
	files := make([]model.FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || IsTempFile(entry.Name()) {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(entry.Name()), keyword) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, model.FileInfo{
			Username:     name,
			Name:         entry.Name(),
			Size:         info.Size(),
			ModifiedTime: info.ModTime().Format(time.RFC3339),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModifiedTime > files[j].ModifiedTime
	})
	return files, nil
}

// Delete 删除文件
func (s *LocalStorage) Delete(username, filename string) error {
	if err := ValidateFilename(filename); err != nil {
		return err
	}
	name, dir, err := s.userDir(username)
	if err != nil {
		return err
	}

	lock := s.userLock(name)
	lock.Lock()
	defer lock.Unlock()

	target, err := SafeJoin(dir, filename)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

// ListAll 列出所有用户的文件
func (s *LocalStorage) ListAll() ([]model.FileInfo, error) {
	users, err := s.ListUsers()
	if err != nil {
		return nil, err
	}

	files := make([]model.FileInfo, 0)
	for _, user := range users {
		userFiles, err := s.Query(user, "")
		if err != nil {
			continue
		}
		files = append(files, userFiles...)
	}
	return files, nil
}

// ListUsers 列出所有用户目录
func (s *LocalStorage) ListUsers() ([]string, error) {
	entries, err := os.ReadDir(s.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("读取存储目录失败: %w", err)
	}

	users := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		users = append(users, entry.Name())
	}
	return users, nil
}

// PruneEmptyUserDirs 删除空用户目录，返回删除数量
func (s *LocalStorage) PruneEmptyUserDirs() int {
	users, err := s.ListUsers()
	if err != nil {
		return 0
	}
	removed := 0
	for _, user := range users {
		dir, err := SafeJoin(s.basePath, user)
		if err != nil {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			continue
		}
		if err := os.Remove(dir); err == nil {
			removed++
		}
	}
	return removed
}