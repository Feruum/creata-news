//go:build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/studyverse/ems-backend/internal/testsupport"
)

func TestBackendStartsWithMiniAppDisabled(t *testing.T) {
	if os.Getenv("RUN_BACKEND_STARTUP_TESTS") != "1" {
		t.Skip("RUN_BACKEND_STARTUP_TESTS requires disposable PostgreSQL and Redis")
	}
	redisURL := os.Getenv("BACKEND_STARTUP_REDIS_URL")
	parsedRedis, err := url.Parse(redisURL)
	if err != nil || parsedRedis.Hostname() != "127.0.0.1" || parsedRedis.Scheme != "redis" {
		t.Fatal("BACKEND_STARTUP_REDIS_URL must name an isolated loopback Redis server")
	}
	db := testsupport.ClonedDB(t)
	var database string
	if err := db.NewRaw("SELECT current_database()").Scan(context.Background(), &database); err != nil {
		t.Fatal(err)
	}
	dsn, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	dsn.Path = "/" + database
	directory := t.TempDir()
	binary := filepath.Join(directory, "backend")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build actual server: %v\n%s", err, output)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(directory, "server.log")
	log, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	server := exec.CommandContext(ctx, binary)
	server.Dir = directory
	server.Env = append(os.Environ(),
		"ENV_FILE="+filepath.Join(directory, "absent.env"),
		"HOST=127.0.0.1", fmt.Sprintf("PORT=%d", port),
		"DATABASE_URL="+dsn.String(), "REDIS_URL="+redisURL,
		"KRATOS_PUBLIC_URL=http://127.0.0.1:1", "KRATOS_ADMIN_URL=http://127.0.0.1:1",
		"SPICEDB_ENDPOINT=127.0.0.1:1", "SPICEDB_PRESHARED_KEY=startup-test", "SPICEDB_INSECURE=true",
		"MEILISEARCH_URL=http://127.0.0.1:1", "MEILISEARCH_MASTER_KEY=",
		"STORAGE_ACCOUNT_NAME=", "STORAGE_ACCOUNT_KEY=", "ADMIN_SECRET=",
		"TELEGRAM_BOT_URL=http://127.0.0.1:1", "TELEGRAM_BOT_TOKEN=",
		"TELEGRAM_MINIAPP_CLIENT_SECRET=", "TELEGRAM_MINIAPP_PRIVATE_KEY_B64=", "TELEGRAM_MINIAPP_REDIRECT_URI=",
		"INTERNAL_REMINDER_SECRET=startup-test", "MAP_APP_URL=http://127.0.0.1:3001",
		"OTEL_ENABLED=false", "OTEL_ENVIRONMENT=test",
	)
	server.Stdout, server.Stderr = log, log
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = server.Wait(); close(exited) }()
	defer func() { cancel(); <-exited }()
	readLog := func() string { output, _ := os.ReadFile(logPath); return string(output) }
	client := &http.Client{Timeout: 2 * time.Second}
	address := fmt.Sprintf("http://127.0.0.1:%d", port)
	deadline := time.Now().Add(30 * time.Second)
	for {
		select {
		case <-exited:
			t.Fatalf("server exited before readiness: %s\n%s", server.ProcessState, readLog())
		default:
		}
		response, err := client.Get(address + "/health/ready")
		if err == nil {
			var result struct {
				Status string            `json:"status"`
				Checks map[string]string `json:"checks"`
			}
			decodeErr := json.NewDecoder(response.Body).Decode(&result)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK && decodeErr == nil && result.Status == "ready" && result.Checks["postgres"] == "ok" && result.Checks["redis"] == "ok" {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became ready\n%s", readLog())
		}
		time.Sleep(100 * time.Millisecond)
	}
	response, err := client.Get(address + "/telegram-miniapp/status")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var status struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || status.Enabled {
		t.Fatalf("unconfigured Mini App should be disabled without stopping the server: status=%d enabled=%t", response.StatusCode, status.Enabled)
	}
}
