package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestDelete(t *testing.T) {
	handler, storage := newTestHandler()
	storage.put("alice", "a.txt", []byte("x"))

	req := httptest.NewRequest(http.MethodDelete, "/file?username=alice&filename=a.txt", nil)
	rr := httptest.NewRecorder()
	handler.handleDelete(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d: %s", rr.Code, rr.Body.String())
	}
	if _, ok := storage.files["alice"]["a.txt"]; ok {
		t.Error("文件应已删除")
	}
}

func TestDeleteNotFound(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleDelete(rr, httptest.NewRequest(http.MethodDelete, "/file?username=alice&filename=none.txt", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("期望 404，实际 %d", rr.Code)
	}
}

func TestDeleteInvalidMethod(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleDelete(rr, httptest.NewRequest(http.MethodGet, "/file", nil))
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("期望 405，实际 %d", rr.Code)
	}
}

func TestDeleteMissingFilename(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleDelete(rr, httptest.NewRequest(http.MethodDelete, "/file?username=alice", nil))
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestHealth(t *testing.T) {
	handler, _ := newTestHandler()
	rr := httptest.NewRecorder()
	handler.handleHealth(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"ok"`) {
		t.Errorf("响应应包含 ok: %s", rr.Body.String())
	}
}

func TestSafeNext(t *testing.T) {
	cases := map[string]string{
		"":                     "/",
		"/":                    "/",
		"/query?keyword=a":     "/query?keyword=a",
		"//evil.com":           "/",
		"https://evil.com":     "/",
		`/a\b`:                 "/",
	}
	for input, want := range cases {
		if got := safeNext(input); got != want {
			t.Errorf("safeNext(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLoginPage(t *testing.T) {
	handler, _ := newAuthTestHandler()
	rr := httptest.NewRecorder()
	handler.handleLogin(rr, httptest.NewRequest(http.MethodGet, "/login", nil))

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `name="username"`) {
		t.Error("应包含用户名输入框")
	}
	if !strings.Contains(body, `name="token"`) {
		t.Error("鉴权开启时应包含口令输入框")
	}
}

func TestLoginWrongToken(t *testing.T) {
	handler, _ := newAuthTestHandler()
	form := url.Values{"username": {"alice"}, "token": {"wrong"}, "next": {"/"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.handleLogin(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("期望 401，实际 %d", rr.Code)
	}
}

func TestLoginSuccess(t *testing.T) {
	handler, _ := newAuthTestHandler()
	form := url.Values{"username": {"Alice"}, "token": {"test-token"}, "next": {"/"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.handleLogin(rr, req)
	if rr.Code != http.StatusFound {
		t.Fatalf("期望 302，实际 %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Location"); got != "/" {
		t.Errorf("重定向期望 /，实际 %q", got)
	}

	cookies := rr.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("应设置会话 cookie")
	}
	username, ok := handler.session.Parse(cookies[0].Value, nowTime())
	if !ok || username != "alice" {
		t.Errorf("cookie 解析不符: %q %v", username, ok)
	}
}

func TestLoginInvalidUsername(t *testing.T) {
	handler, _ := newAuthTestHandler()
	form := url.Values{"username": {"../root"}, "token": {"test-token"}}
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	handler.handleLogin(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("期望 400，实际 %d", rr.Code)
	}
}

func TestLogout(t *testing.T) {
	handler, _ := newAuthTestHandler()
	rr := httptest.NewRecorder()
	handler.handleLogout(rr, httptest.NewRequest(http.MethodPost, "/logout", nil))

	if rr.Code != http.StatusFound {
		t.Fatalf("期望 302，实际 %d", rr.Code)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].MaxAge >= 0 {
		t.Error("应清除会话 cookie")
	}
}