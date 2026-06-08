package api

import (
	"log-service/internal/config"
	"log-service/internal/middleware"
	"log-service/internal/service"
	"net/http"
)

type Handler struct {
	storage   service.Storage
	cleanup   *service.Cleanup
	rateLimit *middleware.RateLimiter
	cfg       *config.Config
}

func NewHandler(storage service.Storage, cleanup *service.Cleanup, cfg *config.Config) *Handler {
	rl := middleware.NewRateLimiter(cfg.RateLimit.Limit, cfg.GetRateLimitInterval())
	return &Handler{
		storage:   storage,
		cleanup:   cleanup,
		rateLimit: rl,
		cfg:       cfg,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/upload", h.handleUpload)
	mux.HandleFunc("/download", h.handleDownload)
	mux.HandleFunc("/query", h.handleQuery)
}

func (h *Handler) Middleware() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		middleware.Recover,
		middleware.Logger,
		h.rateLimit.Handler,
	}
}
