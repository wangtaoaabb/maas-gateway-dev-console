package config

import "encoding/json"

const CurrentVersion = 1

type Profile struct {
	Version int           `json:"version"`
	Name    string        `json:"name"`
	Console ConsoleConfig `json:"console"`
	Mock    MockConfig    `json:"mock"`
	Gateway GatewayConfig `json:"gateway"`
}

type ConsoleConfig struct {
	ListenAddress string `json:"listen_address"`
}

type MockConfig struct {
	ListenAddress string        `json:"listen_address"`
	RequestLimit  int           `json:"request_limit"`
	Routes        []RouteConfig `json:"routes"`
}

type RouteConfig struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Enabled     bool           `json:"enabled"`
	Priority    int            `json:"priority"`
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	Match       MatchConfig    `json:"match,omitempty"`
	Response    ResponseConfig `json:"response"`
}

type MatchConfig struct {
	Headers map[string]string `json:"headers,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	JSON    map[string]any    `json:"json,omitempty"`
}

type ResponseConfig struct {
	Status           int               `json:"status"`
	Headers          map[string]string `json:"headers,omitempty"`
	Body             string            `json:"body,omitempty"`
	SSE              []SSEEvent        `json:"sse,omitempty"`
	DelayMS          int               `json:"delay_ms,omitempty"`
	CloseAfterEvents int               `json:"close_after_events,omitempty"`
}

type SSEEvent struct {
	Data    string `json:"data"`
	DelayMS int    `json:"delay_ms,omitempty"`
}

type GatewayConfig struct {
	SourceDir       string                 `json:"source_dir"`
	Environment     string                 `json:"environment"`
	HTTPPort        int                    `json:"http_port"`
	MetricsPort     int                    `json:"metrics_port"`
	Redis           RedisConfig            `json:"redis"`
	Delve           DelveConfig            `json:"delve"`
	MaaSToken       string                 `json:"maas_token"`
	DefaultTier     string                 `json:"default_service_level"`
	ExternalService ExternalServicesConfig `json:"external_services"`
	AdditionalYAML  string                 `json:"additional_yaml,omitempty"`
}

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	PoolSize int    `json:"pool_size"`
}

type DelveConfig struct {
	Binary         string `json:"binary"`
	ListenAddress  string `json:"listen_address"`
	Continue       bool   `json:"continue_immediately"`
	CheckGoVersion bool   `json:"check_go_version"`
}

type ExternalServicesConfig struct {
	MaaS                 string `json:"maas"`
	MaaSInternal         string `json:"maas_internal"`
	Platform             string `json:"platform"`
	PlatformInternal     string `json:"platform_internal"`
	PlatformMaaSInternal string `json:"platform_maas_internal"`
	PlatformBase         string `json:"platform_base"`
}

func Clone(profile Profile) (Profile, error) {
	data, err := json.Marshal(profile)
	if err != nil {
		return Profile{}, err
	}
	var cloned Profile
	if err := json.Unmarshal(data, &cloned); err != nil {
		return Profile{}, err
	}
	return cloned, nil
}
