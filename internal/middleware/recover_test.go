package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecover(t *testing.T) {
	// 创建一个会 panic 的处理器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("测试 panic")
	})

	// 包装恢复中间件
	recoveredHandler := Recover(handler)

	// 创建测试请求
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// 执行请求，不应 panic
	recoveredHandler.ServeHTTP(rr, req)

	// 验证返回 500
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("期望状态 500，实际 %d", rr.Code)
	}
}

func TestRecoverNoPanic(t *testing.T) {
	// 创建正常处理器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	recoveredHandler := Recover(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	recoveredHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("期望状态 200，实际 %d", rr.Code)
	}
}
