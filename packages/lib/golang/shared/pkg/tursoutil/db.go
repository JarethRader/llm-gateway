package tursoutil

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func Connect(url, token string) (*sqlx.DB, error) {
	dsn := url
	if token != "" {
		dsn = fmt.Sprintf("%s?authToken=%s", url, token)
	}

	db, err := sqlx.Open("libsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}
