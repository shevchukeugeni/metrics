package postgres

import (
	"database/sql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Config struct {
	Url string
}

func NewPostgresDB(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.Url)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
