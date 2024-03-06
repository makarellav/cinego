package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(app.recoverer)

	if app.config.limiter.enabled {
		r.Use(app.rateLimiter)
	}

	r.Use(app.authenticate)

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", app.healthcheckHandler)

		r.Get("/movies", app.requirePermission("movies:read", app.listMoviesHandler))
		r.Post("/movies", app.requirePermission("movies:write", app.createMovieHandler))
		r.Get("/movies/{id}", app.requirePermission("movies:read", app.getMovieHandler))
		r.Patch("/movies/{id}", app.requirePermission("movies:write", app.updateMovieHandler))
		r.Delete("/movies/{id}", app.requirePermission("movies:write", app.deleteMovieHandler))

		r.Post("/users", app.registerUserHandler)
		r.Put("/users/activated", app.activateUserHandler)

		r.Post("/tokens/authentication", app.createAuthenticationTokenHandler)
	})

	return r
}
