package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/lum1n/devdash/internal/core"
)

// Server is the JSON API consumed by the TanStack Start app and CLI.
type Server struct {
	App *core.App
}

// Handler returns the API mux.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/overview", s.overview)
	mux.HandleFunc("GET /api/repos/{id}", s.repo)
	mux.HandleFunc("POST /api/repos/{id}/open", s.open)
	mux.HandleFunc("POST /api/repos/{id}/focus", s.focus)
	mux.HandleFunc("POST /api/repos/{id}/next", s.next)
	mux.HandleFunc("GET /api/repos/{id}/diff/{ref}", s.diff)
	mux.HandleFunc("GET /api/repos/{id}/notes", s.notes)
	mux.HandleFunc("POST /api/repos/{id}/notes", s.createNote)
	mux.HandleFunc("GET /api/repos/{id}/notes/{name}", s.readNote)
	mux.HandleFunc("PUT /api/repos/{id}/notes/{name}", s.writeNote)
	mux.HandleFunc("DELETE /api/repos/{id}/notes/{name}", s.deleteNote)
	mux.HandleFunc("POST /api/plugins/{id}/run", s.pluginRun)
	mux.HandleFunc("POST /api/scan", s.scan)
	mux.HandleFunc("POST /api/roots", s.addRoot)
	return withCORS(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "devdash",
		"time":    time.Now().UTC(),
	})
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	ov, err := s.App.Overview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

func (s *Server) scan(w http.ResponseWriter, r *http.Request) {
	if err := s.App.Scan(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	ov, err := s.App.Overview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

func (s *Server) repo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, err := s.App.Detail(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) open(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Action == "" {
		writeError(w, http.StatusBadRequest, errOr("action is required"))
		return
	}
	res, err := s.App.Open(r.Context(), id, body.Action)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) focus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Action string `json:"action"`
		Hours  int    `json:"hours,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Action == "" {
		writeError(w, http.StatusBadRequest, errOr("action is required"))
		return
	}
	if err := s.App.SetFocus(id, body.Action, body.Hours); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ov, err := s.App.Overview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

func (s *Server) next(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errOr("text is required"))
		return
	}
	if err := s.App.SetNext(id, body.Text); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	ov, err := s.App.Overview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

func (s *Server) diff(w http.ResponseWriter, r *http.Request) {
	d, err := s.App.Diff(r.Context(), r.PathValue("id"), r.PathValue("ref"), r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) notes(w http.ResponseWriter, r *http.Request) {
	list, err := s.App.ListNotes(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errOr("title is required"))
		return
	}
	n, err := s.App.CreateNote(r.PathValue("id"), body.Title, body.Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) readNote(w http.ResponseWriter, r *http.Request) {
	n, err := s.App.ReadNote(r.PathValue("id"), r.PathValue("name"))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) writeNote(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, errOr("content is required"))
		return
	}
	n, err := s.App.WriteNote(r.PathValue("id"), r.PathValue("name"), body.Content)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
	if err := s.App.DeleteNote(r.PathValue("id"), r.PathValue("name")); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) pluginRun(w http.ResponseWriter, r *http.Request) {
	pluginID := r.PathValue("id")
	var body struct {
		Action  string `json:"action"`
		Repo    string `json:"repo"`
		Harness string `json:"harness,omitempty"`
		Session string `json:"session,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Action == "" || body.Repo == "" {
		writeError(w, http.StatusBadRequest, errOr("action and repo are required"))
		return
	}
	res, err := s.App.RunPlugin(r.Context(), pluginID, body.Action, body.Repo, map[string]string{
		"harness": body.Harness,
		"session": body.Session,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) addRoot(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		writeError(w, http.StatusBadRequest, errOr("path is required"))
		return
	}
	if err := s.App.AddRoot(body.Path); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.App.Scan(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	ov, err := s.App.Overview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, ov)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	msg := "request failed"
	if err != nil {
		msg = err.Error()
	}
	writeJSON(w, status, map[string]string{"error": msg})
}

func errOr(msg string) error {
	return &plainError{msg}
}

type plainError struct{ s string }

func (e *plainError) Error() string { return e.s }

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://127.0.0.1:3000" || origin == "http://localhost:3000" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
