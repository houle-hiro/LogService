package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownload(t *testing.T) {
	handler, storage := newTestHandler()
	storage.put("alice", "notes.txt", []byte("hello"))

	req := httptest.NewRequest(http.MethodGet, "/download?username=alice&filename=notes.txt", nil)
	rr := httptest.NewRecorder()
	handler.handleDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", rr.Code, rr.Body.String())
	}
	if rr.Body.String() != "hello" {
		t.Errorf("内容不符: %q", rr.Body.String())
	}
	if got := rr.Header().Get("Content-Length"); got != "5" {
		t.Errorf("Content-Length 期望 5，实际 %q", got)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type 期望 text/plain，实际 %q", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="notes.txt"`) {
		t.Errorf("Content-Disposition 不符: %q", cd)
	}
}

func TestDownloadContentTypeByExtension(t *testing.T) {
	cases := map[string]string{
		"a.png":  "image/png",
		"a.zip":  "application/zip",
		"a.json": "application/json",
		"a.bin":  "application/octet-stream",
	}

	for name, want := range cases {
		handler, storage := newTestHandler()
		storage.put("alice", name, []byte("x"))

		req := httptest.NewRequest(http.MethodGet, "/download?username=alice&filename="+name, nil)
		rr := httptest.NewRecorder()
		handler.handleDownload(rr, req)

		if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, want) {
			t.Errorf("%s Content-Type 期望 %q，实际 %q", name, want, got)
		}
	}
}

func TestDownloadChineseFilename(t *testing.T) {
	handler, storage := newTestHandler()
	name := "设计文档.pdf"
	storage.put("alice", name, []byte("pdf"))

	req := httptest.NewRequest(http.MethodGet, "/download?username=alice&filename="+name, nil)
	rr := httptest.NewRecorder()
	handler.handleDownload(rr, req)

	cd := rr.Header().Get("Content-Disposition")
	if !strings.Contains(cd, "filename*=UTF-8''") {
		t.Errorf("应带 RFC 5987 文件名，实际 %q", cd)
	}
	if !strings.Contains(cd, "%E8%AE%BE%E8%AE%A1") {
		t.Errorf("中文未按 UTF-8 百分号编码: %q", cd)
	}
}

func TestDownloadIsolation(t *testing.T) {
	handler, storage := newTestHandler()
	storage.put("bob", "secret.txt", []byte("bob only"))

	// alice 拿不到 bob 的文件
	req := httptest.NewRequest(http.MethodGet, "/download?username=alice&filename=secret.txt", nil)
	rr := httptest.NewRecorder()
	handler.handleDownload(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("跨用户下载应 404，实际 %d", rr.Code)
	}
}

func TestDownloadInvalidMethod(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleDownload(rr, httptest.NewRequest(http.MethodPost, "/download", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", rr.Code)
	}
}

func TestDownloadMissingFilename(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleDownload(rr, httptest.NewRequest(http.MethodGet, "/download?username=alice", nil))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestDownloadPathTraversal(t *testing.T) {
	handler, _ := newTestHandler()
	for _, filename := range []string{"../etc/passwd", `..\windows\a.dll`, "a/b.txt"} {
		req := httptest.NewRequest(http.MethodGet, "/download?username=alice&filename="+filename, nil)
		rr := httptest.NewRecorder()
		handler.handleDownload(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("filename=%q 期望 400，实际 %d", filename, rr.Code)
		}
	}
}

func TestDownloadNotFound(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleDownload(rr, httptest.NewRequest(http.MethodGet, "/download?username=alice&filename=missing.txt", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际 %d", rr.Code)
	}
}

func TestBuildContentDisposition(t *testing.T) {
	cd := buildContentDisposition(`a"b.txt`)
	if !strings.Contains(cd, `filename="a_b.txt"`) {
		t.Errorf("ASCII 回退名应替换引号，实际 %q", cd)
	}
}

func TestEncodeRFC5987(t *testing.T) {
	if got := encodeRFC5987("a b.txt"); got != "a%20b.txt" {
		t.Errorf("空格应编码为 %%20，实际 %q", got)
	}
	if got := encodeRFC5987("中文.log"); got != "%E4%B8%AD%E6%96%87.log" {
		t.Errorf("中文编码不符: %q", got)
	}
}