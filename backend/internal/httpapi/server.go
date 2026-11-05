// Package httpapi 将 HTTP 请求适配为应用服务调用，只负责路由、
// 鉴权上下文、请求限制和 JSON 等传输层事务。
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"supervision-system/internal/domain"
	"supervision-system/internal/service"
)

type contextKey string

const userKey contextKey = "user"

// Server 将 HTTP 处理器连接到服务层和静态前端资源。
type Server struct {
	svc       *service.Service
	log       *slog.Logger
	frontend  string
	maxUpload int64
}

func New(svc *service.Service, log *slog.Logger, frontend string, maxUpload int64) *Server {
	return &Server{svc: svc, log: log, frontend: frontend, maxUpload: maxUpload}
}

// Handler 返回已经应用公共中间件的完整应用路由。
func (s *Server) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("POST /api/login", s.login)
	m.HandleFunc("POST /api/logout", s.auth(s.logout))
	m.HandleFunc("GET /api/me", s.auth(s.me))
	m.HandleFunc("GET /api/dashboard", s.auth(s.dashboard))
	m.HandleFunc("GET /api/tasks", s.auth(s.tasks))
	m.HandleFunc("GET /api/tasks/{id}", s.auth(s.taskDetail))
	m.HandleFunc("POST /api/tasks", s.auth(s.createTask))
	m.HandleFunc("PUT /api/tasks/{id}", s.auth(s.updateTask))
	m.HandleFunc("POST /api/tasks/{id}/stages", s.auth(s.createStage))
	m.HandleFunc("POST /api/tasks/{id}/action", s.auth(s.taskAction))
	m.HandleFunc("POST /api/tasks/{id}/stages/{stageID}/complete", s.auth(s.completeStage))
	m.HandleFunc("GET /api/attachments/{id}/download", s.auth(s.download))
	m.HandleFunc("GET /api/departments", s.auth(s.departments))
	m.HandleFunc("GET /api/users", s.auth(s.users))
	m.HandleFunc("POST /api/users", s.auth(s.createUser))
	m.HandleFunc("PUT /api/users/{id}", s.auth(s.updateUser))
	m.HandleFunc("GET /api/audits", s.auth(s.audits))
	m.Handle("/", http.FileServer(http.Dir(s.frontend)))
	return s.middleware(m)
}

type handler func(http.ResponseWriter, *http.Request)

func (s *Server) auth(next handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)
		if token == "" {
			fail(w, http.StatusUnauthorized, "请先登录")
			return
		}
		// 每个受保护请求都重新解析会话，使 MySQL 中的账号停用和会话过期
		// 能够立即生效。
		u, e := s.svc.Session(r.Context(), token)
		if e != nil {
			respondError(w, e)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	}
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// 这些默认安全响应头同时保护 JSON 响应和随服务发布的前端资源。
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		ww := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(ww, r)
		s.log.Info("http request", "method", r.Method, "path", r.URL.Path, "status", ww.status, "duration", time.Since(start))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(n int) { w.status = n; w.ResponseWriter.WriteHeader(n) }
func current(r *http.Request) domain.User { return r.Context().Value(userKey).(domain.User) }
func bearer(r *http.Request) string {
	v := r.Header.Get("Authorization")
	if !strings.HasPrefix(v, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(v, "Bearer ")
}
