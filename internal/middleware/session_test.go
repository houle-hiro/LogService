package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSessionIssueParse(t *testing.T) {
	s := NewSession("secret-key", time.Hour)
	value := s.Issue("alice", time.Now())

	username, ok := s.Parse(value, time.Now())
	if !ok || username != "alice" {
		t.Errorf("解析会话失败: %q %v", username, ok)
	}
}

func TestSessionExpired(t *testing.T) {
	s := NewSession("secret-key", time.Hour)
	value := s.Issue("alice", time.Now().Add(-2*time.Hour))

	if _, ok := s.Parse(value, time.Now()); ok {
		t.Error("过期会话应校验失败")
	}
}

func TestSessionTampered(t *testing.T) {
	s := NewSession("secret-key", time.Hour)
	value := s.Issue("alice", time.Now())

	// 改动 payload 的最后一个字符
	tampered := []byte(value)
	tampered[0] = byte('X')

	if _, ok := s.Parse(string(tampered), time.Now()); ok {
		t.Error("篡改后的会话应校验失败")
	}
	if _, ok := s.Parse("bad-format", time.Now()); ok {
		t.Error("非法格式应校验失败")
	}
}

func TestSessionWrongSecret(t *testing.T) {
	issuer := NewSession("secret-a", time.Hour)
	verifier := NewSession("secret-b", time.Hour)

	if _, ok := verifier.Parse(issuer.Issue("alice", time.Now()), time.Now()); ok {
		t.Error("不同密钥签发的会话应校验失败")
	}
}

func TestSessionCookieRoundTrip(t *testing.T) {
	s := NewSession("secret-key", time.Hour)
	rr := httptest.NewRecorder()
	s.SetCookie(rr, "bob")

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("期望 1 个 cookie，实际 %d", len(cookies))
	}
	if !cookies[0].HttpOnly {
		t.Error("cookie 应为 HttpOnly")
	}
	if cookies[0].SameSite != http.SameSiteLaxMode {
		t.Error("cookie 应为 SameSite=Lax")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookies[0])
	username, ok := s.FromRequest(req)
	if !ok || username != "bob" {
		t.Errorf("从请求解析用户失败: %q %v", username, ok)
	}
}

func TestSessionClearCookie(t *testing.T) {
	s := NewSession("secret-key", time.Hour)
	rr := httptest.NewRecorder()
	s.ClearCookie(rr)

	cookies := rr.Result().Cookies()
	if len(cookies) != 1 || cookies[0].MaxAge >= 0 {
		t.Error("清除 cookie 应设置 MaxAge < 0")
	}
	if !strings.Contains(cookies[0].Value, "") {
		t.Error("清除时值应为空")
	}
}