package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"maas-gateway-dev-console/internal/config"
)

func TestWriteGatewayConfigUsesProfileAddresses(t *testing.T) {
	profile := config.DefaultProfile()
	profile.Gateway.ExternalService.MaaS = "http://127.0.0.1:29999"
	profile.Gateway.AdditionalYAML = "Tracing:\n  Endpoint: \"http://127.0.0.1:4318\"\n"
	directory, err := WriteGatewayConfig(t.TempDir(), profile)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "config.ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, `MaasSVCAddr: "http://127.0.0.1:29999"`) {
		t.Fatalf("generated config does not contain configured MaaS address:\n%s", content)
	}
	if !strings.Contains(content, `Endpoint: "http://127.0.0.1:4318"`) {
		t.Fatalf("generated config does not contain additional YAML:\n%s", content)
	}
}
