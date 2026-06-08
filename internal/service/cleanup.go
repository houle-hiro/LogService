package service

import (
	"log"
	"sync"
	"time"
)

type Cleanup struct {
	storage   Storage
	maxAge    time.Duration
	interval  time.Duration
	stopChan  chan struct{}
	stopOnce  sync.Once
}

func NewCleanup(storage Storage, maxAgeSeconds int) *Cleanup {
	return &Cleanup{
		storage:  storage,
		maxAge:   time.Duration(maxAgeSeconds) * time.Second,
		interval: 1 * time.Hour,
		stopChan: make(chan struct{}),
	}
}

func (c *Cleanup) Start() {
	go c.run()
}

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

func (c *Cleanup) clean() {
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
			if err := c.storage.Delete(file.Name); err != nil {
				log.Printf("删除过期文件失败 %s: %v", file.Name, err)
			} else {
				deleted++
				log.Printf("删除过期文件: %s", file.Name)
			}
		}
	}
	
	if deleted > 0 {
		log.Printf("清理任务完成，删除 %d 个过期文件", deleted)
	}
}
