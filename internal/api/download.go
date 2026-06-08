package api

import (
	"log-service/internal/model"
	"log-service/internal/service"
	"net/http"
)

func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"code":405,"message":"方法不允许"}`, http.StatusMethodNotAllowed)
		return
	}
	
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		writeJSON(w, http.StatusBadRequest, model.Response{
			Code:    400,
			Message: "缺少 filename 参数",
		})
		return
	}
	
	if err := service.ValidateFilename(filename); err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{
			Code:    400,
			Message: err.Error(),
		})
		return
	}
	
	reader, size, err := h.storage.Get(filename)
	if err != nil {
		writeJSON(w, http.StatusNotFound, model.Response{
			Code:    404,
			Message: "文件不存在",
		})
		return
	}
	defer reader.Close()
	
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Length", string(rune(size)))
	
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}
