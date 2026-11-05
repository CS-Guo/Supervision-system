package httpapi

import (
	"net/http"
	"supervision-system/internal/domain"
)

func (s *Server) departments(w http.ResponseWriter, r *http.Request) {
	v, e := s.svc.Departments(r.Context())
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) users(w http.ResponseWriter, r *http.Request) {
	v, e := s.svc.Users(r.Context(), current(r))
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var in domain.UserInput
	if !decode(w, r, &in) {
		return
	}
	v, e := s.svc.CreateUser(r.Context(), current(r), in)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 201, v)
}
func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id, e := pathID(r, "id")
	if e != nil {
		fail(w, 400, "无效编号")
		return
	}
	var in domain.UserInput
	if !decode(w, r, &in) {
		return
	}
	v, e := s.svc.UpdateUser(r.Context(), current(r), id, in)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
func (s *Server) audits(w http.ResponseWriter, r *http.Request) {
	v, e := s.svc.Audits(r.Context(), current(r))
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, v)
}
