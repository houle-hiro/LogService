package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CookieName 会话 cookie 名称
const CookieName = "fs_session"

// Session 基于 HMAC-SHA256 的签名会话
type Session struct {
	secret []byte
	ttl    time.Duration
}

// NewSession 创建会话签名器
func NewSession(secret string, ttl time.Duration) *Session {
	return &Session{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Issue 生成会话值，格式为 base64(username|exp).base64(sig)
func (s *Session) Issue(username string, now time.Time) string {
	exp := now.Add(s.ttl).Unix()
	payload := fmt.Sprintf("%s|%d", username, exp)
	sig := s.sign(payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
}

// Parse 校验并解析会话值
func (s *Session) Parse(value string, now time.Time) (string, bool) {
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", false
	}
	payload := string(raw)
	if !hmac.Equal([]byte(s.sign(payload)), []byte(parts[1])) {
		return "", false
	}

	sep := strings.LastIndex(payload, "|")
	if sep <= 0 {
		return "", false
	}
	username := payload[:sep]
	exp, err := strconv.ParseInt(payload[sep+1:], 10, 64)
	if err != nil || now.Unix() > exp {
		return "", false
	}
	return username, true
}

func (s *Session) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// SetCookie 写入会话 cookie
func (s *Session) SetCookie(w http.ResponseWriter, username string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    s.Issue(username, time.Now()),
		Path:     "/",
		MaxAge:   int(s.ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearCookie 清除会话 cookie
func (s *Session) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// FromRequest 从请求 cookie 中读取用户名
func (s *Session) FromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return "", false
	}
	return s.Parse(cookie.Value, time.Now())
}