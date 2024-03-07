package main

import (
	"errors"
	"fmt"
	"github.com/makarellav/cinego/internal/data"
	"github.com/makarellav/cinego/internal/validator"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"strings"
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

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authentication")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerParts := strings.Split(authorizationHeader, " ")

		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)

			return
		}

		token := headerParts[1]

		v := validator.New()

		if data.ValidateToken(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)

			return
		}

		user, err := app.models.Users.GetByToken(data.ScopeAuthentication, token)

		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}

			return
		}

		r = app.contextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}

func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticatedRequiredResponse(w, r)

			return
		}

		next.ServeHTTP(w, r)
	}
}

func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if !user.Activated {
			app.inactiveAccountResponse(w, r)

			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireAuthenticatedUser(fn)
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		permissions, err := app.models.Permissions.GetAllForUser(user.ID)

		if err != nil {
			app.notPermittedResponse(w, r)

			return
		}

		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)

			return
		}

		next.ServeHTTP(w, r)
	}

	return app.requireActivatedUser(fn)
}

func (app *application) enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")

		origin := r.Header.Get("Origin")

		if origin != "" {
			for i := range app.config.cors.trustedOrigins {
				if app.config.cors.trustedOrigins[i] == origin {
					w.Header().Add("Access-Control-Allow-Origin", origin)

					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Allow-Methods") != "" {
						w.Header().Add("Access-Control-Allow-Methods", "OPTIONS, PUT, POST, PATCH")
						w.Header().Add("Access-Control-Allow-Headers", "Authorization, Content-Type")

						w.WriteHeader(http.StatusOK)
						return
					}

					break
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
