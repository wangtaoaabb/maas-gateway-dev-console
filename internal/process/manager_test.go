package process

import (
	"reflect"
	"testing"

	"maas-gateway-dev-console/internal/config"
)

func TestGatewayCommand(t *testing.T) {
	profile := config.DefaultProfile()
	command, args := gatewayCommand(profile, "debug")
	if command != profile.Gateway.Delve.Binary {
		t.Fatalf("command = %q", command)
	}
	want := []string{"debug", ".", "--headless", "--listen=127.0.0.1:2345", "--api-version=2", "--accept-multiclient", "--check-go-version=false"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestWithEnvironmentReplacesValues(t *testing.T) {
	result := withEnvironment([]string{"A=1", "ENV=old"}, map[string]string{"ENV": "ci", "CONF_PATH": "/tmp/config"})
	want := map[string]bool{"A=1": true, "ENV=ci": true, "CONF_PATH=/tmp/config": true}
	for _, item := range result {
		delete(want, item)
	}
	if len(want) != 0 {
		t.Fatalf("missing environment values: %#v; got %#v", want, result)
	}
}
