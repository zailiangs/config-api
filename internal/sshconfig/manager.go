package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Runner interface {
	CombinedOutput(name string, args ...string) ([]byte, error)
}

type Result struct {
	Port       int
	Changed    bool
	RolledBack bool
}

type Manager struct {
	ConfigPath string
	SSHD       string
	Runner     Runner

	mu sync.Mutex
}

func (m *Manager) SetPort(port int) (Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := Result{Port: port}
	original, info, err := readConfig(m.ConfigPath)
	if err != nil {
		return result, err
	}

	updated, changed, err := SetGlobalPort(original, port)
	if err != nil {
		return result, err
	}
	result.Changed = changed

	if !changed {
		if err := m.validate(port); err != nil {
			return result, fmt.Errorf("existing configuration is not valid for requested port: %w", err)
		}
		return result, nil
	}

	backupPath := m.ConfigPath + ".config-api.bak"
	if err := atomicWrite(backupPath, original, info.Mode()); err != nil {
		return result, fmt.Errorf("create backup: %w", err)
	}
	if err := atomicWrite(m.ConfigPath, updated, info.Mode()); err != nil {
		return result, fmt.Errorf("write sshd configuration: %w", err)
	}

	if err := m.validate(port); err != nil {
		return m.rollback(result, original, info.Mode(), fmt.Errorf("validate sshd configuration: %w", err))
	}
	if _, err := m.restartAndVerify(); err != nil {
		return m.rollback(result, original, info.Mode(), fmt.Errorf("restart ssh service: %w", err))
	}
	return result, nil
}

func (m *Manager) validate(port int) error {
	if output, err := m.Runner.CombinedOutput(m.SSHD, "-t", "-f", m.ConfigPath); err != nil {
		return commandError(output, err)
	}

	output, err := m.Runner.CombinedOutput(m.SSHD, "-T", "-f", m.ConfigPath)
	if err != nil {
		return commandError(output, err)
	}

	var ports []int
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.EqualFold(fields[0], "port") {
			value, parseErr := strconv.Atoi(fields[1])
			if parseErr != nil {
				return fmt.Errorf("unexpected sshd -T port output %q", line)
			}
			ports = append(ports, value)
		}
	}
	if len(ports) != 1 || ports[0] != port {
		return fmt.Errorf("effective SSH ports are %v, expected only %d", ports, port)
	}
	return nil
}

func (m *Manager) restartAndVerify() (string, error) {
	service := "ssh"
	output, err := m.Runner.CombinedOutput("systemctl", "restart", service)
	if err != nil {
		if !serviceNotFound(string(output)) {
			return service, commandError(output, err)
		}
		service = "sshd"
		output, err = m.Runner.CombinedOutput("systemctl", "restart", service)
		if err != nil {
			return service, commandError(output, err)
		}
	}

	output, err = m.Runner.CombinedOutput("systemctl", "is-active", "--quiet", service)
	if err != nil {
		return service, commandError(output, err)
	}
	return service, nil
}

func (m *Manager) rollback(result Result, original []byte, mode os.FileMode, cause error) (Result, error) {
	result.RolledBack = true
	if err := atomicWrite(m.ConfigPath, original, mode); err != nil {
		return result, fmt.Errorf("%v; rollback write failed: %w", cause, err)
	}
	if _, err := m.restartAndVerify(); err != nil {
		return result, fmt.Errorf("%v; configuration restored but SSH restart failed: %w", cause, err)
	}
	return result, cause
}

func readConfig(path string) ([]byte, os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("stat sshd configuration: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read sshd configuration: %w", err)
	}
	return data, info, nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)

	if err := file.Chmod(mode.Perm()); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func serviceNotFound(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "unit ssh.service not found") ||
		strings.Contains(lower, "unit ssh.service could not be found")
}

func commandError(output []byte, err error) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, message)
}
