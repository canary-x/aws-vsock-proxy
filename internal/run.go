package internal

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func Run() error {
	cfg, err := parseConfigFromEnv()
	if err != nil {
		return errors.Wrap(err, "loading config")
	}

	log, err := zap.NewProduction()
	if err != nil {
		return errors.Wrap(err, "instantiating logger")
	}

	ln, err := listen(cfg, log)
	if err != nil {
		return errors.Wrapf(err, "listening on port %d", cfg.ServerPort)
	}

	ctx, cancel := context.WithCancel(context.Background())
	registerInterrupt(cancel, log)

	rtr := mux.NewRouter()
	if err := registerRoutes(ctx, rtr); err != nil {
		return errors.Wrap(err, "registering routes")
	}
	log.Sugar().Infof("Starting server listening on port %d", cfg.ServerPort)

	srv := NewServer(rtr, cfg)

	defer func() {
		log.Info("Process terminated")
	}()
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	err = srv.Serve(ln)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.Wrap(err, "serving http")
	}
	log.Info("Server terminated")

	return nil
}

func registerInterrupt(cancelCtx context.CancelFunc, log *zap.Logger) {
	stopServer := func() {
		log.Info("Received SIGINT. Shutting down gracefully.")
		cancelCtx()
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c // Received shutdown signal
		stopServer()
	}()
}
