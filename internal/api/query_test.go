package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"log-service/internal/model"
)

func decodeQuery(t *testing.T, body string) model.QueryResponse {
	t.Helper()
	var resp model.Response
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("解析响应失败: %v, body=%s", err, body)
	}
	dataBytes, _ := json.Marshal(resp.Data)
	var qr model.QueryResponse
	if err := json.Unmarshal(dataBytes, &qr); err != nil {
		t.Fatalf("解析 data 失败: %v", err)
	}
	return qr
}

func TestQueryIsolation(t *testing.T) {
	handler, storage := newTestHandler()
	storage.put("alice", "a1.log", []byte("1"))
	storage.put("alice", "a2.zip", []byte("2"))
	storage.put("bob", "b1.png", []byte("3"))

	rr := httptest.NewRecorder()
	handler.handleQuery(rr, httptest.NewRequest(http.MethodGet, "/query?username=alice", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	qr := decodeQuery(t, rr.Body.String())
	if qr.Count != 2 {
		t.Errorf("alice 期望 2 个文件，实际 %d", qr.Count)
	}
	if qr.Username != "alice" {
		t.Errorf("用户名期望 alice，实际 %q", qr.Username)
	}
	for _, f := range qr.Files {
		if f.Username != "alice" {
			t.Errorf("不应出现其他用户文件: %+v", f)
		}
	}
}

func TestQueryKeyword(t *testing.T) {
	handler, storage := newTestHandler()
	storage.put("alice", "Build-Log.TXT", []byte("1"))
	storage.put("alice", "shot.png", []byte("2"))

	rr := httptest.NewRecorder()
	handler.handleQuery(rr, httptest.NewRequest(http.MethodGet, "/query?username=alice&keyword=build", nil))

	qr := decodeQuery(t, rr.Body.String())
	if qr.Count != 1 || qr.Files[0].Name != "Build-Log.TXT" {
		t.Errorf("关键字过滤不符: %+v", qr)
	}
}

func TestQueryNameCompat(t *testing.T) {
	handler, storage := newTestHandler()
	storage.put("alice", "app-v1.zip", []byte("1"))
	storage.put("alice", "other.zip", []byte("2"))

	rr := httptest.NewRecorder()
	handler.handleQuery(rr, httptest.NewRequest(http.MethodGet, "/query?username=alice&name=app", nil))

	if qr := decodeQuery(t, rr.Body.String()); qr.Count != 1 {
		t.Errorf("旧参数 name 应作为关键字，实际 %d", qr.Count)
	}
}

func TestQueryMissingUsername(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleQuery(rr, httptest.NewRequest(http.MethodGet, "/query", nil))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestQueryEmptyResult(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleQuery(rr, httptest.NewRequest(http.MethodGet, "/query?username=nobody", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	if qr := decodeQuery(t, rr.Body.String()); qr.Count != 0 {
		t.Errorf("期望 0 个文件，实际 %d", qr.Count)
	}
}

func TestQueryInvalidMethod(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleQuery(rr, httptest.NewRequest(http.MethodPost, "/query", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", rr.Code)
	}
}