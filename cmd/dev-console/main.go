package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"maas-gateway-dev-console/internal/config"
	"maas-gateway-dev-console/internal/mock"
	"maas-gateway-dev-console/internal/process"
	"maas-gateway-dev-console/internal/web"
)

func main() {
	mode := "console"
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "mock" {
		mode = "mock"
		args = args[1:]
	}
	flags := flag.NewFlagSet("dev-console", flag.ExitOnError)
	profilePath := flags.String("profile", "profiles/default.json", "path to the local development profile")
	_ = flags.Parse(args)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	store, err := config.OpenStore(*profilePath)
	if err != nil {
		logger.Error("failed to open profile", "error", err)
		os.Exit(1)
	}
	mockServer := mock.NewServer(store, logger)
	if mode == "mock" {
		runMockOnly(mockServer, logger)
		return
	}

	projectRoot, err := filepath.Abs(".")
	if err != nil {
		logger.Error("failed to resolve project root", "error", err)
		os.Exit(1)
	}
	manager := process.NewManager(projectRoot)
	webServer := web.NewServer(store, mockServer, manager, logger)
	errors := make(chan error, 2)
	go func() { errors <- mockServer.Start() }()
	go func() { errors <- webServer.Start() }()

	profile := store.Get()
	fmt.Printf("\nMaaS Gateway Dev Console\n  Web:  http://%s\n  Mock: http://%s\n\n", profile.Console.ListenAddress, profile.Mock.ListenAddress)
	waitForExit(errors, func(ctx context.Context) {
		_ = manager.Stop(ctx)
		_ = webServer.Shutdown(ctx)
		_ = mockServer.Shutdown(ctx)
	}, logger)
}

func runMockOnly(server *mock.Server, logger *slog.Logger) {
	errors := make(chan error, 1)
	go func() { errors <- server.Start() }()
	waitForExit(errors, func(ctx context.Context) { _ = server.Shutdown(ctx) }, logger)
}

func waitForExit(errors <-chan error, shutdown func(context.Context), logger *slog.Logger) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	select {
	case signal := <-signals:
		logger.Info("shutdown requested", "signal", signal.String())
	case err := <-errors:
		if err != nil {
			logger.Error("server stopped", "error", err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	shutdown(ctx)
}
