package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
)

func (app *application) routes() http.Handler {
	r := chi.NewRouter()

	r.Use(app.recoverer)

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Route("/v1", func(r chi.Router) {
		r.Get("/healthcheck", app.healthcheckHandler)
		r.Get("/movies/{id}", app.getMovieHandler)
		r.Post("/movies", app.createMovieHandler)
	})

	return r
}
