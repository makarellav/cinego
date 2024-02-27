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
		r.Get("/movies", app.listMoviesHandler)
		r.Post("/movies", app.createMovieHandler)
		r.Patch("/movies/{id}", app.updateMovieHandler)
		r.Delete("/movies/{id}", app.deleteMovieHandler)
	})

	return r
}
