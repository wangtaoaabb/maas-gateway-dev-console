package mock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"maas-gateway-dev-console/internal/config"
)

const maxRecordedBody = 1024 * 1024

type ProfileSource interface {
	Get() config.Profile
}

type Server struct {
	profiles ProfileSource
	recorder *Recorder
	logger   *slog.Logger
	server   *http.Server
}

func NewServer(profiles ProfileSource, logger *slog.Logger) *Server {
	recorder := NewRecorder(func() int { return profiles.Get().Mock.RequestLimit })
	return &Server{profiles: profiles, recorder: recorder, logger: logger}
}

func (s *Server) Recorder() *Recorder { return s.recorder }

func (s *Server) Start() error {
	profile := s.profiles.Get()
	s.server = &http.Server{
		Addr:              profile.Mock.ListenAddress,
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.logger.Info("mock server listening", "address", profile.Mock.ListenAddress)
	err := s.server.ListenAndServe()
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "maas-gateway-dev-console-mock"})
		return
	}

	started := time.Now()
	body, truncated, err := readBody(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "failed to read request body"})
		return
	}
	route, matched := matchRoute(s.profiles.Get().Mock.Routes, r, body)
	status := http.StatusNotImplemented
	if matched {
		status = route.Response.Status
		s.writeConfiguredResponse(w, route.Response)
	} else {
		writeJSON(w, status, map[string]any{
			"error":  "unconfigured external interface",
			"method": r.Method,
			"path":   r.URL.Path,
		})
	}
	record := RequestRecord{
		Time:      started.UTC(),
		Method:    r.Method,
		Path:      r.URL.Path,
		Query:     r.URL.RawQuery,
		Headers:   r.Header.Clone(),
		Body:      string(body),
		Matched:   matched,
		Status:    status,
		Duration:  time.Since(started).Milliseconds(),
		Truncated: truncated,
	}
	if matched {
		record.RouteID = route.ID
	}
	s.recorder.Add(record)
	s.logger.Info("mock request", "method", r.Method, "path", r.URL.Path, "status", status, "route", record.RouteID, "matched", matched)
}

func readBody(body io.ReadCloser) ([]byte, bool, error) {
	if body == nil {
		return nil, false, nil
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, maxRecordedBody+1))
	if err != nil {
		return nil, false, err
	}
	if len(data) > maxRecordedBody {
		return data[:maxRecordedBody], true, nil
	}
	return data, false, nil
}

func matchRoute(routes []config.RouteConfig, request *http.Request, body []byte) (config.RouteConfig, bool) {
	candidates := append([]config.RouteConfig(nil), routes...)
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Priority > candidates[j].Priority })
	var jsonBody map[string]any
	for _, route := range candidates {
		if !route.Enabled || !strings.EqualFold(route.Method, request.Method) || route.Path != request.URL.Path {
			continue
		}
		if !matchStringMap(route.Match.Headers, request.Header.Get) || !matchStringMap(route.Match.Query, request.URL.Query().Get) {
			continue
		}
		if len(route.Match.JSON) > 0 {
			if jsonBody == nil && json.Unmarshal(body, &jsonBody) != nil {
				continue
			}
			if !matchJSONFields(route.Match.JSON, jsonBody) {
				continue
			}
		}
		return route, true
	}
	return config.RouteConfig{}, false
}

func matchStringMap(expected map[string]string, getter func(string) string) bool {
	for key, value := range expected {
		if getter(key) != value {
			return false
		}
	}
	return true
}

func matchJSONFields(expected map[string]any, actual map[string]any) bool {
	for path, expectedValue := range expected {
		actualValue, ok := jsonPath(actual, path)
		if !ok || !jsonValuesEqual(expectedValue, actualValue) {
			return false
		}
	}
	return true
}

func jsonPath(value map[string]any, path string) (any, bool) {
	var current any = value
	for _, component := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[component]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func jsonValuesEqual(left, right any) bool {
	a, errA := json.Marshal(left)
	b, errB := json.Marshal(right)
	return errA == nil && errB == nil && bytes.Equal(a, b)
}

func (s *Server) writeConfiguredResponse(w http.ResponseWriter, response config.ResponseConfig) {
	if response.DelayMS > 0 {
		time.Sleep(time.Duration(response.DelayMS) * time.Millisecond)
	}
	for key, value := range response.Headers {
		w.Header().Set(key, value)
	}
	if len(response.SSE) > 0 {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "text/event-stream")
		}
		w.WriteHeader(response.Status)
		flusher, _ := w.(http.Flusher)
		for index, event := range response.SSE {
			if response.CloseAfterEvents > 0 && index >= response.CloseAfterEvents {
				return
			}
			if event.DelayMS > 0 {
				time.Sleep(time.Duration(event.DelayMS) * time.Millisecond)
			}
			for _, line := range strings.Split(event.Data, "\n") {
				_, _ = fmt.Fprintf(w, "data: %s\n", line)
			}
			_, _ = io.WriteString(w, "\n")
			if flusher != nil {
				flusher.Flush()
			}
		}
		return
	}
	if w.Header().Get("Content-Type") == "" && response.Body != "" {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(response.Status)
	_, _ = io.WriteString(w, response.Body)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func ParseAfter(value string) uint64 {
	after, _ := strconv.ParseUint(value, 10, 64)
	return after
}
