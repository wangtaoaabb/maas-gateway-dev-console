package config

import (
	"path/filepath"
	"testing"
)

func TestOpenStoreCreatesAndReloadsDefaultProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles", "default.json")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	profile := store.Get()
	if len(profile.Mock.Routes) < 10 {
		t.Fatalf("default routes = %d, want at least 10", len(profile.Mock.Routes))
	}
	profile.Name = "saved"
	profile.Mock.Routes[0].Description = "查询路由列表"
	if err := store.Save(profile); err != nil {
		t.Fatal(err)
	}
	reloaded, err := OpenStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Get().Name != "saved" {
		t.Fatalf("reloaded name = %q", reloaded.Get().Name)
	}
	if reloaded.Get().Mock.Routes[0].Description != "查询路由列表" {
		t.Fatal("route description was not persisted")
	}
	if reloaded.Get().Gateway.SourceDir != defaultGatewayDir {
		t.Fatal("source directory variable was expanded in the saved profile")
	}
}

func TestValidateRejectsDuplicateRouteID(t *testing.T) {
	profile := DefaultProfile()
	profile.Mock.Routes = append(profile.Mock.Routes, profile.Mock.Routes[0])
	if err := Validate(profile); err == nil {
		t.Fatal("Validate accepted duplicate route ID")
	}
}
