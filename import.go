package scryball

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/ninesl/scryball/internal/client"
)

const defaultImportBatchSize = 2000

// ImportStats tracks JSON import progress and results.
type ImportStats struct {
	DecodedCards      int64
	ImportedCards     int64
	ImportedPrintings int64
	SkippedCards      int64
	BatchesCommitted  int64
	SkippedDetails    []SkippedCard
}

// SkippedCard captures why a card from bulk JSON was skipped.
type SkippedCard struct {
	Name   string
	ID     string
	Reason string
}

// ImportCardsJSONFile imports cards from a local JSON array file into this instance.
func (s *Scryball) ImportCardsJSONFile(ctx context.Context, filePath string) (ImportStats, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return ImportStats{}, fmt.Errorf("open JSON file: %w", err)
	}
	defer f.Close()

	return s.ImportCardsJSON(ctx, f)
}

// ImportCardsJSON imports cards from a JSON array stream into this instance.
func (s *Scryball) ImportCardsJSON(ctx context.Context, r io.Reader) (ImportStats, error) {
	return s.importCardsJSONWithBatchSize(ctx, r, defaultImportBatchSize)
}

// ImportCardsJSONFile imports cards from a local JSON array file into the global instance.
func ImportCardsJSONFile(ctx context.Context, filePath string) (ImportStats, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return ImportStats{}, err
	}

	return sb.ImportCardsJSONFile(ctx, filePath)
}

// ImportCardsJSON imports cards from a JSON array stream into the global instance.
func ImportCardsJSON(ctx context.Context, r io.Reader) (ImportStats, error) {
	sb, err := ensureCurrentScryball()
	if err != nil {
		return ImportStats{}, err
	}

	return sb.ImportCardsJSON(ctx, r)
}

func (s *Scryball) importCardsJSONWithBatchSize(ctx context.Context, r io.Reader, batchSize int) (ImportStats, error) {
	if batchSize <= 0 {
		batchSize = defaultImportBatchSize
	}
	_ = batchSize

	dec := json.NewDecoder(bufio.NewReader(r))

	startToken, err := dec.Token()
	if err != nil {
		return ImportStats{}, fmt.Errorf("read start token: %w", err)
	}

	startDelim, ok := startToken.(json.Delim)
	if !ok || startDelim != '[' {
		return ImportStats{}, fmt.Errorf("expected JSON array at top level")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportStats{}, fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	qtx := s.queries.WithTx(tx)
	stats := ImportStats{}
	seenOracleIDs := make(map[string]struct{})
	hasWrites := false

	for dec.More() {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}

		var apiCard client.Card
		if err := dec.Decode(&apiCard); err != nil {
			return stats, fmt.Errorf("decode card JSON: %w", err)
		}

		stats.DecodedCards++

		cardParams, printingParams, err := convertAPICardToDBParams(&apiCard)
		if err != nil {
			stats.SkippedCards++
			stats.SkippedDetails = append(stats.SkippedDetails, SkippedCard{
				Name:   apiCard.Name,
				ID:     apiCard.ID,
				Reason: err.Error(),
			})
			continue
		}

		if err := qtx.UpsertCard(ctx, cardParams); err != nil {
			return stats, fmt.Errorf("upsert card %q (%s): %w", apiCard.Name, cardParams.OracleID, err)
		}

		if err := qtx.UpsertPrinting(ctx, printingParams); err != nil {
			return stats, fmt.Errorf("upsert printing %q (%s): %w", apiCard.Name, printingParams.ID, err)
		}

		if _, seen := seenOracleIDs[cardParams.OracleID]; !seen {
			seenOracleIDs[cardParams.OracleID] = struct{}{}
		}

		stats.ImportedPrintings++
		hasWrites = true
	}

	endToken, err := dec.Token()
	if err != nil {
		return stats, fmt.Errorf("read end token: %w", err)
	}

	endDelim, ok := endToken.(json.Delim)
	if !ok || endDelim != ']' {
		return stats, fmt.Errorf("expected closing JSON array delimiter")
	}

	if hasWrites {
		if err := tx.Commit(); err != nil {
			return stats, fmt.Errorf("commit transaction: %w", err)
		}
		committed = true
		stats.BatchesCommitted = 1

		if _, err := s.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			return stats, fmt.Errorf("checkpoint WAL: %w", err)
		}

		if _, err := s.db.ExecContext(ctx, "PRAGMA journal_mode=DELETE"); err != nil {
			return stats, fmt.Errorf("set DELETE journal mode: %w", err)
		}
	} else {
		committed = true
	}

	stats.ImportedCards = int64(len(seenOracleIDs))

	return stats, nil
}
