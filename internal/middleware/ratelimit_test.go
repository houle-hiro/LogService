package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllow(t *testing.T) {
	rl := NewRateLimiter(3, 1*time.Second)
	defer rl.Stop()

	ip := "127.0.0.1"

	// 前 3 次应该允许
	if !rl.Allow(ip) {
		t.Error("第 1 次请求应被允许")
	}
	if !rl.Allow(ip) {
		t.Error("第 2 次请求应被允许")
	}
	if !rl.Allow(ip) {
		t.Error("第 3 次请求应被允许")
	}

	// 第 4 次应该被拒绝
	if rl.Allow(ip) {
		t.Error("第 4 次请求应被拒绝")
	}
}

func TestRateLimiterResetAfterInterval(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	defer rl.Stop()

	ip := "127.0.0.1"

	// 用完全部配额
	rl.Allow(ip)
	rl.Allow(ip)

	// 应该被拒绝
	if rl.Allow(ip) {
		t.Error("超出限流应被拒绝")
	}

	// 等待窗口重置
	time.Sleep(150 * time.Millisecond)

	// 应该被允许
	if !rl.Allow(ip) {
		t.Error("窗口重置后应被允许")
	}
}

func TestRateLimiterDifferentIPs(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)
	defer rl.Stop()

	ip1 := "127.0.0.1"
	ip2 := "127.0.0.2"

	// 用完全部配额
	rl.Allow(ip1)
	rl.Allow(ip1)

	// ip1 应被拒绝
	if rl.Allow(ip1) {
		t.Error("ip1 超出限流应被拒绝")
	}

	// ip2 应被允许
	if !rl.Allow(ip2) {
		t.Error("ip2 应被允许")
	}
}

func TestRateLimiterHandler(t *testing.T) {
	rl := NewRateLimiter(2, 1*time.Second)
	defer rl.Stop()

	// 创建一个简单的处理器
	handler := rl.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// 前 2 次请求应成功
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("第 %d 次请求期望状态 200，实际 %d", i+1, rr.Code)
		}
	}

	// 第 3 次请求应被限流
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("期望状态 429，实际 %d", rr.Code)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := NewRateLimiter(2, 100*time.Millisecond)
	defer rl.Stop()

	ip := "127.0.0.1"
	rl.Allow(ip)

	// 等待清理周期
	time.Sleep(250 * time.Millisecond)

	// 清理后应该允许新的请求
	// 注意：cleanup 是异步的，可能不是立即生效
	// 这里主要验证不会 panic
	rl.Allow(ip)
}
