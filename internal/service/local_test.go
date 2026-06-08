package service

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestNewLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(tmpDir)
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}

	if storage == nil {
		t.Fatal("存储实例不应为 nil")
	}

	if storage.basePath != tmpDir {
		t.Errorf("存储路径期望 %s，实际 %s", tmpDir, storage.basePath)
	}

	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("存储目录应已创建")
	}
}

func TestLocalStorageSaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(tmpDir)

	content := []byte("test content")
	filename := "test_abc123_20240101120000.zip"

	err := storage.Save(filename, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("保存文件失败: %v", err)
	}

	// 验证子目录已创建
	subDir := filepath.Join(tmpDir, "test")
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("子目录 test 应已创建")
	}

	// 验证文件在子目录中
	filePath := filepath.Join(subDir, filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("文件应已保存到子目录")
	}

	reader, size, err := storage.Get(filename)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	defer reader.Close()

	if size != int64(len(content)) {
		t.Errorf("文件大小期望 %d，实际 %d", len(content), size)
	}

	readContent, _ := io.ReadAll(reader)
	if !bytes.Equal(readContent, content) {
		t.Error("读取内容与写入内容不一致")
	}
}

func TestLocalStorageGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(tmpDir)

	_, _, err := storage.Get("nonexistent_nonexist_20240101120000.zip")
	if err == nil {
		t.Error("读取不存在的文件应返回错误")
	}
}

func TestLocalStorageQuery(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(tmpDir)

	files := []string{
		"app1_abc111_20240101120000.zip",
		"app1_abc222_20240101130000.zip",
		"app2_abc111_20240101140000.zip",
		"app1_def333_20240101150000.zip",
	}

	for _, f := range files {
		err := storage.Save(f, bytes.NewReader([]byte("content")))
		if err != nil {
			t.Fatalf("保存文件 %s 失败: %v", f, err)
		}
	}

	// 验证子目录结构
	for _, name := range []string{"app1", "app2"} {
		subDir := filepath.Join(tmpDir, name)
		if _, err := os.Stat(subDir); os.IsNotExist(err) {
			t.Errorf("子目录 %s 应已创建", name)
		}
	}

	// 测试按名称查询
	results, err := storage.Query("app1_")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(results) != 3 {
		t.Errorf("期望返回 3 个文件，实际 %d", len(results))
	}

	// 测试按名称和 commit 查询
	results, err = storage.Query("app1_abc")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("期望返回 2 个文件，实际 %d", len(results))
	}

	// 测试空前缀（返回所有）
	results, err = storage.Query("")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(results) != 4 {
		t.Errorf("期望返回 4 个文件，实际 %d", len(results))
	}

	// 测试无匹配
	results, err = storage.Query("nonexistent_")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("期望返回 0 个文件，实际 %d", len(results))
	}
}

func TestLocalStorageDelete(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(tmpDir)

	filename := "test_abc123_20240101120000.zip"
	content := []byte("content")

	storage.Save(filename, bytes.NewReader(content))

	err := storage.Delete(filename)
	if err != nil {
		t.Fatalf("删除失败: %v", err)
	}

	// 验证已删除
	filePath := filepath.Join(tmpDir, "test", filename)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("文件应已被删除")
	}
}

func TestLocalStorageListAll(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(tmpDir)

	storage.Save("myapp_log1_20240101120000.zip", bytes.NewReader([]byte("content1")))
	storage.Save("myapp_log2_20240101120001.zip", bytes.NewReader([]byte("content2")))
	storage.Save("other_log1_20240101120002.zip", bytes.NewReader([]byte("content3")))

	files, err := storage.ListAll()
	if err != nil {
		t.Fatalf("列出文件失败: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("期望返回 3 个文件，实际 %d", len(files))
	}
}

func TestLocalStorageConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	storage, _ := NewLocalStorage(tmpDir)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			filename := "concurrent" + string(rune('a'+idx)) + "_commit1_20240101120000.zip"
			err := storage.Save(filename, bytes.NewReader([]byte("content")))
			if err != nil {
				t.Errorf("并发保存失败: %v", err)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	files, _ := storage.ListAll()
	if len(files) != 10 {
		t.Errorf("期望 10 个文件，实际 %d", len(files))
	}
}
