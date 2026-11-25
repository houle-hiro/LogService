package service

import (
	"log"
	"sync"
	"time"
)

// Cleanup 按文件修改时间清理过期文件
type Cleanup struct {
	storage  Storage
	maxAge   time.Duration
	interval time.Duration
	stopChan chan struct{}
	stopOnce sync.Once
}

// NewCleanup 创建清理器，maxAgeSeconds <= 0 表示不清理
func NewCleanup(storage Storage, maxAgeSeconds int) *Cleanup {
	return &Cleanup{
		storage:  storage,
		maxAge:   time.Duration(maxAgeSeconds) * time.Second,
		interval: 1 * time.Hour,
		stopChan: make(chan struct{}),
	}
}

// Start 启动后台清理
func (c *Cleanup) Start() {
	go c.run()
}

// Stop 停止清理
func (c *Cleanup) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopChan)
	})
}

func (c *Cleanup) run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	c.clean()

	for {
		select {
		case <-ticker.C:
			c.clean()
		case <-c.stopChan:
			return
		}
	}
}

// CleanOnce 立即执行一次清理，供测试和手动触发
func (c *Cleanup) CleanOnce() {
	c.clean()
}

func (c *Cleanup) clean() {
	if c.maxAge <= 0 {
		return
	}

	files, err := c.storage.ListAll()
	if err != nil {
		log.Printf("清理任务读取文件列表失败: %v", err)
		return
	}

	now := time.Now()
	deleted := 0

	for _, file := range files {
		modifiedTime, err := time.Parse(time.RFC3339, file.ModifiedTime)
		if err != nil {
			continue
		}

		if now.Sub(modifiedTime) > c.maxAge {
			if err := c.storage.Delete(file.Username, file.Name); err != nil {
				log.Printf("删除过期文件失败 %s/%s: %v", file.Username, file.Name, err)
				continue
			}
			deleted++
			log.Printf("删除过期文件: %s/%s", file.Username, file.Name)
		}
	}

	// 顺手清掉变空的用户目录
	if pruner, ok := c.storage.(interface{ PruneEmptyUserDirs() int }); ok {
		removed := pruner.PruneEmptyUserDirs()
		if removed > 0 {
			log.Printf("清理空用户目录 %d 个", removed)
		}
	}

	if deleted > 0 {
		log.Printf("清理任务完成，删除 %d 个过期文件", deleted)
	}
}