package postgres

import (
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/shevchukeugeni/metrics/internal/store/postgres/migrations"
)

type Config struct {
	URL string
}

func NewPostgresDB(cfg Config) (*sql.DB, error) {
	if cfg.URL == "" {
		return nil, errors.New("incorrect URL")
	}

	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	if err = migrateDB(db, "schema_migration"); err != nil {
		return nil, err
	}

	return db, nil

}

func migrateDB(db *sql.DB, table string) error {
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{MigrationsTable: table})
	if err != nil {
		return err
	}
	migrator, err := migrate.NewWithDatabaseInstance("embed://", table, driver)
	if err != nil {
		return err
	}

	err = migrator.Up()
	if err != nil && err.Error() == "no change" { // "no change" is not an error
		err = nil
	}
	return err
}
