package sshconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type runnerFunc func(name string, args ...string) ([]byte, error)

func (f runnerFunc) CombinedOutput(name string, args ...string) ([]byte, error) {
	return f(name, args...)
}

func TestManagerSetPortSuccessAndIdempotent(t *testing.T) {
	path := writeTestConfig(t, "Port 22\n")
	restarts := 0
	runner := runnerFunc(func(name string, args ...string) ([]byte, error) {
		command := name + " " + strings.Join(args, " ")
		switch {
		case strings.Contains(command, " -t "):
			return nil, nil
		case strings.Contains(command, " -T "):
			return []byte("port 18822\n"), nil
		case command == "systemctl restart ssh":
			restarts++
			return nil, nil
		case command == "systemctl is-active --quiet ssh":
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", command)
		}
	})
	manager := &Manager{ConfigPath: path, SSHD: "sshd", Runner: runner}

	first, err := manager.SetPort(18822)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.SetPort(18822)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Changed || second.Changed || restarts != 1 {
		t.Fatalf("first=%+v second=%+v restarts=%d", first, second, restarts)
	}
	if _, err := os.Stat(path + ".config-api.bak"); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}

func TestManagerValidationFailureRollsBack(t *testing.T) {
	const original = "Port 22\n"
	path := writeTestConfig(t, original)
	validateCalls := 0
	runner := runnerFunc(func(name string, args ...string) ([]byte, error) {
		command := name + " " + strings.Join(args, " ")
		switch {
		case strings.Contains(command, " -t "):
			validateCalls++
			if validateCalls == 1 {
				return []byte("bad config"), errors.New("exit 1")
			}
			return nil, nil
		case command == "systemctl restart ssh", command == "systemctl is-active --quiet ssh":
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", command)
		}
	})
	manager := &Manager{ConfigPath: path, SSHD: "sshd", Runner: runner}

	result, err := manager.SetPort(18822)
	if err == nil || !result.RolledBack {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != original {
		t.Fatalf("configuration was not restored: %q", got)
	}
}

func TestManagerRejectsMultipleEffectivePortsAndRollsBack(t *testing.T) {
	path := writeTestConfig(t, "Include extra.conf\nPort 22\n")
	runner := runnerFunc(func(name string, args ...string) ([]byte, error) {
		command := name + " " + strings.Join(args, " ")
		switch {
		case strings.Contains(command, " -t "):
			return nil, nil
		case strings.Contains(command, " -T "):
			return []byte("port 18822\nport 2200\n"), nil
		case command == "systemctl restart ssh", command == "systemctl is-active --quiet ssh":
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", command)
		}
	})
	manager := &Manager{ConfigPath: path, SSHD: "sshd", Runner: runner}
	result, err := manager.SetPort(18822)
	if err == nil || !result.RolledBack {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestManagerFallsBackToSSHDService(t *testing.T) {
	path := writeTestConfig(t, "Port 22\n")
	var commands []string
	runner := runnerFunc(func(name string, args ...string) ([]byte, error) {
		command := name + " " + strings.Join(args, " ")
		commands = append(commands, command)
		switch {
		case strings.Contains(command, " -t "):
			return nil, nil
		case strings.Contains(command, " -T "):
			return []byte("port 18822\n"), nil
		case command == "systemctl restart ssh":
			return []byte("Unit ssh.service not found."), errors.New("exit 5")
		case command == "systemctl restart sshd", command == "systemctl is-active --quiet sshd":
			return nil, nil
		default:
			return nil, fmt.Errorf("unexpected command: %s", command)
		}
	})
	manager := &Manager{ConfigPath: path, SSHD: "sshd", Runner: runner}
	if _, err := manager.SetPort(18822); err != nil {
		t.Fatal(err)
	}
	if !contains(commands, "systemctl restart sshd") {
		t.Fatalf("sshd fallback not called: %v", commands)
	}
}

func TestManagerSerializesConcurrentRequests(t *testing.T) {
	path := writeTestConfig(t, "Port 22\n")
	var mu sync.Mutex
	active, maximum := 0, 0
	runner := runnerFunc(func(name string, args ...string) ([]byte, error) {
		command := name + " " + strings.Join(args, " ")
		mu.Lock()
		active++
		if active > maximum {
			maximum = active
		}
		mu.Unlock()
		time.Sleep(time.Millisecond)
		mu.Lock()
		active--
		mu.Unlock()

		if strings.Contains(command, " -T ") {
			data, _ := os.ReadFile(path)
			if strings.Contains(string(data), "18822") {
				return []byte("port 18822\n"), nil
			}
			return []byte("port 18823\n"), nil
		}
		return nil, nil
	})
	manager := &Manager{ConfigPath: path, SSHD: "sshd", Runner: runner}

	var wg sync.WaitGroup
	for _, port := range []int{18822, 18823} {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			if _, err := manager.SetPort(port); err != nil {
				t.Errorf("SetPort(%d): %v", port, err)
			}
		}(port)
	}
	wg.Wait()
	if maximum != 1 {
		t.Fatalf("commands overlapped, maximum active=%d", maximum)
	}
}

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sshd_config")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
