package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/local/llmguard/internal/config"
	"github.com/local/llmguard/internal/scanners"
	"github.com/local/llmguard/internal/server"
)

func main() {
	configPath := flag.String("config", "/app/llmguard.yaml", "path to llmguard.yaml")
	flag.Parse()

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// ── Logger ────────────────────────────────────────────────────────────────
	log := buildLogger(cfg.Logging.Level)
	log.Info().
		Str("config", *configPath).
		Str("upstream", cfg.Upstream.URL).
		Int("port", cfg.Server.Port).
		Msg("llmguard starting")

	// ── Scanners ──────────────────────────────────────────────────────────────
	inputScanners, err := scanners.BuildInputScanners(cfg.Scanners.Input)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build input scanners")
	}
	outputScanners, err := scanners.BuildOutputScanners(cfg.Scanners.Output)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to build output scanners")
	}

	log.Info().
		Int("input_scanners", len(inputScanners)).
		Int("output_scanners", len(outputScanners)).
		Msg("scanners loaded")

	// ── Server ────────────────────────────────────────────────────────────────
	srv, err := server.New(cfg.Upstream.URL, inputScanners, outputScanners, log)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create server")
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      srv.Handler(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info().Str("addr", httpServer.Addr).Msg("listening")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	<-quit
	log.Info().Msg("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("shutdown error")
	}
	log.Info().Msg("stopped")
}

func buildLogger(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	return zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger()
}
