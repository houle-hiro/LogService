package api

import (
	"errors"
	"net/http"
	"strings"

	"log-service/internal/model"
	"log-service/internal/service"
)

// handleUpload 上传文件，任意扩展名，按 username 分目录
func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.GetMaxUploadSize())

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusRequestEntityTooLarge, model.Response{
				Code:    413,
				Message: "文件超过大小限制",
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, model.Response{
			Code:    400,
			Message: "解析上传请求失败: " + err.Error(),
		})
		return
	}

	username := r.FormValue("username")
	if username == "" {
		username = h.currentUsername(r)
	}
	user, err := service.NormalizeUsername(username)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{
			Code:    400,
			Message: "未找到 file 字段: " + err.Error(),
		})
		return
	}
	defer file.Close()

	filename := trimUploadFilename(header.Filename)
	if err := service.ValidateFilename(filename); err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	overwritten, size, err := h.storage.Save(user, filename, file)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.Response{
			Code:    500,
			Message: "保存文件失败: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, model.Response{
		Code:    0,
		Message: "上传成功",
		Data: model.UploadResponse{
			Username:    user,
			Name:        filename,
			Size:        size,
			Overwritten: overwritten,
		},
	})
}

// trimUploadFilename 去掉客户端可能带上的目录部分，兼容 Windows 路径
func trimUploadFilename(name string) string {
	if idx := strings.LastIndexAny(name, `/\`); idx >= 0 {
		name = name[idx+1:]
	}
	return name
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	jsonEncode(w, v)
}