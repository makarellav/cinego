package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	if app.config.limiter.enabled {
		r.Use(app.rateLimiter)
	}

	r.Use(app.recoverer)

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", app.healthcheckHandler)

		r.Get("/movies", app.listMoviesHandler)
		r.Post("/movies", app.createMovieHandler)
		r.Get("/movies/{id}", app.getMovieHandler)
		r.Patch("/movies/{id}", app.updateMovieHandler)
		r.Delete("/movies/{id}", app.deleteMovieHandler)

		r.Post("/users", app.registerUserHandler)
		r.Put("/users/activated", app.activateUserHandler)

		r.Post("/tokens/authentication", app.createAuthenticationTokenHandler)
	})

	return r
}
