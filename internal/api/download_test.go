package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleDownload(t *testing.T) {
	handler, storage := newTestHandler()

	// 预先保存文件
	storage.files["test_abc123_20240101120000.zip"] = []byte("zip content")

	req := httptest.NewRequest("GET", "/download?filename=test_abc123_20240101120000.zip", nil)
	rr := httptest.NewRecorder()

	handler.handleDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("期望状态 200，实际 %d, 响应: %s", rr.Code, rr.Body.String())
	}

	// 验证响应头
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/zip" {
		t.Errorf("期望 Content-Type application/zip，实际 %s", contentType)
	}

	// 验证响应体
	if rr.Body.String() != "zip content" {
		t.Errorf("期望响应体 'zip content'，实际 %s", rr.Body.String())
	}
}

func TestHandleDownloadInvalidMethod(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("POST", "/download", nil)
	rr := httptest.NewRecorder()

	handler.handleDownload(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态 405，实际 %d", rr.Code)
	}
}

func TestHandleDownloadMissingFilename(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("GET", "/download", nil)
	rr := httptest.NewRecorder()

	handler.handleDownload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望状态 400，实际 %d", rr.Code)
	}
}

func TestHandleDownloadInvalidFilename(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("GET", "/download?filename=invalid.zip", nil)
	rr := httptest.NewRecorder()

	handler.handleDownload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望状态 400，实际 %d", rr.Code)
	}
}

func TestHandleDownloadNotFound(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("GET", "/download?filename=test_abc123_20240101120000.zip", nil)
	rr := httptest.NewRecorder()

	handler.handleDownload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("期望状态 404，实际 %d", rr.Code)
	}
}
