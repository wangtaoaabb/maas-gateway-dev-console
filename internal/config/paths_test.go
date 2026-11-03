package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveGatewaySourceDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MAAS_GATEWAY_DIR", "")
	want := filepath.Join(home, "GolandProjects", "maas-gateway")
	got, err := ResolveGatewaySourceDir(defaultGatewayDir)
	if err != nil || got != want {
		t.Fatalf("fallback = %q, %v; want %q", got, err, want)
	}
	custom := filepath.Join(home, "custom", "gateway")
	t.Setenv("MAAS_GATEWAY_DIR", custom)
	got, err = ResolveGatewaySourceDir(defaultGatewayDir)
	if err != nil || got != custom {
		t.Fatalf("environment override = %q, %v; want %q", got, err, custom)
	}
	got, err = ResolveGatewaySourceDir("${HOME}/checkout")
	if err != nil || got != filepath.Join(home, "checkout") {
		t.Fatalf("HOME expansion = %q, %v", got, err)
	}
	got, err = ResolveGatewaySourceDir(custom)
	if err != nil || got != custom {
		t.Fatalf("literal directory = %q, %v; want %q", got, err, custom)
	}
	t.Setenv("DEV_CONSOLE_MISSING_DIR", "")
	if _, err := ResolveGatewaySourceDir("${DEV_CONSOLE_MISSING_DIR}/gateway"); err == nil {
		t.Fatal("accepted an undefined source directory variable")
	}
	if _, err := ResolveGatewaySourceDir(""); err == nil {
		t.Fatal("accepted an empty source directory")
	}
	if _, err := os.Stat(custom); !os.IsNotExist(err) {
		t.Fatalf("resolver must not create the source directory: %v", err)
	}
}
