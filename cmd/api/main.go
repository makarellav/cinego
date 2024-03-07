package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/makarellav/cinego/internal/data"
	"github.com/makarellav/cinego/internal/mailer"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
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
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config config
	logger *slog.Logger
	models *data.Models
	mailer *mailer.Mailer
	wg     sync.WaitGroup
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

	smtpPort, err := strconv.Atoi(os.Getenv("SMTP_PORT"))

	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	flag.StringVar(&cfg.smtp.host, "smtp_host", os.Getenv("SMTP_HOST"), "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp_port", smtpPort, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp_username", os.Getenv("SMTP_USERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp_password", os.Getenv("SMTP_PASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp_sender", os.Getenv("SMTP_SENDER"), "SMTP sender")

	flag.Func("cors_trusted_origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)

		return nil
	})

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
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()

	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
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
