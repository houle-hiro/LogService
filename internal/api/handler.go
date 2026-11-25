package api

import (
	"embed"
	"net/http"

	"log-service/internal/config"
	"log-service/internal/middleware"
	"log-service/internal/service"
)

//go:embed web/*.html
var webFS embed.FS

// Handler HTTP 处理器集合
type Handler struct {
	storage   service.Storage
	cleanup   *service.Cleanup
	rateLimit *middleware.RateLimiter
	auth      *middleware.Auth
	session   *middleware.Session
	cfg       *config.Config
}

// NewHandler 创建处理器
func NewHandler(storage service.Storage, cleanup *service.Cleanup, cfg *config.Config) *Handler {
	session := middleware.NewSession(cfg.GetSessionSecret(), cfg.GetSessionTTL())
	return &Handler{
		storage:   storage,
		cleanup:   cleanup,
		rateLimit: middleware.NewRateLimiter(cfg.RateLimit.Limit, cfg.GetRateLimitInterval()),
		auth:      middleware.NewAuth(cfg.AuthEnabled(), cfg.Auth.Token, session),
		session:   session,
		cfg:       cfg,
	}
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleHome)
	mux.HandleFunc("/login", h.handleLogin)
	mux.HandleFunc("/logout", h.handleLogout)
	mux.HandleFunc("/healthz", h.handleHealth)

	mux.HandleFunc("/upload", h.handleUpload)
	mux.HandleFunc("/download", h.handleDownload)
	mux.HandleFunc("/query", h.handleQuery)
	mux.HandleFunc("/file", h.handleDelete)

	mux.HandleFunc("/api/me", h.handleMe)
}

// Middleware 中间件链，顺序即包裹顺序：第一个在最外层
func (h *Handler) Middleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Recover,
		middleware.Logger,
		h.rateLimit.Handler,
		h.auth.Handler,
	}
}

// currentUsername 取当前登录用户名，query 参数优先，便于 curl 指定
func (h *Handler) currentUsername(r *http.Request) string {
	if name := r.URL.Query().Get("username"); name != "" {
		return name
	}
	return middleware.UsernameFrom(r.Context())
}