package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct {
	mu      sync.RWMutex
	path    string
	profile Profile
}

func OpenStore(path string) (*Store, error) {
	store := &Store{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		store.profile = DefaultProfile()
		if err := store.Save(store.profile); err != nil {
			return nil, err
		}
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profile: %w", err)
	}
	if err := json.Unmarshal(data, &store.profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	if err := Validate(store.profile); err != nil {
		return nil, fmt.Errorf("validate profile: %w", err)
	}
	return store, nil
}

func (s *Store) Get() Profile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	profile, err := Clone(s.profile)
	if err != nil {
		panic(err)
	}
	return profile
}

func (s *Store) Save(profile Profile) error {
	if err := Validate(profile); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create profile directory: %w", err)
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write temporary profile: %w", err)
	}
	if err := os.Rename(temporary, s.path); err != nil {
		return fmt.Errorf("replace profile: %w", err)
	}
	s.mu.Lock()
	s.profile = profile
	s.mu.Unlock()
	return nil
}

func Validate(profile Profile) error {
	if profile.Version != CurrentVersion {
		return fmt.Errorf("unsupported profile version %d", profile.Version)
	}
	if strings.TrimSpace(profile.Name) == "" {
		return errors.New("profile name is required")
	}
	if strings.TrimSpace(profile.Console.ListenAddress) == "" || strings.TrimSpace(profile.Mock.ListenAddress) == "" {
		return errors.New("console and mock listen addresses are required")
	}
	if profile.Mock.RequestLimit <= 0 {
		return errors.New("mock request_limit must be positive")
	}
	seen := make(map[string]struct{}, len(profile.Mock.Routes))
	for index, route := range profile.Mock.Routes {
		if route.ID == "" || route.Method == "" || route.Path == "" {
			return fmt.Errorf("mock.routes[%d] requires id, method and path", index)
		}
		if !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("mock.routes[%d].path must start with /", index)
		}
		if _, ok := seen[route.ID]; ok {
			return fmt.Errorf("duplicate route id %q", route.ID)
		}
		seen[route.ID] = struct{}{}
		if route.Response.Status < 100 || route.Response.Status > 599 {
			return fmt.Errorf("mock.routes[%d].response.status is invalid", index)
		}
		if route.Response.DelayMS < 0 {
			return fmt.Errorf("mock.routes[%d].response.delay_ms must not be negative", index)
		}
	}
	if profile.Gateway.SourceDir == "" || profile.Gateway.Environment == "" {
		return errors.New("gateway source_dir and environment are required")
	}
	if profile.Gateway.HTTPPort <= 0 || profile.Gateway.MetricsPort <= 0 || profile.Gateway.Redis.Port <= 0 {
		return errors.New("gateway, metrics and redis ports must be positive")
	}
	return nil
}
