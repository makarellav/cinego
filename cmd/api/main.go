package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/makarellav/cinego/internal/data"
	"log/slog"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		url          string
		maxOpenConns int
		maxIdleTime  time.Duration
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

type application struct {
	config config
	logger *slog.Logger
	models *data.Models
}

func main() {
	var cfg config

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := godotenv.Load()

	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")

	flag.StringVar(&cfg.db.url, "db_url", os.Getenv("DB_URL"), "PostgreSQL URL")
	flag.IntVar(&cfg.db.maxOpenConns, "db_max_open_conns", 25, "PostrgreSQL max open connections")
	flag.DurationVar(&cfg.db.maxIdleTime, "db_max_idle_time", 15*time.Minute, "PostgreSQL max connection idle time")

	flag.Float64Var(&cfg.limiter.rps, "limiter_rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter_burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter_enabled", true, "Enable rate limiter")

	flag.Parse()

	db, err := openDB(cfg)
	defer db.Close()

	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	logger.Info("database connection pool established")

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(db),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  time.Minute,
		ErrorLog:     slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.Info("starting the server", "addr", srv.Addr, "env", cfg.env)

	err = srv.ListenAndServe()

	logger.Error(err.Error())
	os.Exit(1)
}

func openDB(cfg config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbCfg, err := pgxpool.ParseConfig(fmt.Sprintf("%s?pool_max_conns=%d&pool_max_conn_idle_time=%v", cfg.db.url, cfg.db.maxOpenConns, cfg.db.maxIdleTime))

	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, dbCfg)

	if err != nil {
		pool.Close()

		return nil, err
	}

	return pool, nil
}
