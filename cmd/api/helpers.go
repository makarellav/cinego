package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
)

type envelope map[string]any

func (app *application) readIDParam(r *http.Request) (int64, error) {
	idParam := r.PathValue("id")

	id, err := strconv.ParseInt(idParam, 10, 64)

	if err != nil || id < 1 {
		return 0, errors.New("invalid id parameter")
	}

	return id, nil
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, header http.Header) error {
	resp, err := json.Marshal(data)

	if err != nil {
		return err
	}

	resp = append(resp, '\n')

	for k, v := range header {
		w.Header()[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(resp)

	return nil
}
