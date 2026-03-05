package main

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"
)

func openDB() (*sql.DB, error) {
	return sql.Open("postgres", os.Getenv("DATABASE_URL"))
}
