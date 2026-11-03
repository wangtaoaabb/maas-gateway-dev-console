package mock

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"maas-gateway-dev-console/internal/config"
)

type staticProfiles struct{ profile config.Profile }

func (s staticProfiles) Get() config.Profile { return s.profile }

func TestConfiguredRouteMatchesJSONAndRecordsRequest(t *testing.T) {
	profiles := staticProfiles{profile: config.DefaultProfile()}
	server := NewServer(profiles, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	request := httptest.NewRequest(http.MethodPost, "/api/internal/platform/v1/api_key/get_tenant_id", bytes.NewBufferString(`{"api_key":"local-api-key"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	records := server.Recorder().List(0)
	if len(records) != 1 || records[0].RouteID != "api-key-local" || !records[0].Matched {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestUnconfiguredRouteIsVisibleFailure(t *testing.T) {
	profiles := staticProfiles{profile: config.DefaultProfile()}
	server := NewServer(profiles, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	request := httptest.NewRequest(http.MethodPost, "/not/configured", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNotImplemented)
	}
	records := server.Recorder().List(0)
	if len(records) != 1 || records[0].Matched {
		t.Fatalf("unexpected records: %#v", records)
	}
}

func TestStreamingRouteWritesSSE(t *testing.T) {
	profiles := staticProfiles{profile: config.DefaultProfile()}
	server := NewServer(profiles, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	request := httptest.NewRequest(http.MethodPost, "/mock-engine/v1/chat/completions", bytes.NewBufferString(`{"model":"mock-engine-model","stream":true}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("data: [DONE]")) {
		t.Fatalf("SSE response missing DONE: %s", response.Body.String())
	}
}
