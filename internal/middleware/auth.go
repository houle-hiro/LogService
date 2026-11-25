package middleware

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type contextKey string

// UsernameKey 上下文中保存当前用户名的键
const UsernameKey contextKey = "username"

// Auth 鉴权中间件：接受 X-Token 头、token 查询参数或签名会话 cookie
type Auth struct {
	enabled bool
	token   string
	session *Session
}

// NewAuth 创建鉴权中间件
func NewAuth(enabled bool, token string, session *Session) *Auth {
	return &Auth{enabled: enabled, token: token, session: session}
}

// UsernameFrom 从上下文取用户名
func UsernameFrom(ctx context.Context) string {
	if name, ok := ctx.Value(UsernameKey).(string); ok {
		return name
	}
	return ""
}

// WithUsername 把用户名放入上下文
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, UsernameKey, username)
}

// Handler 鉴权入口
func (a *Auth) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 健康检查与登录页不需要登录
		if r.URL.Path == "/healthz" || r.URL.Path == "/login" {
			next.ServeHTTP(w, r)
			return
		}

		if !a.enabled {
			// 关闭鉴权时仍尽量带上会话用户
			if username, ok := a.session.FromRequest(r); ok {
				r = r.WithContext(WithUsername(r.Context(), username))
			}
			next.ServeHTTP(w, r)
			return
		}

		if token := extractToken(r); token != "" && a.matchToken(token) {
			next.ServeHTTP(w, r)
			return
		}

		if username, ok := a.session.FromRequest(r); ok {
			next.ServeHTTP(w, r.WithContext(WithUsername(r.Context(), username)))
			return
		}

		if wantsHTML(r) {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    401,
			"message": "未登录或 token 无效，请在请求头带 X-Token",
		})
	})
}

// VerifyToken 供登录页校验口令，鉴权关闭时视为通过
func (a *Auth) VerifyToken(token string) bool {
	if !a.enabled {
		return true
	}
	return a.matchToken(token)
}

func (a *Auth) matchToken(token string) bool {
	if a.token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) == 1
}

func extractToken(r *http.Request) string {
	if token := r.Header.Get("X-Token"); token != "" {
		return token
	}
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
	}
	return r.URL.Query().Get("token")
}

// wantsHTML 浏览器直接访问页面时跳登录页，接口调用返回 JSON
func wantsHTML(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	return strings.Contains(r.Header.Get("Accept"), "text/html")
}