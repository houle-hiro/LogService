package service

import (
	"strings"
	"testing"
)

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{"zip", "release.zip", false},
		{"多段扩展名", "app-v1.0.tar.gz", false},
		{"日志文件", "build_20240101.log", false},
		{"图片", "screenshot.png", false},
		{"无扩展名", "Makefile", false},
		{"中文名", "设计文档.pdf", false},
		{"空格与括号", "my report (1).docx", false},
		{"带 commit 与时间的旧名", "myapp_abc123_20240101120000.zip", false},
		{"空文件名", "", true},
		{"点", ".", true},
		{"双点", "..", true},
		{"含斜杠", "a/b.txt", true},
		{"含反斜杠", `a\b.txt`, true},
		{"含换行", "a\nb.txt", true},
		{"含 NUL", "a\x00b.txt", true},
		{"超长", strings.Repeat("a", 256), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilename(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"普通", "alice", "alice", false},
		{"大写转小写", "Alice", "alice", false},
		{"邮箱", "alice@example.com", "alice@example.com", false},
		{"带点与下划线", "a.b_c-1", "a.b_c-1", false},
		{"前后空格", "  bob  ", "bob", false},
		{"空", "", "", true},
		{"含斜杠", "a/b", "", true},
		{"路径穿越", "../root", "", true},
		{"以点开头", ".hidden", "", true},
		{"以连字符开头", "-abc", "", true},
		{"含空格", "a b", "", true},
		{"超长", strings.Repeat("a", 65), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeUsername(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeUsername(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("NormalizeUsername(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSafeJoin(t *testing.T) {
	base := t.TempDir()

	if _, err := SafeJoin(base, "alice", "a.txt"); err != nil {
		t.Errorf("正常路径不应报错: %v", err)
	}
	if _, err := SafeJoin(base, "..", "etc", "passwd"); err == nil {
		t.Error("越界路径应报错")
	}
	if _, err := SafeJoin(base, "alice", "..", "..", "x"); err == nil {
		t.Error("多级越界应报错")
	}
}

func TestIsTempFile(t *testing.T) {
	if !IsTempFile(".upload-123") {
		t.Error("临时文件应被识别")
	}
	if IsTempFile("normal.txt") {
		t.Error("普通文件不应被识别为临时文件")
	}
}