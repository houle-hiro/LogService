package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"log-service/internal/model"
	"log-service/internal/service"
)

func jsonEncode(w http.ResponseWriter, v interface{}) {
	_ = json.NewEncoder(w).Encode(v)
}

// handleDownload 下载文件，按扩展名推断 Content-Type
func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: "缺少 filename 参数"})
		return
	}
	if err := service.ValidateFilename(filename); err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	user, err := service.NormalizeUsername(h.currentUsername(r))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.Response{Code: 400, Message: err.Error()})
		return
	}

	reader, size, err := h.storage.Get(user, filename)
	if err != nil {
		writeJSON(w, http.StatusNotFound, model.Response{Code: 404, Message: "文件不存在"})
		return
	}
	defer reader.Close()

	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("Content-Disposition", buildContentDisposition(filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	if _, err := io.Copy(w, reader); err != nil {
		// 头已发出，只能记日志
		return
	}
}

// buildContentDisposition 同时给 ASCII 回退名和 RFC 5987 的 UTF-8 名
func buildContentDisposition(filename string) string {
	ascii := make([]rune, 0, len(filename))
	for _, r := range filename {
		if r > 0x20 && r < 0x7f && r != '"' && r != '\\' && r != '/' {
			ascii = append(ascii, r)
		} else {
			ascii = append(ascii, '_')
		}
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`,
		string(ascii), encodeRFC5987(filename))
}

// encodeRFC5987 按字节做 percent-encoding
func encodeRFC5987(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			strings.IndexByte("!#$&+-.^_`|~", c) >= 0 {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}