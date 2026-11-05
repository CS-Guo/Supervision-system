package httpapi

import "net/http"

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	token, u, e := s.svc.Login(r.Context(), in.Username, in.Password)
	if e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, map[string]any{"token": token, "user": u})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if e := s.svc.Logout(r.Context(), bearer(r)); e != nil {
		respondError(w, e)
		return
	}
	write(w, 200, map[string]bool{"ok": true})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) { write(w, 200, current(r)) }
