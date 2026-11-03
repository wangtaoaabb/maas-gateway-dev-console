package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveGatewaySourceDir expands profile variables only when starting Gateway,
// preserving the portable expression in the saved profile and Web form.
func ResolveGatewaySourceDir(source string) (string, error) {
	var resolveErr error
	expanded := os.Expand(source, func(name string) string {
		if value, ok := os.LookupEnv(name); ok && value != "" {
			return value
		}
		if name == "MAAS_GATEWAY_DIR" {
			home, err := os.UserHomeDir()
			if err != nil {
				resolveErr = fmt.Errorf("resolve home directory: %w", err)
				return ""
			}
			return filepath.Join(home, "GolandProjects", "maas-gateway")
		}
		resolveErr = fmt.Errorf("source directory environment variable %q is not set", name)
		return ""
	})
	if resolveErr != nil {
		return "", resolveErr
	}
	if expanded == "" {
		return "", fmt.Errorf("gateway source directory is empty")
	}
	return filepath.Abs(expanded)
}
