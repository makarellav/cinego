package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.port),
		Handler:      app.routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  time.Minute,
		ErrorLog:     slog.NewLogLogger(app.logger.Handler(), slog.LevelError),
	}

	errCh := make(chan error)

	go func() {
		quitCh := make(chan os.Signal, 1)
		signal.Notify(quitCh, syscall.SIGINT, syscall.SIGTERM)
		s := <-quitCh

		app.logger.Info("shutting down the server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		errCh <- srv.Shutdown(ctx)
	}()

	app.logger.Info("starting the server", "addr", srv.Addr, "env", app.config.env)

	err := srv.ListenAndServe()

	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-errCh

	if err != nil {
		return err
	}

	app.logger.Info("stopped the server", "addr", srv.Addr)

	return nil
}
