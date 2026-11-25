package api

import (
	"net/http"

	"log-service/internal/model"
	"log-service/internal/service"
)

// handleQuery 列出当前用户的文件，keyword 为文件名子串过滤
func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}

	user, err := service.NormalizeUsername(h.currentUsername(r))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		keyword = r.URL.Query().Get("name")
	}

	files, err := h.storage.Query(user, keyword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: "查询失败: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, model.Response{
		Code:    0,
		Message: "查询成功",
		Data: model.QueryResponse{
			Username: user,
			Keyword:  keyword,
			Files:    files,
			Count:    len(files),
		},
	})
}