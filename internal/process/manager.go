package process

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"maas-gateway-dev-console/internal/config"
	runtimeconfig "maas-gateway-dev-console/internal/runtime"
)

type State struct {
	Status       string    `json:"status"`
	Mode         string    `json:"mode,omitempty"`
	PID          int       `json:"pid,omitempty"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	ConfigDir    string    `json:"config_dir,omitempty"`
	GatewayURL   string    `json:"gateway_url,omitempty"`
	DelveAddress string    `json:"delve_address,omitempty"`
	Error        string    `json:"error,omitempty"`
}

type Manager struct {
	mu          sync.Mutex
	projectRoot string
	cmd         *exec.Cmd
	state       State
	generation  uint64
	stopAsked   bool
	logs        *LogStore
}

func NewManager(projectRoot string) *Manager {
	return &Manager{projectRoot: projectRoot, state: State{Status: "stopped"}, logs: NewLogStore(5000)}
}

func (m *Manager) Logs() *LogStore { return m.logs }

func (m *Manager) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

func (m *Manager) Start(profile config.Profile, mode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cmd != nil {
		return errors.New("gateway is already running")
	}
	if mode != "run" && mode != "debug" {
		return fmt.Errorf("unsupported gateway mode %q", mode)
	}
	sourceDir, err := config.ResolveGatewaySourceDir(profile.Gateway.SourceDir)
	if err != nil {
		return err
	}
	profile.Gateway.SourceDir = sourceDir
	if info, err := os.Stat(sourceDir); err != nil || !info.IsDir() {
		return fmt.Errorf("gateway source directory is unavailable: %s", profile.Gateway.SourceDir)
	}
	if mode == "run" {
		if err := requireFreePort(profile.Gateway.HTTPPort, "gateway"); err != nil {
			return err
		}
	}
	if mode == "debug" {
		_, portText, err := net.SplitHostPort(profile.Gateway.Delve.ListenAddress)
		if err != nil {
			return fmt.Errorf("invalid Delve listen address: %w", err)
		}
		port, _ := strconv.Atoi(portText)
		if err := requireFreePort(port, "Delve"); err != nil {
			return err
		}
	}
	runtimeRoot := filepath.Join(m.projectRoot, ".runtime", "current")
	configDir, err := runtimeconfig.WriteGatewayConfig(runtimeRoot, profile)
	if err != nil {
		return err
	}
	command, args := gatewayCommand(profile, mode)
	cmd := exec.Command(command, args...)
	cmd.Dir = profile.Gateway.SourceDir
	cmd.Env = withEnvironment(os.Environ(), map[string]string{
		"ENV":       profile.Gateway.Environment,
		"CONF_PATH": configDir,
	})
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start gateway: %w", err)
	}
	m.generation++
	generation := m.generation
	m.cmd = cmd
	m.stopAsked = false
	status := "starting"
	if mode == "debug" && !profile.Gateway.Delve.Continue {
		status = "waiting_for_debugger"
	}
	m.state = State{
		Status:       status,
		Mode:         mode,
		PID:          cmd.Process.Pid,
		StartedAt:    time.Now().UTC(),
		ConfigDir:    configDir,
		GatewayURL:   fmt.Sprintf("http://127.0.0.1:%d", profile.Gateway.HTTPPort),
		DelveAddress: profile.Gateway.Delve.ListenAddress,
	}
	m.logs.Add("console", fmt.Sprintf("started gateway in %s mode with pid %d", mode, cmd.Process.Pid))
	go m.capture("gateway", stdout)
	go m.capture("gateway", stderr)
	go m.wait(generation, cmd)
	if status == "starting" {
		go m.watchHealth(generation, profile.Gateway.HTTPPort)
	}
	return nil
}

func gatewayCommand(profile config.Profile, mode string) (string, []string) {
	if mode == "run" {
		return "go", []string{"run", "."}
	}
	args := []string{
		"debug", ".", "--headless", "--listen=" + profile.Gateway.Delve.ListenAddress,
		"--api-version=2", "--accept-multiclient",
		fmt.Sprintf("--check-go-version=%t", profile.Gateway.Delve.CheckGoVersion),
	}
	if profile.Gateway.Delve.Continue {
		args = append(args, "--continue")
	}
	return profile.Gateway.Delve.Binary, args
}

func withEnvironment(existing []string, values map[string]string) []string {
	result := make([]string, 0, len(existing)+len(values))
	for _, item := range existing {
		name, _, _ := strings.Cut(item, "=")
		if _, replaced := values[name]; !replaced {
			result = append(result, item)
		}
	}
	for name, value := range values {
		result = append(result, name+"="+value)
	}
	return result
}

func requireFreePort(port int, name string) error {
	connection, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 150*time.Millisecond)
	if err != nil {
		return nil
	}
	_ = connection.Close()
	return fmt.Errorf("%s port %d is already in use", name, port)
}

func (m *Manager) capture(source string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 4*1024*1024)
	for scanner.Scan() {
		m.logs.Add(source, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		m.logs.Add("console", "log reader failed: "+err.Error())
	}
}

func (m *Manager) watchHealth(generation uint64, port int) {
	url := fmt.Sprintf("127.0.0.1:%d", port)
	for attempt := 0; attempt < 120; attempt++ {
		connection, err := net.DialTimeout("tcp", url, 200*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			m.mu.Lock()
			if m.generation == generation && m.cmd != nil {
				m.state.Status = "running"
			}
			m.mu.Unlock()
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	m.mu.Lock()
	if m.generation == generation && m.cmd != nil {
		m.state.Status = "failed"
		m.state.Error = "gateway did not open its HTTP port within 30 seconds"
	}
	m.mu.Unlock()
}

func (m *Manager) wait(generation uint64, cmd *exec.Cmd) {
	err := cmd.Wait()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.generation != generation || m.cmd != cmd {
		return
	}
	m.cmd = nil
	if m.stopAsked {
		m.stopAsked = false
		m.state.Status = "stopped"
		m.state.PID = 0
		m.state.Error = ""
		m.logs.Add("console", "gateway stopped")
		return
	}
	m.state.Status = "stopped"
	m.state.PID = 0
	if err != nil {
		m.state.Status = "failed"
		m.state.Error = err.Error()
		m.logs.Add("console", "gateway exited: "+err.Error())
	} else {
		m.logs.Add("console", "gateway stopped")
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	cmd := m.cmd
	if cmd == nil || cmd.Process == nil {
		m.state.Status = "stopped"
		m.mu.Unlock()
		return nil
	}
	m.state.Status = "stopping"
	m.stopAsked = true
	pid := cmd.Process.Pid
	m.mu.Unlock()

	_ = syscall.Kill(-pid, syscall.SIGTERM)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		m.mu.Lock()
		stopped := m.cmd == nil
		m.mu.Unlock()
		if stopped {
			return nil
		}
		select {
		case <-ctx.Done():
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
