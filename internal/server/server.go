package server

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pythonsaber2/dockly/internal/auth"
	"github.com/pythonsaber2/dockly/internal/core"
	"github.com/pythonsaber2/dockly/internal/store"
)

//go:embed web/*
var webAssets embed.FS

type Engine interface {
	Deploy(context.Context, core.Application, core.Deployment) (core.Deployment, error)
	StartImage(context.Context, core.Application, string, core.Deployment) (core.Deployment, error)
	Logs(context.Context, string, int) (string, error)
	Status(context.Context, string) (string, error)
	Remove(context.Context, string) error
}

type Config struct {
	SecureCookies bool
	APIToken      string
	Version       string
	EngineReady   bool
	Logger        *slog.Logger
}
type Server struct {
	store    *store.Store
	engine   Engine
	sessions *auth.Sessions
	cfg      Config
	mux      *http.ServeMux
}

func New(s *store.Store, engine Engine, cfg Config) http.Handler {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	x := &Server{store: s, engine: engine, sessions: auth.NewSessions(), cfg: cfg, mux: http.NewServeMux()}
	x.routes()
	return x.securityHeaders(x.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /api/setup/status", s.setupStatus)
	s.mux.HandleFunc("POST /api/setup", s.setup)
	s.mux.HandleFunc("POST /api/login", s.login)
	s.mux.HandleFunc("POST /api/logout", s.logout)
	s.mux.HandleFunc("POST /hooks/{appID}", s.webhook)
	s.mux.Handle("/api/", s.requireAuth(http.HandlerFunc(s.api)))
	assets, _ := fs.Sub(webAssets, "web")
	s.mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets))))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := webAssets.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "UI unavailable", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	})
}

