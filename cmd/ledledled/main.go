// Command ledledled runs the LED strip controller: render engine, sACN /
// Art-Net output and the web UI, all in one process.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ledledled/internal/engine"
	"ledledled/internal/library"
	"ledledled/internal/server"
	"ledledled/internal/show"
	"ledledled/internal/webui"
)

func main() {
	configPath := flag.String("config", "show.json", "path to the show file (created if missing)")
	listen := flag.String("listen", ":8080", "HTTP listen address for the web UI")
	libDir := flag.String("library", "", "directory for saved shows/presets (default: <config dir>/library)")
	debug := flag.Bool("debug", false, "verbose logging")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	if *libDir == "" {
		*libDir = filepath.Join(filepath.Dir(*configPath), "library")
	}
	if err := run(*configPath, *listen, *libDir); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(configPath, listen, libDir string) error {
	lib, err := library.New(libDir)
	if err != nil {
		return err
	}
	sh, err := show.Load(configPath)
	if err != nil {
		return err
	}
	eng, err := engine.New(sh)
	if err != nil {
		return err
	}
	// Persist immediately so a freshly generated sACN CID stays stable.
	if _, statErr := os.Stat(configPath); errors.Is(statErr, os.ErrNotExist) {
		if err := show.Save(configPath, eng.Show()); err != nil {
			slog.Warn("could not write initial show", "err", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := server.New(eng, configPath, lib, webui.FS())
	defer srv.Close()

	httpSrv := &http.Server{Addr: listen, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		slog.Info("web UI listening", "addr", listen)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server", "err", err)
			stop()
		}
	}()

	engDone := make(chan struct{})
	go func() {
		eng.Run(ctx)
		close(engDone)
	}()
	slog.Info("engine running", "config", configPath,
		"controllers", len(sh.Controllers), "strips", len(sh.Strips), "fps", sh.Settings.FPS)

	<-ctx.Done()
	slog.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
	<-engDone
	return nil
}
