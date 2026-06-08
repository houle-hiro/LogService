package api

import (
	"log-service/internal/model"
	"log-service/internal/service"
	"net/http"
)

func (h *Handler) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"code":405,"message":"方法不允许"}`, http.StatusMethodNotAllowed)
		return
	}
	
	name := r.URL.Query().Get("name")
	commit := r.URL.Query().Get("commit")
	
	prefix := service.BuildPrefix(name, commit)
	
	files, err := h.storage.Query(prefix)
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
			Files: files,
			Count: len(files),
		},
	})
}
