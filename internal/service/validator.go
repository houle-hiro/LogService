package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ErrNotFound 文件或用户目录不存在
var ErrNotFound = errors.New("文件不存在")

// ErrInvalidName 名称不合法
var ErrInvalidName = errors.New("名称不合法")

// usernamePattern 用户名：字母数字开头，允许 . _ @ + -，最长 64
var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._@+-]{0,63}$`)

// tempFilePrefix 上传中的临时文件前缀，列目录时必须跳过
const tempFilePrefix = ".upload-"

// NormalizeUsername 校验用户名并统一转小写
func NormalizeUsername(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return "", fmt.Errorf("%w: 用户名为空", ErrInvalidName)
	}
	if strings.Contains(name, "..") {
		return "", fmt.Errorf("%w: 用户名不能包含 ..", ErrInvalidName)
	}
	if !usernamePattern.MatchString(name) {
		return "", fmt.Errorf("%w: 用户名只允许字母、数字和 . _ @ + -，且以字母或数字开头，最长 64 字符", ErrInvalidName)
	}
	return name, nil
}

// ValidateFilename 校验文件名：保留原名，但禁止路径与不可见字符
func ValidateFilename(filename string) error {
	if filename == "" {
		return fmt.Errorf("%w: 文件名为空", ErrInvalidName)
	}
	if filename == "." || filename == ".." {
		return fmt.Errorf("%w: 文件名不能是 . 或 ..", ErrInvalidName)
	}
	if len(filename) > 255 {
		return fmt.Errorf("%w: 文件名最多 255 字节", ErrInvalidName)
	}
	if !utf8.ValidString(filename) {
		return fmt.Errorf("%w: 文件名不是合法的 UTF-8", ErrInvalidName)
	}
	for _, r := range filename {
		if r == '/' || r == '\\' || r < 0x20 || r == 0x7f {
			return fmt.Errorf("%w: 文件名包含路径分隔符或控制字符", ErrInvalidName)
		}
	}
	if filepath.Base(filename) != filename {
		return fmt.Errorf("%w: 文件名包含路径", ErrInvalidName)
	}
	return nil
}

// SafeJoin 在 base 下拼接路径，并确保结果不越出 base
func SafeJoin(base string, elems ...string) (string, error) {
	parts := append([]string{base}, elems...)
	target := filepath.Join(parts...)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", fmt.Errorf("%w: 路径解析失败", ErrInvalidName)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: 路径越界", ErrInvalidName)
	}
	return target, nil
}

// IsTempFile 判断是否为上传临时文件
func IsTempFile(name string) bool {
	return strings.HasPrefix(name, tempFilePrefix)
}