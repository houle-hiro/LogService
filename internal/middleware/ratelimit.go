package middleware

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	limit      int
	interval   time.Duration
	visitors   map[string]*visitor
	mu         sync.RWMutex
	stopChan   chan struct{}
	stopOnce   sync.Once
}

type visitor struct {
	count    int
	lastSeen time.Time
}

func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		limit:    limit,
		interval: interval,
		visitors: make(map[string]*visitor),
		stopChan: make(chan struct{}),
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) Stop() {
	rl.stopOnce.Do(func() {
		close(rl.stopChan)
	})
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	v, exists := rl.visitors[ip]
	now := time.Now()
	
	if !exists || now.Sub(v.lastSeen) > rl.interval {
		rl.visitors[ip] = &visitor{count: 1, lastSeen: now}
		return true
	}
	
	if v.count >= rl.limit {
		return false
	}
	
	v.count++
	v.lastSeen = now
	return true
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.interval * 2)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, v := range rl.visitors {
				if now.Sub(v.lastSeen) > rl.interval*2 {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}

func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if !rl.Allow(ip) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"code":429,"message":"请求过于频繁"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