func (s *Server) api(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/"), "/")
	if path == "apps" {
		if r.Method == http.MethodGet {
			s.listApps(w)
			return
		}
		if r.Method == http.MethodPost {
			s.createApp(w, r)
			return
		}
	}
	if path == "deployments" && r.Method == http.MethodGet {
		s.listDeployments(w, "")
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) >= 2 && parts[0] == "apps" {
		id := parts[1]
		if len(parts) == 2 {
			switch r.Method {
			case http.MethodGet:
				s.getApp(w, id)
			case http.MethodPatch:
				s.updateApp(w, r, id)
			case http.MethodDelete:
				s.deleteApp(w, r, id)
			default:
				methodNotAllowed(w)
			}
			return
		}
		action := parts[2]
		switch {
		case action == "deploy" && r.Method == http.MethodPost:
			s.deployApp(w, id, "manual")
		case action == "deployments" && r.Method == http.MethodGet:
			s.listDeployments(w, id)
		case action == "logs" && r.Method == http.MethodGet:
			s.logs(w, r, id)
		case action == "status" && r.Method == http.MethodGet:
			s.status(w, r, id)
		case action == "env" && r.Method == http.MethodPut:
			s.updateEnv(w, r, id)
		case action == "rollback" && r.Method == http.MethodPost && len(parts) == 4:
			s.rollback(w, id, parts[3])
		default:
			http.NotFound(w, r)
		}
		return
	}
	http.NotFound(w, r)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "version": s.cfg.Version, "engineReady": s.cfg.EngineReady})
}
func (s *Server) setupStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]bool{"required": s.store.AdminHash() == ""})
}
func (s *Server) setup(w http.ResponseWriter, r *http.Request) {
	if s.store.AdminHash() != "" {
		writeError(w, 409, "setup already completed")
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if err = s.store.SetAdminHash(hash); err != nil {
		writeError(w, 500, "could not save setup")
		return
	}
	token := core.NewID() + core.NewID()
	_ = s.store.SetWebhookToken(token)
	s.issueSession(w)
	writeJSON(w, 201, map[string]any{"message": "Dockly is ready", "webhookToken": token})
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if s.store.AdminHash() == "" {
		writeError(w, 428, "setup required")
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !auth.VerifyPassword(s.store.AdminHash(), in.Password) {
		time.Sleep(250 * time.Millisecond)
		writeError(w, 401, "invalid password")
		return
	}
	s.issueSession(w)
	writeJSON(w, 200, map[string]string{"message": "signed in"})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("dockly_session"); err == nil {
		s.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "dockly_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: s.cfg.SecureCookies})
	w.WriteHeader(204)
}
func (s *Server) issueSession(w http.ResponseWriter) {
	token, err := s.sessions.Create()
	if err != nil {
		writeError(w, 500, "could not create session")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "dockly_session", Value: token, Path: "/", MaxAge: 86400, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: s.cfg.SecureCookies})
}
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok := false
		if c, err := r.Cookie("dockly_session"); err == nil {
			ok = s.sessions.Valid(c.Value)
		}
		if !ok && s.cfg.APIToken != "" {
			raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			ok = len(raw) == len(s.cfg.APIToken) && subtle.ConstantTimeCompare([]byte(raw), []byte(s.cfg.APIToken)) == 1
		}
		if !ok {
			writeError(w, 401, "authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) listApps(w http.ResponseWriter) {
	apps := s.store.Applications()
	out := make([]core.Application, len(apps))
	for i, a := range apps {
		out[i] = a.Public()
	}
	writeJSON(w, 200, map[string]any{"applications": out})
}
func (s *Server) getApp(w http.ResponseWriter, id string) {
	a, ok := s.store.Application(id)
	if !ok {
		writeError(w, 404, "application not found")
		return
	}
	writeJSON(w, 200, map[string]any{"application": a.Public()})
}

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,47}$`)

func (s *Server) createApp(w http.ResponseWriter, r *http.Request) {
	var a core.Application
	if !decode(w, r, &a) {
		return
	}
	if !namePattern.MatchString(a.Name) {
		writeError(w, 400, "name must contain only letters, numbers, hyphens, or underscores")
		return
	}
	if a.Port < 1 || a.Port > 65535 || a.PublishPort < 1 || a.PublishPort > 65535 {
		writeError(w, 400, "container and published ports must be between 1 and 65535")
		return
	}
	created, err := s.store.CreateApplication(a)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	writeJSON(w, 201, map[string]any{"application": created.Public()})
}
func (s *Server) updateApp(w http.ResponseWriter, r *http.Request, id string) {
	a, ok := s.store.Application(id)
	if !ok {
		writeError(w, 404, "application not found")
		return
	}
	var in struct {
		Name        *string `json:"name"`
		Repository  *string `json:"repository"`
		Branch      *string `json:"branch"`
		Dockerfile  *string `json:"dockerfile"`
		Context     *string `json:"context"`
		Port        *int    `json:"port"`
		PublishPort *int    `json:"publishPort"`
		HealthPath  *string `json:"healthPath"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Name != nil {
		a.Name = *in.Name
	}
	if in.Repository != nil {
		a.Repository = *in.Repository
	}
	if in.Branch != nil {
		a.Branch = *in.Branch
	}
	if in.Dockerfile != nil {
		a.Dockerfile = *in.Dockerfile
	}
	if in.Context != nil {
		a.Context = *in.Context
	}
	if in.Port != nil {
		a.Port = *in.Port
	}
	if in.PublishPort != nil {
		a.PublishPort = *in.PublishPort
	}
	if in.HealthPath != nil {
		a.HealthPath = *in.HealthPath
	}
	if !namePattern.MatchString(a.Name) || a.Port < 1 || a.Port > 65535 || a.PublishPort < 1 || a.PublishPort > 65535 {
		writeError(w, 400, "invalid application settings")
		return
	}
	if err := s.store.UpdateApplication(a); err != nil {
		writeError(w, 500, "could not update application")
		return
	}
	writeJSON(w, 200, map[string]any{"application": a.Public()})
}
func (s *Server) updateEnv(w http.ResponseWriter, r *http.Request, id string) {
	a, ok := s.store.Application(id)
	if !ok {
		writeError(w, 404, "application not found")
		return
	}
	var in struct {
		Env map[string]string `json:"env"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Env == nil {
		in.Env = map[string]string{}
	}
	a.Env = in.Env
	if err := s.store.UpdateApplication(a); err != nil {
		writeError(w, 500, "could not save environment")
		return
	}
	writeJSON(w, 200, map[string]any{"application": a.Public()})
}
func (s *Server) deleteApp(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.store.Application(id); !ok {
		writeError(w, 404, "application not found")
		return
	}
	if s.engine != nil {
		_ = s.engine.Remove(r.Context(), id)
	}
	if err := s.store.DeleteApplication(id); err != nil {
		writeError(w, 500, "could not delete application")
		return
	}
	w.WriteHeader(204)
}
func (s *Server) deployApp(w http.ResponseWriter, id, trigger string) {
	a, ok := s.store.Application(id)
	if !ok {
		writeError(w, 404, "application not found")
		return
	}
	if s.engine == nil {
		writeError(w, 503, "deployment engine unavailable")
		return
	}
	d := core.Deployment{ID: core.NewID(), AppID: id, Status: "queued", Trigger: trigger, StartedAt: time.Now().UTC()}
	_ = s.store.SaveDeployment(d)
	a.Status = "deploying"
	_ = s.store.UpdateApplication(a)
	go s.runDeploy(a, d)
	writeJSON(w, 202, map[string]any{"deployment": d})
}
func (s *Server) runDeploy(a core.Application, d core.Deployment) {
	result, err := s.engine.Deploy(context.Background(), a, d)
	_ = s.store.SaveDeployment(result)
	current, ok := s.store.Application(a.ID)
	if ok {
		if err != nil {
			current.Status = "failed"
		} else {
			current.Status = "running"
		}
		_ = s.store.UpdateApplication(current)
	}
	if err != nil {
		s.cfg.Logger.Error("deployment failed", "app", a.ID, "deployment", d.ID, "error", err)
	}
}
func (s *Server) listDeployments(w http.ResponseWriter, appID string) {
	writeJSON(w, 200, map[string]any{"deployments": s.store.Deployments(appID)})
}
func (s *Server) rollback(w http.ResponseWriter, appID, deploymentID string) {
	a, ok := s.store.Application(appID)
	if !ok {
		writeError(w, 404, "application not found")
		return
	}
	target, ok := s.store.Deployment(deploymentID)
	if !ok || target.AppID != appID || target.Status != "successful" || target.Image == "" {
		writeError(w, 400, "deployment cannot be rolled back")
		return
	}
	d := core.Deployment{ID: core.NewID(), AppID: appID, Status: "queued", Trigger: "rollback", Commit: target.Commit, Image: target.Image, StartedAt: time.Now().UTC()}
	_ = s.store.SaveDeployment(d)
	go func() {
		result, err := s.engine.StartImage(context.Background(), a, target.Image, d)
		_ = s.store.SaveDeployment(result)
		current, _ := s.store.Application(appID)
		if err != nil {
			current.Status = "failed"
		} else {
			current.Status = "running"
		}
		_ = s.store.UpdateApplication(current)
	}()
	writeJSON(w, 202, map[string]any{"deployment": d})
}
func (s *Server) logs(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.store.Application(id); !ok {
		writeError(w, 404, "application not found")
		return
	}
	lines, _ := strconv.Atoi(r.URL.Query().Get("lines"))
	out, err := s.engine.Logs(r.Context(), id, lines)
	if err != nil {
		writeError(w, 502, err.Error())
		return
	}
	writeJSON(w, 200, map[string]string{"logs": out})
}
func (s *Server) status(w http.ResponseWriter, r *http.Request, id string) {
	if _, ok := s.store.Application(id); !ok {
		writeError(w, 404, "application not found")
		return
	}
	status, err := s.engine.Status(r.Context(), id)
	if err != nil {
		writeJSON(w, 200, map[string]string{"status": "stopped"})
		return
	}
	writeJSON(w, 200, map[string]string{"status": status})
}
func (s *Server) webhook(w http.ResponseWriter, r *http.Request) {
	expected := s.store.WebhookToken()
	actual := r.Header.Get("X-Dockly-Token")
	if actual == "" {
		actual = r.URL.Query().Get("token")
	}
	if expected == "" || len(actual) != len(expected) || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		writeError(w, 401, "invalid webhook token")
		return
	}
	s.deployApp(w, r.PathValue("appID"), "webhook")
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, 415, "content type must be application/json")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, 400, "invalid JSON: "+err.Error())
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func methodNotAllowed(w http.ResponseWriter) { writeError(w, 405, "method not allowed") }
func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}
