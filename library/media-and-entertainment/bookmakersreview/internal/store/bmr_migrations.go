// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package store

import "fmt"

// EnsureBetsTable lazily creates the local bet ledger used by `bets
// record`/`grade`/`report`. This is the user's own personal data (which
// wagers they placed and at what price) — no BookmakersReview API endpoint
// could ever provide it, so it only exists as a hand-authored local table.
func (s *Store) EnsureBetsTable() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS bmr_bets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	event_id INTEGER NOT NULL,
	market_id INTEGER NOT NULL,
	boid INTEGER NOT NULL,
	selection TEXT,
	price REAL NOT NULL,
	book_paid INTEGER NOT NULL,
	book_name TEXT,
	stake REAL,
	outcome TEXT,
	placed_at INTEGER NOT NULL,
	closing_price REAL,
	clv_pct REAL,
	graded_at INTEGER
);`
	if _, err := s.DB().Exec(ddl); err != nil {
		return fmt.Errorf("creating bmr_bets table: %w", err)
	}
	return nil
}
