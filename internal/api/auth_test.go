package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func nowTime() time.Time { return time.Now() }

func newTestServer(h *Handler) http.Handler {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	var handler http.Handler = mux
	for _, mw := range h.Middleware() {
		handler = mw(handler)
	}
	return handler
}

func TestAuthRequiresToken(t *testing.T) {
	handler, storage := newAuthTestHandler()
	srv := newTestServer(handler)
	storage.put("alice", "a.txt", []byte("x"))

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/query?username=alice", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("无 token 应 401，实际 %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "X-Token") {
		t.Errorf("401 响应应提示 X-Token: %s", rr.Body.String())
	}
}

func TestAuthWithTokenHeader(t *testing.T) {
	handler, storage := newAuthTestHandler()
	srv := newTestServer(handler)
	storage.put("alice", "a.txt", []byte("x"))

	req := httptest.NewRequest(http.MethodGet, "/query?username=alice", nil)
	req.Header.Set("X-Token", "test-token")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("带正确 token 应 200，实际 %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAuthWithTokenQuery(t *testing.T) {
	handler, storage := newAuthTestHandler()
	srv := newTestServer(handler)
	storage.put("alice", "a.txt", []byte("x"))

	req := httptest.NewRequest(http.MethodGet, "/query?username=alice&token=test-token", nil)
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("query token 应 200，实际 %d", rr.Code)
	}
}

func TestHealthNoAuth(t *testing.T) {
	handler, _ := newAuthTestHandler()
	srv := newTestServer(handler)

	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rr.Code != http.StatusOK {
		t.Errorf("健康检查不需要登录，实际 %d", rr.Code)
	}
}

func TestHomeRedirectsToLogin(t *testing.T) {
	handler, _ := newAuthTestHandler()
	srv := newTestServer(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("未登录应 302，实际 %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); !strings.HasPrefix(loc, "/login") {
		t.Errorf("应跳转登录页，实际 %q", loc)
	}
}

func TestLoginThenBrowse(t *testing.T) {
	handler, storage := newAuthTestHandler()
	srv := newTestServer(handler)
	storage.put("alice", "a.txt", []byte("x"))

	// 登录拿 cookie
	form := url.Values{"username": {"alice"}, "token": {"test-token"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRR := httptest.NewRecorder()
	srv.ServeHTTP(loginRR, loginReq)
	if loginRR.Code != http.StatusFound {
		t.Fatalf("登录失败: %d %s", loginRR.Code, loginRR.Body.String())
	}
	cookies := loginRR.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("登录未返回 cookie")
	}

	// 用 cookie 打开首页
	homeReq := httptest.NewRequest(http.MethodGet, "/", nil)
	homeReq.Header.Set("Accept", "text/html")
	homeReq.AddCookie(cookies[0])
	homeRR := httptest.NewRecorder()
	srv.ServeHTTP(homeRR, homeReq)

	if homeRR.Code != http.StatusOK {
		t.Fatalf("首页应 200，实际 %d: %s", homeRR.Code, homeRR.Body.String())
	}
	if !strings.Contains(homeRR.Body.String(), "alice") {
		t.Error("首页应显示用户名")
	}

	// 首页里的接口调用不应再带 username 参数
	queryReq := httptest.NewRequest(http.MethodGet, "/query", nil)
	queryReq.AddCookie(cookies[0])
	queryRR := httptest.NewRecorder()
	srv.ServeHTTP(queryRR, queryReq)

	if queryRR.Code != http.StatusOK {
		t.Fatalf("cookie 查询应 200，实际 %d: %s", queryRR.Code, queryRR.Body.String())
	}
	if !strings.Contains(queryRR.Body.String(), `"count":1`) {
		t.Errorf("应查到 1 个文件: %s", queryRR.Body.String())
	}
}

func TestMeEndpoint(t *testing.T) {
	handler, _ := newAuthTestHandler()
	srv := newTestServer(handler)

	form := url.Values{"username": {"bob"}, "token": {"test-token"}}
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginRR := httptest.NewRecorder()
	srv.ServeHTTP(loginRR, loginReq)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.AddCookie(loginRR.Result().Cookies()[0])
	rr := httptest.NewRecorder()
	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("期望 200，实际 %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"username":"bob"`) {
		t.Errorf("应返回 bob: %s", rr.Body.String())
	}
}