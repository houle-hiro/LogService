package service

import (
	"io"

	"log-service/internal/model"
)

// Storage 文件存储抽象，所有操作都以 username 为隔离维度
type Storage interface {
	// Save 保存文件，返回是否覆盖了同名文件与写入字节数
	Save(username, filename string, reader io.Reader) (overwritten bool, size int64, err error)
	// Get 打开文件，不存在时返回 ErrNotFound
	Get(username, filename string) (io.ReadCloser, int64, error)
	// Query 列出某用户的文件，keyword 为空表示不过滤，匹配文件名子串（不区分大小写）
	Query(username, keyword string) ([]model.FileInfo, error)
	// Delete 删除文件，不存在时返回 ErrNotFound
	Delete(username, filename string) error
	// ListAll 列出所有用户的文件，供老化清理使用
	ListAll() ([]model.FileInfo, error)
	// ListUsers 列出已有用户目录
	ListUsers() ([]string, error)
}