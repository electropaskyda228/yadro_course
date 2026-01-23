package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"yadro.com/course/api/adapters/aaa"
	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/adapters/rest/middleware"
	"yadro.com/course/api/adapters/search"
	"yadro.com/course/api/adapters/update"
	"yadro.com/course/api/adapters/words"
	"yadro.com/course/api/config"
	"yadro.com/course/api/core"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting server")
	log.Debug("debug messages are enabled")

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		log.Error("cannot init words adapter", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := wordsClient.Close(); err != nil {
			log.Error("problem with closing wordsClient", "error", err)
		}
	}()

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		log.Error("cannot init update adapter", "error", err)
		return
	}

	searchClient, err := search.NewClient(cfg.SearchAddress, log)
	if err != nil {
		log.Error("cannot init search adapter", "error", err)
		return
	}

	auth, err := aaa.New(cfg.TokenTTL, log)
	if err != nil {
		log.Error("cannot init auth adapter", "error", err)
		return
	}

	concurrencyLimiter, err := core.NewConcurrencyHandler(cfg.SearchConcurrency)
	if err != nil {
		log.Error("cannot init concurrency limiter", "error", err)
		return
	}
	if err := concurrencyLimiter.Start(); err != nil {
		log.Error("cannot start concurrency limiter", "error", err)
		return
	}
	defer concurrencyLimiter.Stop()

	rateLimiter, err := core.NewRateLimiterHandler(cfg.SearchRate)
	if err != nil {
		log.Error("cannot init rate limiter", "error", err)
		return
	}
	if err := rateLimiter.Start(); err != nil {
		log.Error("cannot start rate limiter", "error", err)
		return
	}
	defer rateLimiter.Stop()

	mux := http.NewServeMux()
	mux.Handle("GET /api/words", rest.NewWordsHandler(log, wordsClient))
	mux.Handle("GET /api/ping", rest.NewPingHandler(log, map[string]core.Pinger{"words": wordsClient, "update": updateClient, "search": searchClient}))
	setHandler(mux, "POST /api/db/update", rest.NewUpdateHandler(log, updateClient), auth)
	mux.Handle("GET /api/db/stats", rest.NewUpdateStatsHandler(log, updateClient))
	mux.Handle("GET /api/db/status", rest.NewUpdateStatusHandler(log, updateClient))
	setHandler(mux, "DELETE /api/db", rest.NewDropHandler(log, updateClient), auth)
	mux.Handle("GET /api/search",
		middleware.Concurrency(rest.NewSearchHandler(log, searchClient), concurrencyLimiter))
	mux.Handle("GET /api/isearch",
		middleware.Rate(rest.NewSearchIndexHandler(log, searchClient), rateLimiter))
	mux.Handle("POST /api/login", rest.NewLoginHandler(log, auth))

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server closed unexpectedly", "error", err)
			return
		}
	}
}

func setHandler(mux *http.ServeMux, endpoint string, f http.HandlerFunc, verifire middleware.TokenVerifier) {
	mux.Handle(endpoint, middleware.Auth(f, verifire))
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
