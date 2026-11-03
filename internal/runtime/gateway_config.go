package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"maas-gateway-dev-console/internal/config"
)

func WriteGatewayConfig(runtimeRoot string, profile config.Profile) (string, error) {
	configDir := filepath.Join(runtimeRoot, "gateway-config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", fmt.Errorf("create runtime config directory: %w", err)
	}
	path := filepath.Join(configDir, "config."+profile.Gateway.Environment+".yml")
	g := profile.Gateway
	content := fmt.Sprintf(`HttpServer:
  WebApi:
    Port: %s
    ShutdownTimeout: "5s"
    ShutdownDelay: "0s"

RemoteServiceInfoConfig:
  MaasSVCAddr: %s
  MaasInternalSVCAddr: %s
  PlatformSVCAddr: %s
  PlatformInternalSVCAddr: %s
  PlatformMaasInternalSVCAddr: %s
  PlatformBaseSVCAddr: %s

MaaSConfig:
  Token: %s

Redis:
  Host: %s
  Port: %d
  PoolSize: %d

LLMDefaultServiceLevel: %s

Metrics:
  Endpoint:
    Port: %s
  Subsystem: "maas_gateway"
  TimeFirstStreamChunkSeconds:
    Buckets: [0.001, 0.01, 0.1, 1]
  TimePerStreamChunkSeconds:
    Buckets: [0.001, 0.01, 0.1, 1]
  OutputStreamTokensPerSecond:
    Buckets: [1, 10, 100]

MemoryLimitConfig:
  MaxRequestKB: 8192
  LargeRequestKB: 1024
  MaxMemoryUsageMBAllowingLargeRequest: 1024
  BodyLoggingMaxSizeKB: 1024
`, quote(fmt.Sprint(g.HTTPPort)), quote(g.ExternalService.MaaS), quote(g.ExternalService.MaaSInternal),
		quote(g.ExternalService.Platform), quote(g.ExternalService.PlatformInternal),
		quote(g.ExternalService.PlatformMaaSInternal), quote(g.ExternalService.PlatformBase),
		quote(g.MaaSToken), quote(g.Redis.Host), g.Redis.Port, g.Redis.PoolSize,
		quote(g.DefaultTier), quote(fmt.Sprint(g.MetricsPort)))
	if g.AdditionalYAML != "" {
		content += "\n# Additional configuration from the active Web profile.\n" + g.AdditionalYAML
		if content[len(content)-1] != '\n' {
			content += "\n"
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write gateway config: %w", err)
	}
	return configDir, nil
}

func quote(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}
