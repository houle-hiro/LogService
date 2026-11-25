package api

import (
	"errors"
	"net/http"

	"log-service/internal/model"
	"log-service/internal/service"
)

// handleDelete 删除当前用户的单个文件
func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: "缺少 filename 参数"})
		return
	}

	user, err := service.NormalizeUsername(h.currentUsername(r))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	if err := h.storage.Delete(user, filename); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, model.Response{Code: 404, Message: "文件不存在"})
			return
		}
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, model.Response{Code: 0, Message: "删除成功"})
}