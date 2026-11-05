package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"supervision-system/internal/repository"
	"supervision-system/internal/service"
)

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if e := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); e != nil {
		fail(w, 400, "请求数据格式错误")
		return false
	}
	return true
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	write(w, status, map[string]string{"error": msg})
}
func respondError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, service.ErrUnauthorized):
		fail(w, 401, "登录已失效或账号已停用")
	case errors.Is(e, service.ErrForbidden):
		fail(w, 403, "没有操作权限")
	case errors.Is(e, service.ErrInvalid):
		fail(w, 400, "请求参数不完整或不合法")
	case errors.Is(e, repository.ErrNotFound):
		fail(w, 404, "数据不存在")
	case errors.Is(e, repository.ErrConflict):
		fail(w, 409, "数据已存在")
	default:
		fail(w, 500, "服务器处理失败")
	}
}
func pathID(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}
