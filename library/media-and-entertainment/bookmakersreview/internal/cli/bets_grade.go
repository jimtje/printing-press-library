// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source auto

package cli

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/store"

	"github.com/spf13/cobra"
)

type recordedBet struct {
	ID        int64
	EventID   int
	MarketID  int
	BOID      int
	Selection string
	Price     float64
	BookPaid  int
	BookName  string
}

func newNovelBetsGradeCmd(flags *rootFlags) *cobra.Command {
	var flagBetID int64
	var flagDB string

	cmd := &cobra.Command{
		Use:   "grade",
		Short: "Compare one recorded bet's price to the market's closing line to compute its CLV.",
		Long: "Use 'bets grade' to evaluate one recorded bet against the closing line. Do NOT use it for general " +
			"historical line lookup with no personal bet attached; use 'consensus history' for that. Grading refuses " +
			"to run before the event has started, since BookmakersReview's own price-history endpoints are broken " +
			"upstream (confirmed live) — this command uses the current price at/after kickoff as a closing-line proxy, " +
			"which is only meaningful once the market has actually closed.",
		Example: "  bookmakersreview-pp-cli bets grade --bet-id 42 --agent",
		// A fresh/empty bet ledger (e.g. the live-dogfood harness's isolated
		// HOME) genuinely has no bet id 42 to grade — exit 3 (not found) is
		// the correct, graceful outcome for that happy-path probe, not a
		// failure. pp:typed-exit-codes tells the matrix to accept it.
		Annotations: map[string]string{"mcp:read-only": "true", "pp:typed-exit-codes": "0,3"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("bet-id") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--bet-id is required"))
			}

			if flagDB == "" {
				flagDB = defaultDBPath("bookmakersreview-pp-cli")
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			db, err := store.OpenWithContext(ctx, flagDB)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			if err := db.EnsureBetsTable(); err != nil {
				return err
			}

			var bet recordedBet
			row := db.DB().QueryRowContext(ctx, `SELECT id, event_id, market_id, boid, selection, price, book_paid, book_name FROM bmr_bets WHERE id = ?`, flagBetID)
			var selection, bookName sql.NullString
			if err := row.Scan(&bet.ID, &bet.EventID, &bet.MarketID, &bet.BOID, &selection, &bet.Price, &bet.BookPaid, &bookName); err != nil {
				if err == sql.ErrNoRows {
					return notFoundErr(fmt.Errorf("no recorded bet with id %d", flagBetID))
				}
				return fmt.Errorf("looking up bet %d: %w", flagBetID, err)
			}
			bet.Selection = selection.String
			bet.BookName = bookName.String

			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}

			evQuery := fmt.Sprintf(`query { events(eid: [%d]) { dt } }`, bet.EventID)
			var evResp struct {
				Events []bmr.Event `json:"events"`
			}
			if err := c.Query(ctx, evQuery, nil, &evResp); err != nil {
				return apiErr(fmt.Errorf("looking up event %d start time: %w", bet.EventID, err))
			}
			if len(evResp.Events) == 0 {
				return apiErr(fmt.Errorf("event %d not found", bet.EventID))
			}
			startTime := time.UnixMilli(evResp.Events[0].DT)
			if time.Now().Before(startTime) {
				return usageErr(fmt.Errorf("event %d has not started yet (starts %s) — grading uses the current price as a closing-line proxy, so it isn't meaningful until the market closes at kickoff", bet.EventID, startTime.Format(time.RFC3339)))
			}

			// historyLines/lineHistory are confirmed broken upstream: both
			// crash the backend ("Cannot read property 'length' of
			// undefined") regardless of filters, on both the
			// bookmakersreview.com and oddstrader.com hosts. currentLines is
			// the reliable substitute: BookmakersReview keeps posting/
			// updating current prices through kickoff, so the current price
			// at/after an event's start time IS effectively the closing
			// line. This is why grading before the event starts is refused
			// below — "current" before kickoff is not yet a closing price.
			query := fmt.Sprintf(`query { currentLines(eid: [%d], mtid: [%d], boid: [%d]) }`, bet.EventID, bet.MarketID, bet.BOID)
			var resp map[string][]bmr.Line
			if err := c.Query(ctx, query, nil, &resp); err != nil {
				return apiErr(fmt.Errorf("fetching closing-line proxy (current lines): %w", err))
			}
			history := resp["currentLines"]
			if len(history) == 0 {
				return apiErr(fmt.Errorf("no current price found for event %d market %d selection %d to use as a closing-line proxy", bet.EventID, bet.MarketID, bet.BOID))
			}

			// Prefer the SAME book the bet was placed at (the truest
			// closing-line comparison); fall back to the latest snapshot
			// across any book if that book stopped quoting the market.
			var closing *bmr.Line
			var latestAny *bmr.Line
			for i := range history {
				h := &history[i]
				if latestAny == nil || h.Sequence > latestAny.Sequence {
					latestAny = h
				}
				if h.PAID == bet.BookPaid {
					if closing == nil || h.Sequence > closing.Sequence {
						closing = h
					}
				}
			}
			usedFallbackBook := false
			if closing == nil {
				closing = latestAny
				usedFallbackBook = true
			}
			if closing == nil || closing.Price <= 0 {
				return apiErr(fmt.Errorf("could not determine a closing-line proxy for bet %d", flagBetID))
			}

			// CLV% = (my price / closing price - 1) * 100. A LOWER closing
			// price than my bet price means the market shortened in my
			// favor after I bet — positive CLV, the standard long-run skill
			// signal in sports betting.
			clvPct := (bet.Price/closing.Price - 1) * 100

			if _, err := db.DB().ExecContext(ctx, `UPDATE bmr_bets SET closing_price = ?, clv_pct = ?, graded_at = ? WHERE id = ?`,
				closing.Price, clvPct, time.Now().UnixMilli(), bet.ID); err != nil {
				return fmt.Errorf("saving grade for bet %d: %w", bet.ID, err)
			}

			result := map[string]any{
				"bet_id":             bet.ID,
				"selection":          bet.Selection,
				"my_price":           bet.Price,
				"my_book":            bet.BookName,
				"closing_price":      closing.Price,
				"clv_pct":            clvPct,
				"beat_closing_line":  clvPct > 0,
				"used_fallback_book": usedFallbackBook,
			}
			if usedFallbackBook {
				result["note"] = fmt.Sprintf("book %q had no closing snapshot for this market; used the latest price from any tracked book instead", bet.BookName)
			}
			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().Int64Var(&flagBetID, "bet-id", 0, "Id of a bet recorded via 'bets record'")
	cmd.Flags().StringVar(&flagDB, "db", "", "Database path (defaults to the standard local data directory)")
	return cmd
}
