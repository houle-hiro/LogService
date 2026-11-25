package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"log-service/internal/model"
)

func buildUploadRequest(t *testing.T, query string, fieldName, filename, content string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("构造 multipart 失败: %v", err)
	}
	part.Write([]byte(content))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload"+query, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func decodeUpload(t *testing.T, body string) model.UploadResponse {
	t.Helper()
	var resp model.Response
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, body)
	}
	dataBytes, _ := json.Marshal(resp.Data)
	var up model.UploadResponse
	if err := json.Unmarshal(dataBytes, &up); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	return up
}

func TestUploadAnyExtension(t *testing.T) {
	cases := []struct {
		name string
		file string
	}{
		{"zip", "release.zip"},
		{"tar.gz", "app-v1.tar.gz"},
		{"log", "build.log"},
		{"png", "shot.png"},
		{"中文", "设计文档.pdf"},
		{"无扩展名", "Makefile"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			handler, storage := newTestHandler()
			req := buildUploadRequest(t, "?username=alice", "file", c.file, "content")
			rr := httptest.NewRecorder()

			handler.handleUpload(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("期望 200，实际 %d: %s", rr.Code, rr.Body.String())
			}
			up := decodeUpload(t, rr.Body.String())
			if up.Name != c.file {
				t.Errorf("文件名期望 %q，实际 %q", c.file, up.Name)
			}
			if up.Username != "alice" {
				t.Errorf("用户名期望 alice，实际 %q", up.Username)
			}
			if _, ok := storage.files["alice"][c.file]; !ok {
				t.Error("文件应已保存到 alice 目录")
			}
		})
	}
}

func TestUploadOverwriteFlag(t *testing.T) {
	handler, _ := newTestHandler()

	rr := httptest.NewRecorder()
	handler.handleUpload(rr, buildUploadRequest(t, "?username=alice", "file", "a.txt", "v1"))
	if rr.Code != http.StatusOK {
		t.Fatalf("首次上传失败: %d", rr.Code)
	}
	if decodeUpload(t, rr.Body.String()).Overwritten {
		t.Error("首次上传不应标记覆盖")
	}

	rr = httptest.NewRecorder()
	handler.handleUpload(rr, buildUploadRequest(t, "?username=alice", "file", "a.txt", "v2"))
	if rr.Code != http.StatusOK {
		t.Fatalf("覆盖上传失败: %d", rr.Code)
	}
	if !decodeUpload(t, rr.Body.String()).Overwritten {
		t.Error("同名上传应标记覆盖")
	}
}

func TestUploadUsernameFromForm(t *testing.T) {
	handler, storage := newTestHandler()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("username", "Bob")
	part, _ := writer.CreateFormFile("file", "x.txt")
	part.Write([]byte("data"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()
	handler.handleUpload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := storage.files["bob"]["x.txt"]; !ok {
		t.Error("用户名应转小写并保存到 bob 目录")
	}
}

func TestUploadInvalidMethod(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleUpload(rr, httptest.NewRequest(http.MethodGet, "/upload", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", rr.Code)
	}
}

func TestUploadMissingFileField(t *testing.T) {
	handler, _ := newTestHandler()
	req := buildUploadRequest(t, "?username=alice", "other", "a.txt", "content")
	rr := httptest.NewRecorder()
	handler.handleUpload(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestUploadMissingUsername(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleUpload(rr, buildUploadRequest(t, "", "file", "a.txt", "content"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestUploadInvalidUsername(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleUpload(rr, buildUploadRequest(t, "?username=../root", "file", "a.txt", "content"))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestUploadTooLarge(t *testing.T) {
	handler, _ := newTestHandler()
	req := buildUploadRequest(t, "?username=alice", "file", "big.bin", strings.Repeat("x", 1024))
	rr := httptest.NewRecorder()
	req.Body = http.MaxBytesReader(rr, req.Body, 10)

	handler.handleUpload(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("期望 413，实际 %d: %s", rr.Code, rr.Body.String())
	}
}

func TestTrimUploadFilename(t *testing.T) {
	cases := map[string]string{
		"a.txt":                 "a.txt",
		`C:\Users\me\a.txt`:     "a.txt",
		"/tmp/dir/a.txt":        "a.txt",
		"../../etc/passwd":      "passwd",
		`..\..\windows\a.dll`:   "a.dll",
	}
	for input, want := range cases {
		if got := trimUploadFilename(input); got != want {
			t.Errorf("trimUploadFilename(%q) = %q, want %q", input, got, want)
		}
	}
}