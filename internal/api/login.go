package api

import (
	"html/template"
	"net/http"
	"strings"

	"log-service/internal/middleware"
	"log-service/internal/model"
	"log-service/internal/service"
)

var templates = template.Must(template.ParseFS(webFS, "web/*.html"))

type loginPageData struct {
	AuthEnabled bool
	Error       string
	Next        string
}

type indexPageData struct {
	Username    string
	AuthEnabled bool
	MaxSizeMB   int
}

// handleHome 首页
func (h *Handler) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}

	username := middleware.UsernameFrom(r.Context())
	if username == "" {
		http.Redirect(w, r, "/login?next=%2F", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = templates.ExecuteTemplate(w, "index.html", indexPageData{
		Username:    username,
		AuthEnabled: h.cfg.AuthEnabled(),
		MaxSizeMB:   h.cfg.Upload.MaxSizeMB,
	})
}

// handleLogin 登录页与登录提交
func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if username, ok := h.session.FromRequest(r); ok {
			http.Redirect(w, r, "/", http.StatusFound)
			_ = username
			return
		}
		h.renderLogin(w, http.StatusOK, loginPageData{
			AuthEnabled: h.cfg.AuthEnabled(),
			Next:        safeNext(r.URL.Query().Get("next")),
		})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			h.renderLogin(w, http.StatusBadRequest, loginPageData{
				AuthEnabled: h.cfg.AuthEnabled(),
				Error:       "表单解析失败",
				Next:        "/",
			})
			return
		}

		next := safeNext(r.FormValue("next"))
		if !h.auth.VerifyToken(r.FormValue("token")) {
			h.renderLogin(w, http.StatusUnauthorized, loginPageData{
				AuthEnabled: h.cfg.AuthEnabled(),
				Error:       "访问口令不正确",
				Next:        next,
			})
			return
		}

		user, err := service.NormalizeUsername(r.FormValue("username"))
		if err != nil {
			h.renderLogin(w, http.StatusBadRequest, loginPageData{
				AuthEnabled: h.cfg.AuthEnabled(),
				Error:       err.Error(),
				Next:        next,
			})
			return
		}

		h.session.SetCookie(w, user)
		http.Redirect(w, r, next, http.StatusFound)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
	}
}

// handleLogout 退出登录
func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}
	h.session.ClearCookie(w)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// handleMe 返回当前登录用户
func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.Response{Code: 405, Message: "方法不允许"})
		return
	}
	writeJSON(w, http.StatusOK, model.Response{
		Code:    0,
		Message: "ok",
		Data:    model.SessionInfo{Username: h.currentUsername(r)},
	})
}

func (h *Handler) renderLogin(w http.ResponseWriter, status int, data loginPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = templates.ExecuteTemplate(w, "login.html", data)
}

// safeNext 只允许站内相对路径，防开放重定向
func safeNext(next string) string {
	if next == "" ||
		!strings.HasPrefix(next, "/") ||
		strings.HasPrefix(next, "//") ||
		strings.Contains(next, "\\") {
		return "/"
	}
	return next
}