package data

import "github.com/jackc/pgx/v5/pgxpool"

type Models struct {
	Movies MovieModel
}

func NewModels(db *pgxpool.Pool) *Models {
	return &Models{Movies: MovieModel{DB: db}}
}
