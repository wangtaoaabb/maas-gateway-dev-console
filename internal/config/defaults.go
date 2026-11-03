package config

const (
	defaultGatewayDir = "${MAAS_GATEWAY_DIR}"
	defaultMockURL    = "http://127.0.0.1:19090"
)

func DefaultProfile() Profile {
	return Profile{
		Version: CurrentVersion,
		Name:    "default",
		Console: ConsoleConfig{ListenAddress: "127.0.0.1:19080"},
		Mock: MockConfig{
			ListenAddress: "127.0.0.1:19090",
			RequestLimit:  500,
			Routes:        defaultRoutes(),
		},
		Gateway: GatewayConfig{
			SourceDir:   defaultGatewayDir,
			Environment: "ci",
			HTTPPort:    9998,
			MetricsPort: 19092,
			Redis: RedisConfig{
				Host:     "127.0.0.1",
				Port:     16380,
				PoolSize: 10,
			},
			Delve: DelveConfig{
				Binary:        "/Users/taowang/go/bin/dlv",
				ListenAddress: "127.0.0.1:2345",
			},
			MaaSToken:   "local-mock",
			DefaultTier: "advanced",
			ExternalService: ExternalServicesConfig{
				MaaS:                 defaultMockURL,
				MaaSInternal:         defaultMockURL,
				Platform:             defaultMockURL,
				PlatformInternal:     defaultMockURL,
				PlatformMaaSInternal: defaultMockURL,
				PlatformBase:         defaultMockURL,
			},
		},
	}
}

