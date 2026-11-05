package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"supervision-system/internal/domain"
)

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	d, e := s.svc.Dashboard(r.Context(), current(r))
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, d)
}
func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	v, e := s.svc.Tasks(r.Context(), current(r))
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) taskDetail(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "id")
	if e != nil {
		fail(w, 400, "无效编号")
		return
	}
	v, e := s.svc.TaskDetail(r.Context(), current(r), id)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var in domain.TaskInput
	if !decode(w, r, &in) {
		return
	}
	v, e := s.svc.CreateTask(r.Context(), current(r), in)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 201, v)
}
func (s *Server) updateTask(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "id")
	if e != nil {
		fail(w, 400, "无效编号")
		return
	}
	var in domain.TaskInput
	if !decode(w, r, &in) {
		return
	}
	v, e := s.svc.UpdateTask(r.Context(), current(r), id, in)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) createStage(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "id")
	if e != nil {
		fail(w, 400, "无效任务编号")
		return
	}
	var in domain.StageInput
	if !decode(w, r, &in) {
		return
	}
	v, e := s.svc.CreateStage(r.Context(), current(r), id, in)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 201, v)
}
func (s *Server) taskAction(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "id")
	if e != nil {
		fail(w, 400, "无效编号")
		return
	}
	var in struct {
		Action  string `json:"action"`
		Comment string `json:"comment"`
	}
	if !decode(w, r, &in) {
		return
	}
	if e = s.svc.TaskAction(r.Context(), current(r), id, in.Action, in.Comment); e != nil {
		respondError(w, e)
		return
	}
	v, e := s.svc.TaskDetail(r.Context(), current(r), id)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v.Task)
}
func (s *Server) completeStage(w http.ResponseWriter, r *http.Request) {
	taskID, e := pathID(r, "id")
	if e != nil {
		fail(w, 400, "无效任务编号")
		return
	}
	stageID, e := pathID(r, "stageID")
	if e != nil {
		fail(w, 400, "无效阶段编号")
		return
	}
	// 解析前先限制整个 multipart 请求体。额外预留 1 MiB，用于容纳
	// multipart 头部和文件内容之外的文本字段。
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload+(1<<20))
	if e = r.ParseMultipartForm(s.maxUpload); e != nil {
		fail(w, 400, "附件超过大小限制或表单无效")
		return
	}
	progress, e := strconv.Atoi(r.FormValue("progress"))
	if e != nil {
		fail(w, 400, "进度必须是数字")
		return
	}
	files := r.MultipartForm.File["attachments"]
	if e = s.svc.CompleteStage(r.Context(), current(r), taskID, stageID, r.FormValue("completion"), progress, r.FormValue("completed") == "true", files); e != nil {
		respondError(w, e)
		return
	}
	v, e := s.svc.TaskDetail(r.Context(), current(r), taskID)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) download(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "id")
	if e != nil {
		fail(w, 400, "无效附件编号")
		return
	}
	// 服务层会在打开文件前重新校验用户对附件所属任务的访问权限。
	a, f, e := s.svc.Attachment(r.Context(), current(r), id)
	if e != nil {
		respondError(w, e)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", a.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(a.OriginalName)))
	w.Header().Set("Content-Length", strconv.FormatInt(a.Size, 10))
	_, _ = io.Copy(w, f)
}
