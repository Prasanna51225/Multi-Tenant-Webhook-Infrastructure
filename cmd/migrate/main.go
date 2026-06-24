package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	direction := flag.String("direction", "up", "migration direction: up or down")
	dbURL := flag.String("db-url", "postgres://webhook:webhook@localhost:5432/webhook?sslmode=disable", "database url")
	migrationsPath := flag.String("migrations", "file://migrations", "migrations path")
	flag.Parse()

	m, err := migrate.New(*migrationsPath, *dbURL)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	switch *direction {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run migrations up: %v", err)
		}
		fmt.Println("migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run migrations down: %v", err)
		}
		fmt.Println("migrations rolled back successfully")
	default:
		log.Fatalf("invalid direction: %s (use 'up' or 'down')", *direction)
	}

	_, _ = m.Close()
}
