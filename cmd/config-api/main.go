package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/zailiangs/config-api/internal/api"
	"github.com/zailiangs/config-api/internal/sshconfig"
)

type commandRunner struct{}

func (commandRunner) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

func main() {
	listenAddr := envOrDefault("LISTEN_ADDR", "0.0.0.0:18881")
	configPath := envOrDefault("SSHD_CONFIG_PATH", "/etc/ssh/sshd_config")
	sshd := os.Getenv("SSHD_BINARY")
	if sshd == "" {
		var err error
		sshd, err = exec.LookPath("sshd")
		if err != nil {
			log.Fatalf("find sshd binary: %v", err)
		}
	}

	manager := &sshconfig.Manager{
		ConfigPath: configPath,
		SSHD:       sshd,
		Runner:     commandRunner{},
	}
	server := &http.Server{
		Addr:              listenAddr,
		Handler:           api.NewHandler(manager),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
	log.Printf("config-api listening on %s", listenAddr)
	log.Fatal(server.ListenAndServe())
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
