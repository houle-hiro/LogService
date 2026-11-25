package service

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewLocalStorage(t *testing.T) {
	tmpDir := t.TempDir()

	storage, err := NewLocalStorage(filepath.Join(tmpDir, "data"))
	if err != nil {
		t.Fatalf("创建存储失败: %v", err)
	}
	if storage.basePath != filepath.Join(tmpDir, "data") {
		t.Errorf("存储路径不符: %s", storage.basePath)
	}
	if _, err := os.Stat(storage.basePath); os.IsNotExist(err) {
		t.Error("存储目录应已创建")
	}
}

func TestSaveAndGet(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	content := []byte("hello world")
	overwritten, size, err := storage.Save("alice", "notes.txt", bytes.NewReader(content))
	if err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if overwritten {
		t.Error("首次保存不应标记为覆盖")
	}
	if size != int64(len(content)) {
		t.Errorf("大小期望 %d，实际 %d", len(content), size)
	}

	// 验证目录结构 data/alice/notes.txt
	if _, err := os.Stat(filepath.Join(storage.basePath, "alice", "notes.txt")); err != nil {
		t.Errorf("文件应保存在用户目录下: %v", err)
	}

	reader, gotSize, err := storage.Get("alice", "notes.txt")
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	defer reader.Close()
	if gotSize != int64(len(content)) {
		t.Errorf("读取大小期望 %d，实际 %d", len(content), gotSize)
	}
	got, _ := io.ReadAll(reader)
	if !bytes.Equal(got, content) {
		t.Error("内容不一致")
	}
}

func TestSaveOverwrite(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	storage.Save("alice", "a.txt", bytes.NewReader([]byte("v1")))
	overwritten, size, err := storage.Save("alice", "a.txt", bytes.NewReader([]byte("v2-long")))
	if err != nil {
		t.Fatalf("覆盖保存失败: %v", err)
	}
	if !overwritten {
		t.Error("应标记为覆盖")
	}
	if size != int64(len("v2-long")) {
		t.Errorf("大小期望 %d，实际 %d", len("v2-long"), size)
	}

	reader, _, _ := storage.Get("alice", "a.txt")
	defer reader.Close()
	got, _ := io.ReadAll(reader)
	if string(got) != "v2-long" {
		t.Errorf("内容应为 v2-long，实际 %s", got)
	}
}

func TestGetNotFound(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	if _, _, err := storage.Get("alice", "missing.txt"); err != ErrNotFound {
		t.Errorf("应返回 ErrNotFound，实际 %v", err)
	}
	if _, _, err := storage.Get("nobody", "missing.txt"); err != ErrNotFound {
		t.Errorf("不存在的用户应返回 ErrNotFound，实际 %v", err)
	}
}

func TestGetRejectsTraversal(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	if _, _, err := storage.Get("alice", "../etc/passwd"); err == nil {
		t.Error("路径穿越应报错")
	}
	if _, _, err := storage.Get("../alice", "a.txt"); err == nil {
		t.Error("非法用户名应报错")
	}
}

func TestQueryIsolation(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	storage.Save("alice", "a1.log", bytes.NewReader([]byte("1")))
	storage.Save("alice", "a2.zip", bytes.NewReader([]byte("2")))
	storage.Save("bob", "b1.png", bytes.NewReader([]byte("3")))

	aliceFiles, err := storage.Query("alice", "")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(aliceFiles) != 2 {
		t.Fatalf("alice 期望 2 个文件，实际 %d", len(aliceFiles))
	}
	for _, f := range aliceFiles {
		if f.Username != "alice" {
			t.Errorf("不应出现其他用户的文件: %+v", f)
		}
	}

	bobFiles, _ := storage.Query("bob", "")
	if len(bobFiles) != 1 || bobFiles[0].Name != "b1.png" {
		t.Errorf("bob 查询结果不符: %+v", bobFiles)
	}

	missing, err := storage.Query("carol", "")
	if err != nil {
		t.Fatalf("不存在的用户不应报错: %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("不存在的用户应返回空列表，实际 %d", len(missing))
	}
}

func TestQueryKeyword(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	storage.Save("alice", "Build-Log.TXT", bytes.NewReader([]byte("1")))
	storage.Save("alice", "screenshot.png", bytes.NewReader([]byte("2")))

	files, _ := storage.Query("alice", "build")
	if len(files) != 1 || files[0].Name != "Build-Log.TXT" {
		t.Errorf("关键字过滤应不区分大小写，实际 %+v", files)
	}

	files, _ = storage.Query("alice", ".png")
	if len(files) != 1 {
		t.Errorf("期望 1 个 png，实际 %d", len(files))
	}

	files, _ = storage.Query("alice", "nothing")
	if len(files) != 0 {
		t.Errorf("无匹配应返回空，实际 %d", len(files))
	}
}

func TestDelete(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	storage.Save("alice", "a.txt", bytes.NewReader([]byte("1")))
	if err := storage.Delete("alice", "a.txt"); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	if _, _, err := storage.Get("alice", "a.txt"); err != ErrNotFound {
		t.Errorf("删除后应不存在，实际 %v", err)
	}
	if err := storage.Delete("alice", "a.txt"); err != ErrNotFound {
		t.Errorf("重复删除应返回 ErrNotFound，实际 %v", err)
	}
}

func TestListAllAndUsers(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	storage.Save("alice", "a.txt", bytes.NewReader([]byte("1")))
	storage.Save("bob", "b.txt", bytes.NewReader([]byte("2")))

	all, err := storage.ListAll()
	if err != nil {
		t.Fatalf("ListAll 失败: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("期望 2 个文件，实际 %d", len(all))
	}

	users, _ := storage.ListUsers()
	if len(users) != 2 {
		t.Errorf("期望 2 个用户目录，实际 %d", len(users))
	}
}

func TestTempFileHidden(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())
	storage.Save("alice", "a.txt", bytes.NewReader([]byte("1")))

	// 手动放一个临时文件，查询时不应出现
	tmp := filepath.Join(storage.basePath, "alice", tempFilePrefix+"123")
	os.WriteFile(tmp, []byte("partial"), 0644)

	files, _ := storage.Query("alice", "")
	if len(files) != 1 {
		t.Errorf("临时文件不应出现在列表中，实际 %d", len(files))
	}
}

func TestPruneEmptyUserDirs(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	storage.Save("alice", "a.txt", bytes.NewReader([]byte("1")))
	os.MkdirAll(filepath.Join(storage.basePath, "emptyuser"), 0755)

	removed := storage.PruneEmptyUserDirs()
	if removed != 1 {
		t.Errorf("期望清理 1 个空目录，实际 %d", removed)
	}
	if _, err := os.Stat(filepath.Join(storage.basePath, "emptyuser")); !os.IsNotExist(err) {
		t.Error("空目录应已删除")
	}
	if _, err := os.Stat(filepath.Join(storage.basePath, "alice")); err != nil {
		t.Error("非空目录不应被删除")
	}
}

func TestConcurrentSave(t *testing.T) {
	storage, _ := NewLocalStorage(t.TempDir())

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "file-" + string(rune('a'+idx)) + ".txt"
			if _, _, err := storage.Save("alice", name, bytes.NewReader([]byte("content"))); err != nil {
				t.Errorf("并发保存失败: %v", err)
			}
		}(i)
	}
	wg.Wait()

	files, _ := storage.Query("alice", "")
	if len(files) != 10 {
		t.Errorf("期望 10 个文件，实际 %d", len(files))
	}
}