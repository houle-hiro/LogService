package api

import (
	"encoding/json"
	"log-service/internal/model"
	"log-service/internal/service"
	"net/http"
)

func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"code":405,"message":"方法不允许"}`, http.StatusMethodNotAllowed)
		return
	}
	
	// 使用配置中的最大上传大小限制
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.GetMaxUploadSize())
	
	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{
			Code:    400,
			Message: "上传文件为空: " + err.Error(),
		})
		return
	}
	defer file.Close()
	
	filename := header.Filename
	if err := service.ValidateFilename(filename); err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}
	
	if err := h.storage.Save(filename, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: "保存文件失败: " + err.Error(),
		})
		return
	}
	
	writeJSON(w, http.StatusOK, model.Response{
		Code:    0,
		Message: "上传成功",
		Data:    filename,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
