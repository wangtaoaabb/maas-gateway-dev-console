package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"maas-gateway-dev-console/internal/config"
	"maas-gateway-dev-console/internal/mock"
	"maas-gateway-dev-console/internal/process"
)

//go:embed static/*
var staticFiles embed.FS

type Server struct {
	profiles *config.Store
	mock     *mock.Server
	process  *process.Manager
	logger   *slog.Logger
	server   *http.Server
}

func NewServer(profiles *config.Store, mockServer *mock.Server, manager *process.Manager, logger *slog.Logger) *Server {
	return &Server{profiles: profiles, mock: mockServer, process: manager, logger: logger}
}

func (s *Server) Start() error {
	assets, err := fs.Sub(staticFiles, "static")
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(assets)))
	mux.HandleFunc("GET /api/profile", s.getProfile)
	mux.HandleFunc("PUT /api/profile", s.putProfile)
	mux.HandleFunc("GET /api/requests", s.getRequests)
	mux.HandleFunc("DELETE /api/requests", s.deleteRequests)
	mux.HandleFunc("GET /api/logs", s.getLogs)
	mux.HandleFunc("DELETE /api/logs", s.deleteLogs)
	mux.HandleFunc("GET /api/runtime", s.getRuntime)
	mux.HandleFunc("POST /api/gateway/start", s.startGateway)
	mux.HandleFunc("POST /api/gateway/stop", s.stopGateway)
	mux.HandleFunc("POST /api/gateway/restart", s.restartGateway)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	profile := s.profiles.Get()
	s.server = &http.Server{Addr: profile.Console.ListenAddress, Handler: requestLogger(s.logger, mux), ReadHeaderTimeout: 5 * time.Second}
	s.logger.Info("web console listening", "url", "http://"+profile.Console.ListenAddress)
	err = s.server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

func (s *Server) getProfile(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.profiles.Get())
}

func (s *Server) putProfile(w http.ResponseWriter, r *http.Request) {
	var profile config.Profile
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024*1024))
	if err := decoder.Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "invalid profile: "+err.Error())
		return
	}
	old := s.profiles.Get()
	if err := s.profiles.Save(profile); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	restartRequired := old.Console.ListenAddress != profile.Console.ListenAddress || old.Mock.ListenAddress != profile.Mock.ListenAddress
	writeJSON(w, http.StatusOK, map[string]any{"profile": profile, "listener_restart_required": restartRequired})
}

func (s *Server) getRequests(w http.ResponseWriter, r *http.Request) {
	after := mock.ParseAfter(r.URL.Query().Get("after"))
	writeJSON(w, http.StatusOK, map[string]any{"requests": s.mock.Recorder().List(after)})
}

func (s *Server) deleteRequests(w http.ResponseWriter, _ *http.Request) {
	s.mock.Recorder().Clear()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) getLogs(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
	writeJSON(w, http.StatusOK, map[string]any{"logs": s.process.Logs().List(after)})
}

func (s *Server) deleteLogs(w http.ResponseWriter, _ *http.Request) {
	s.process.Logs().Clear()
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (s *Server) getRuntime(w http.ResponseWriter, _ *http.Request) {
	profile := s.profiles.Get()
	writeJSON(w, http.StatusOK, map[string]any{
		"gateway": s.process.State(),
		"mock": map[string]any{
			"status": "running",
			"url":    "http://" + profile.Mock.ListenAddress,
		},
	})
}

func (s *Server) startGateway(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid start request")
		return
	}
	if err := s.process.Start(s.profiles.Get(), request.Mode); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, s.process.State())
}

func (s *Server) stopGateway(w http.ResponseWriter, _ *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.process.Stop(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.process.State())
}

func (s *Server) restartGateway(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid restart request")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.process.Stop(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.process.Start(s.profiles.Get(), request.Mode); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, s.process.State())
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		if r.URL.Path != "/api/runtime" && r.URL.Path != "/api/logs" && r.URL.Path != "/api/requests" {
			logger.Debug("console request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(started))
		}
	})
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		fmt.Fprintln(w, `{"error":"failed to encode response"}`)
	}
}