func defaultRoutes() []RouteConfig {
	routes := []RouteConfig{
		jsonRoute("router-list", "LLM router list", "POST", "/api/internal/maas/v2/gateway/llm_router/list", `{"code":0,"msg":"","data":{"router_list":[]}}`),
		jsonRoute("dynamic-config", "Gateway dynamic config", "POST", "/api/internal/maas/v2/gateway/dynamic_config", `{"code":0,"msg":"","data":{"maas_gateway_common_config":{"tenant_label_config":{"tenant_label_map":{"tenant-local":"local"},"ignored_tenant_id_list":[]}}}}`),
		jsonRoute("model-list", "MaaS model list", "POST", "/api/internal/maas/v2/model/list", `{"code":0,"msg":"","data":{"model_list":[{"maas_model_id":"model-local-1","name":"mock-model","is_free_llm_service":true,"parameters":{"max_completion_tokens":1024,"is_input_content_spam":false,"is_output_content_spam":false,"endpoint_info_map_v2":{"default:1":{"base_addr":"http://127.0.0.1:19090","url_path_chat":"/mock-engine/v1/chat/completions","real_model_name":"mock-engine-model","weight":1,"extend_info_v2":{"pass_through_headers":["X-Mock-Scenario"]}}}}}],"result_total":1}}`),
		{
			ID: "api-key-local", Name: "API key to tenant", Enabled: true, Priority: 100,
			Method: "POST", Path: "/api/internal/platform/v1/api_key/get_tenant_id",
			Match:    MatchConfig{JSON: map[string]any{"api_key": "local-api-key"}},
			Response: jsonResponse(`{"code":0,"msg":"","data":{"tenant_id":"tenant-local","account_id":"account-local","api_key_id":"api-key-local","enabled":1}}`),
		},
		{
			ID: "api-key-fallback", Name: "Unknown API key", Enabled: true, Priority: 0,
			Method: "POST", Path: "/api/internal/platform/v1/api_key/get_tenant_id",
			Response: jsonResponse(`{"code":0,"msg":"","data":{"tenant_id":"","account_id":"account-local","api_key_id":"api-key-unknown","enabled":1}}`),
		},
		jsonRoute("stop-guard", "Platform stop guard", "POST", "/api/internal/platform-base/v1/stop_guard/check", `{"code":0,"msg":"","data":{"should_next":true,"message":""}}`),
		jsonRoute("usage-limit-all", "API key and tenant usage limits", "POST", "/api/internal/maas/v2/tenant_config/all_usage_limit", `{"code":0,"msg":"","data":{"api_key_usage_limit":{"LLM":{"RPM":0,"RPD":0,"TPM":0,"IPM":0}},"tenant_usage_limit":{"LLM":{"RPM":0,"RPD":0,"TPM":0,"IPM":0}}}}`),
		jsonRoute("model-access", "Model tenant access", "POST", "/api/internal/maas/v2/model_tenant_has_access", `{"code":0,"msg":"","data":{"has_access":true}}`),
		jsonRoute("tenant-config", "Tenant config", "POST", "/api/internal/maas/v2/tenant_config/detail", `{"code":0,"msg":"","data":{"tenant_config_info_list":[]}}`),
		jsonRoute("token-report", "Token count report", "POST", "/api/internal/maas/v2/data_count/receive_token_count", `{"code":0,"msg":"","data":{}}`),
		{
			ID: "chat-stream", Name: "OpenAI chat streaming", Enabled: true, Priority: 100,
			Method: "POST", Path: "/mock-engine/v1/chat/completions",
			Match: MatchConfig{JSON: map[string]any{"stream": true}},
			Response: ResponseConfig{
				Status:  200,
				Headers: map[string]string{"Content-Type": "text/event-stream", "Cache-Control": "no-cache"},
				SSE: []SSEEvent{
					{Data: `{"id":"chatcmpl-local-1","object":"chat.completion.chunk","created":1,"model":"mock-engine-model","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`},
					{Data: `{"id":"chatcmpl-local-1","object":"chat.completion.chunk","created":1,"model":"mock-engine-model","choices":[{"index":0,"delta":{"content":"hello from dev console"},"finish_reason":null}]}`},
					{Data: `{"id":"chatcmpl-local-1","object":"chat.completion.chunk","created":1,"model":"mock-engine-model","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":5,"total_tokens":9}}`},
					{Data: "[DONE]"},
				},
			},
		},
		{
			ID: "chat-json", Name: "OpenAI chat JSON", Enabled: true, Priority: 0,
			Method: "POST", Path: "/mock-engine/v1/chat/completions",
			Response: jsonResponse(`{"id":"chatcmpl-local-1","object":"chat.completion","created":1,"model":"mock-engine-model","choices":[{"index":0,"message":{"role":"assistant","content":"hello from dev console"},"finish_reason":"stop"}],"usage":{"prompt_tokens":4,"completion_tokens":5,"total_tokens":9}}`),
		},
	}
	descriptions := map[string]string{
		"router-list":      "查询 LLM Router 列表，供 Gateway 获取路由信息。",
		"dynamic-config":   "获取 Gateway 动态配置，例如租户标签。",
		"model-list":       "获取模型列表及推理后端地址，用于模型发现和请求转发。",
		"api-key-local":    "校验本地 API Key，返回所属租户、账户和 Key 信息。",
		"api-key-fallback": "模拟未知 API Key，返回空租户信息。",
		"stop-guard":       "查询平台请求放行状态。",
		"usage-limit-all":  "获取 API Key 与租户的请求量、Token 等用量限制。",
		"model-access":     "检查租户是否有权访问指定模型。",
		"tenant-config":    "获取租户级配置。",
		"token-report":     "接收 Gateway 上报的 Token 用量。",
		"chat-stream":      "模拟推理后端，通过 SSE 分块返回 Chat 响应。",
		"chat-json":        "模拟推理后端，返回完整的非流式 Chat JSON 响应。",
	}
	for i := range routes {
		routes[i].Description = descriptions[routes[i].ID]
	}
	return routes
}

func jsonRoute(id, name, method, path, body string) RouteConfig {
	return RouteConfig{ID: id, Name: name, Enabled: true, Method: method, Path: path, Response: jsonResponse(body)}
}

func jsonResponse(body string) ResponseConfig {
	return ResponseConfig{Status: 200, Headers: map[string]string{"Content-Type": "application/json"}, Body: body}
}
