package main

import (
	"fmt"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"sync"
	"time"
)

func (app *application) recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func (app *application) rateLimiter(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var mu sync.Mutex
	clients := make(map[string]*client)

	go func() {
		ticker := time.NewTicker(time.Minute)

		for range ticker.C {
			mu.Lock()
			for ip, c := range clients {
				if time.Since(c.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)

		if err != nil {
			app.serverErrorResponse(w, r, err)

			return
		}

		mu.Lock()
		if _, ok := clients[ip]; !ok {
			clients[ip] = &client{
				limiter:  rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				lastSeen: time.Now(),
			}
		}

		if !clients[ip].limiter.Allow() {
			mu.Unlock()

			app.rateLimitExceededResponse(w, r)

			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}
