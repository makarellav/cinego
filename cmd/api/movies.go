package main

import (
	"encoding/json"
	"github.com/makarellav/cinego/internal/data"
	"net/http"
	"time"
)

func (app *application) createMovieHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string   `json:"title"`
		Year    int32    `json:"year"`
		Runtime int32    `json:"runtime"`
		Genres  []string `json:"genres"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)

	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, err.Error())

		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"movie": input}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)

	if err != nil {
		app.notFoundResponse(w, r)

		return
	}

	movie := data.Movie{
		ID:        id,
		CreatedAt: time.Now(),
		Title:     "The Breaking Bad",
		Year:      2008,
		Runtime:   50,
		Genres:    []string{"Crime", "Drama", "Thriller"},
		Version:   1,
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
