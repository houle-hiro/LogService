package service

import (
	"testing"
	"time"
)

func newTestCleanup(storage Storage, maxAge time.Duration) *Cleanup {
	return &Cleanup{
		storage:  storage,
		maxAge:   maxAge,
		interval: time.Hour,
		stopChan: make(chan struct{}),
	}
}

func TestCleanupRemovesExpired(t *testing.T) {
	storage := newMockStorage()
	storage.put("alice", "old.txt", []byte("old"), time.Now().Add(-2*time.Hour))
	storage.put("alice", "new.txt", []byte("new"), time.Now())
	storage.put("bob", "bob-old.log", []byte("old"), time.Now().Add(-3*time.Hour))

	newTestCleanup(storage, time.Hour).CleanOnce()

	aliceFiles, _ := storage.Query("alice", "")
	if len(aliceFiles) != 1 || aliceFiles[0].Name != "new.txt" {
		t.Errorf("alice 应只剩 new.txt，实际 %+v", aliceFiles)
	}
	bobFiles, _ := storage.Query("bob", "")
	if len(bobFiles) != 0 {
		t.Errorf("bob 的过期文件应被删除，实际 %+v", bobFiles)
	}
}

func TestCleanupKeepsFresh(t *testing.T) {
	storage := newMockStorage()
	storage.put("alice", "fresh.txt", []byte("x"), time.Now())

	newTestCleanup(storage, 24*time.Hour).CleanOnce()

	files, _ := storage.Query("alice", "")
	if len(files) != 1 {
		t.Errorf("未过期文件应保留，实际 %d", len(files))
	}
}

func TestCleanupDisabledWhenZero(t *testing.T) {
	storage := newMockStorage()
	storage.put("alice", "ancient.txt", []byte("x"), time.Now().Add(-100*24*time.Hour))

	newTestCleanup(storage, 0).CleanOnce()

	files, _ := storage.Query("alice", "")
	if len(files) != 1 {
		t.Errorf("maxAge 为 0 时不应清理，实际 %d", len(files))
	}
}

func TestCleanupStartStop(t *testing.T) {
	storage := newMockStorage()
	cleanup := NewCleanup(storage, 3600)
	cleanup.Start()
	cleanup.Stop()
	// 重复 Stop 不应 panic
	cleanup.Stop()
}