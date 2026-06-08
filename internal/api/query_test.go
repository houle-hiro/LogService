package api

import (
	"encoding/json"
	"log-service/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleQuery(t *testing.T) {
	handler, storage := newTestHandler()

	// 预先保存文件
	storage.files["app1_abc111_20240101120000.zip"] = []byte("content1")
	storage.files["app1_abc222_20240101130000.zip"] = []byte("content2")
	storage.files["app2_abc111_20240101140000.zip"] = []byte("content3")

	// 测试按名称查询
	req := httptest.NewRequest("GET", "/query?name=app1", nil)
	rr := httptest.NewRecorder()

	handler.handleQuery(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态 200，实际 %d, 响应: %s", rr.Code, rr.Body.String())
	}

	var resp model.Response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	queryResp, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("响应数据类型错误")
	}

	if queryResp["count"].(float64) != 2 {
		t.Errorf("期望返回 2 个文件，实际 %v", queryResp["count"])
	}
}

func TestHandleQueryWithCommit(t *testing.T) {
	handler, storage := newTestHandler()

	storage.files["app1_abc111_20240101120000.zip"] = []byte("content1")
	storage.files["app1_abc222_20240101130000.zip"] = []byte("content2")
	storage.files["app2_abc111_20240101140000.zip"] = []byte("content3")

	// 按名称和 commit 查询
	req := httptest.NewRequest("GET", "/query?name=app1&commit=abc111", nil)
	rr := httptest.NewRecorder()

	handler.handleQuery(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态 200，实际 %d", rr.Code)
	}

	var resp model.Response
	json.Unmarshal(rr.Body.Bytes(), &resp)

	queryResp := resp.Data.(map[string]interface{})
	if queryResp["count"].(float64) != 1 {
		t.Errorf("期望返回 1 个文件，实际 %v", queryResp["count"])
	}
}

func TestHandleQueryInvalidMethod(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("POST", "/query", nil)
	rr := httptest.NewRecorder()

	handler.handleQuery(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望状态 405，实际 %d", rr.Code)
	}
}

func TestHandleQueryEmptyResult(t *testing.T) {
	handler, _ := newTestHandler()

	req := httptest.NewRequest("GET", "/query?name=nonexistent", nil)
	rr := httptest.NewRecorder()

	handler.handleQuery(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望状态 200，实际 %d", rr.Code)
	}

	var resp model.Response
	json.Unmarshal(rr.Body.Bytes(), &resp)

	queryResp := resp.Data.(map[string]interface{})
	if queryResp["count"].(float64) != 0 {
		t.Errorf("期望返回 0 个文件，实际 %v", queryResp["count"])
	}
}
