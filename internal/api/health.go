package api

import (
	"net/http"
	"time"

	"log-service/internal/model"
)

// handleHealth 健康检查，部署验证用，不需要登录
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}

	writeJSON(w, http.StatusOK, model.Response{
		Code:    0,
		Message: "ok",
		Data: map[string]interface{}{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		},
	})
}