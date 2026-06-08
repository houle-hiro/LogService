package api

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleUpload(t *testing.T) {
	handler, storage := newTestHandler()

	// 创建 multipart 请求
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test_abc123_20240101120000.zip")
	part.Write([]byte("zip content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	handler.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("期望状态 200，实际 %d, 响应: %s", rr.Code, rr.Body.String())
	}

	// 验证文件已保存
	if _, ok := storage.files["test_abc123_20240101120000.zip"]; !ok {
		t.Error("文件应已保存到存储")
	}
}

func TestHandleUploadInvalidMethod(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("GET", "/upload", nil)
	rr := httptest.NewRecorder()

	handler.handleUpload(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态 405，实际 %d", rr.Code)
	}
}

func TestHandleUploadMissingFile(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("POST", "/upload", nil)
	rr := httptest.NewRecorder()

	handler.handleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望状态 400，实际 %d", rr.Code)
	}
}

func TestHandleUploadInvalidFilename(t *testing.T) {
	handler, _ := newTestHandler()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "invalid.zip")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	handler.handleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望状态 400，实际 %d", rr.Code)
	}
}

func TestHandleUploadTooLarge(t *testing.T) {
	handler, _ := newTestHandler()

	// 创建超过限制的内容（这里通过设置很小的 MaxBytesReader 来模拟）
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test_abc123_20240101120000.zip")
	part.Write([]byte("content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	// 限制请求体大小为 1 字节
	req.Body = http.MaxBytesReader(rr, req.Body, 1)

	handler.handleUpload(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望状态 400，实际 %d", rr.Code)
	}
}
