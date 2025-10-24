package sqlconnect

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func ConnectDB() (*sql.DB, error) {

	connStr := os.Getenv("SQL_CONNECT")

	if connStr == "" {
		return nil, fmt.Errorf("error getting sql url")
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	return db, nil

}
