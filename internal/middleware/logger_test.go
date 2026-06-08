package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogger(t *testing.T) {
	// 创建一个测试处理器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 包装日志中间件
	loggedHandler := Logger(handler)

	// 创建测试请求
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	// 执行请求
	loggedHandler.ServeHTTP(rr, req)

	// 验证响应
	if rr.Code != http.StatusOK {
		t.Errorf("期望状态 200，实际 %d", rr.Code)
	}

	if rr.Body.String() != "OK" {
		t.Errorf("期望响应体 'OK'，实际 %s", rr.Body.String())
	}
}

func TestLoggerWithErrorStatus(t *testing.T) {
	// 创建一个返回错误的处理器
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	})

	loggedHandler := Logger(handler)

	req := httptest.NewRequest("POST", "/api/upload", nil)
	rr := httptest.NewRecorder()

	loggedHandler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("期望状态 500，实际 %d", rr.Code)
	}
}
