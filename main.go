//go:build ignore

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ninesl/scryball"
)

func main() {
	var dbPath string
	var jsonPath string

	flag.StringVar(&dbPath, "db-path", "scryball.db", "Path to SQLite output database")
	flag.StringVar(&jsonPath, "json-file", "", "Path to downloaded Scryfall cards JSON file")
	flag.Parse()

	if jsonPath == "" {
		log.Fatal("missing required --json-file")
	}

	sb, err := scryball.NewWithConfig(scryball.ScryballConfig{DBPath: dbPath})
	if err != nil {
		log.Fatalf("failed to initialize scryball: %v", err)
	}
	defer sb.RetrieveDB().Close()

	start := time.Now()
	stats, err := sb.ImportCardsJSONFile(context.Background(), jsonPath)
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	fmt.Printf("Import complete in %s\n", time.Since(start).Round(time.Millisecond))
	fmt.Printf("Decoded cards:      %d\n", stats.DecodedCards)
	fmt.Printf("Imported cards:     %d\n", stats.ImportedCards)
	fmt.Printf("Imported printings: %d\n", stats.ImportedPrintings)
	fmt.Printf("Skipped cards:      %d\n", stats.SkippedCards)
	fmt.Printf("Batches committed:  %d\n", stats.BatchesCommitted)
	fmt.Printf("Database path:      %s\n", dbPath)

	if err := verifyDatabase(sb.RetrieveDB().DB); err != nil {
		log.Fatalf("database verification failed: %v", err)
	}

	fmt.Println("Database verification: ok")
}

func verifyDatabase(db *sql.DB) error {
	var integrity string
	if err := db.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		return fmt.Errorf("run integrity_check: %w", err)
	}

	if integrity != "ok" {
		return fmt.Errorf("integrity_check failed: %s", integrity)
	}

	return nil
}
