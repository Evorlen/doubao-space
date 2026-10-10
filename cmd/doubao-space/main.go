package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/Icingworld/doubao-space/internal/config"
	"github.com/Icingworld/doubao-space/internal/database"
	"github.com/Icingworld/doubao-space/internal/httpapi"
)

var version = "dev"

func main() {
	dev := flag.Bool("dev", false, "use the project-local data directory")
	addr := flag.String("addr", config.EnvOr("DOUBAO_ADDR", "127.0.0.1:8080"), "HTTP listen address")
	dataDir := flag.String("data-dir", config.EnvOr("DOUBAO_DATA_DIR", ""), "runtime data directory")
	openBrowser := flag.Bool("open-browser", false, "open the application in the default browser")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	resolvedDataDir, err := config.ResolveDataDir(*dataDir, *dev)
	if err != nil {
		logger.Error("resolve data directory", "error", err)
		os.Exit(1)
	}
	if err := config.PrepareDataDir(resolvedDataDir); err != nil {
		logger.Error("prepare data directory", "error", err)
		os.Exit(1)
	}

	db, err := database.Open(resolvedDataDir)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	server := &http.Server{
		Addr:              *addr,
		Handler:           httpapi.NewRouterWithDataDir(db, version, *dev, resolvedDataDir),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		logger.Error("listen", "address", *addr, "error", err)
		os.Exit(1)
	}
	defer listener.Close()

	url := listenURL(listener.Addr().String())
	logger.Info("doubao-space started", "url", url, "data_dir", resolvedDataDir, "version", version)
	if *openBrowser {
		if err := openURL(url); err != nil {
			logger.Warn("open browser", "url", url, "error", err)
		}
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
	case <-shutdown:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("shutdown HTTP server", "error", err)
			os.Exit(1)
		}
		logger.Info("doubao-space stopped")
	}
}

func listenURL(address string) string {
	if strings.HasPrefix(address, ":") {
		return "http://127.0.0.1" + address
	}
	if strings.HasPrefix(address, "0.0.0.0:") {
		return "http://127.0.0.1:" + strings.TrimPrefix(address, "0.0.0.0:")
	}
	return "http://" + address
}

func openURL(url string) error {
	var command string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		command, args = "open", []string{url}
	case "linux":
		command, args = "xdg-open", []string{url}
	case "windows":
		command, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
	return exec.Command(command, args...).Start()
}
