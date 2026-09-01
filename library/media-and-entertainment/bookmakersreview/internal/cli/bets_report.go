// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source local

package cli

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/cliutil"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/store"

	"github.com/spf13/cobra"
)

type betReportGroup struct {
	Group        string  `json:"group"`
	Bets         int     `json:"bets"`
	Graded       int     `json:"graded"`
	AvgCLVPct    float64 `json:"avg_clv_pct"`
	PositiveCLV  int     `json:"positive_clv_bets"`
	Wins         int     `json:"wins,omitempty"`
	Losses       int     `json:"losses,omitempty"`
	Pushes       int     `json:"pushes,omitempty"`
	OutcomeKnown int     `json:"outcome_known,omitempty"`
}

func newNovelBetsReportCmd(flags *rootFlags) *cobra.Command {
	var flagSince string
	var flagGroupBy string
	var flagDB string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "See your running closing-line-value percentage and win rate across every recorded bet.",
		Long: "Aggregates every bet recorded via 'bets record' and graded via 'bets grade' into running CLV stats. " +
			"Win/loss/push rate is only reported for bets where you set --outcome at record time; CLV is computed " +
			"for every graded bet regardless.",
		Example:     "  bookmakersreview-pp-cli bets report --since 30d --group-by book --agent",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if flagGroupBy != "" && flagGroupBy != "none" && flagGroupBy != "market" && flagGroupBy != "book" {
				return usageErr(fmt.Errorf("--group-by must be one of: none, market, book"))
			}

			var since time.Duration
			if flagSince != "" {
				var err error
				since, err = cliutil.ParseDurationLoose(flagSince)
				if err != nil {
					return usageErr(fmt.Errorf("invalid --since duration %q: %w", flagSince, err))
				}
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

			args2 := []any{}
			whereClause := ""
			if since > 0 {
				whereClause = "WHERE placed_at >= ?"
				args2 = append(args2, time.Now().Add(-since).UnixMilli())
			}
			rows, err := db.DB().QueryContext(ctx, fmt.Sprintf(`
				SELECT market_id, book_name, outcome, clv_pct, graded_at
				FROM bmr_bets %s`, whereClause), args2...)
			if err != nil {
				return fmt.Errorf("querying bets: %w", err)
			}
			type rawBet struct {
				MarketID int
				BookName sql.NullString
				Outcome  sql.NullString
				CLVPct   sql.NullFloat64
				Graded   sql.NullInt64
			}
			raw := make([]rawBet, 0)
			for rows.Next() {
				var b rawBet
				if err := rows.Scan(&b.MarketID, &b.BookName, &b.Outcome, &b.CLVPct, &b.Graded); err != nil {
					_ = rows.Close()
					return fmt.Errorf("scanning bet row: %w", err)
				}
				raw = append(raw, b)
			}
			if err := rows.Err(); err != nil {
				_ = rows.Close()
				return fmt.Errorf("iterating bets: %w", err)
			}
			if err := rows.Close(); err != nil {
				return fmt.Errorf("closing bet rows: %w", err)
			}

			groupKey := func(b rawBet) string {
				switch flagGroupBy {
				case "market":
					return fmt.Sprintf("market %d", b.MarketID)
				case "book":
					if b.BookName.Valid && b.BookName.String != "" {
						return b.BookName.String
					}
					return "unknown book"
				default:
					return "all"
				}
			}

			groups := map[string]*betReportGroup{}
			for _, b := range raw {
				key := groupKey(b)
				g, ok := groups[key]
				if !ok {
					g = &betReportGroup{Group: key}
					groups[key] = g
				}
				g.Bets++
				if b.Graded.Valid && b.CLVPct.Valid {
					g.Graded++
					g.AvgCLVPct += b.CLVPct.Float64
					if b.CLVPct.Float64 > 0 {
						g.PositiveCLV++
					}
				}
				if b.Outcome.Valid {
					g.OutcomeKnown++
					switch b.Outcome.String {
					case "win":
						g.Wins++
					case "loss":
						g.Losses++
					case "push":
						g.Pushes++
					}
				}
			}
			result := make([]betReportGroup, 0, len(groups))
			for _, g := range groups {
				if g.Graded > 0 {
					g.AvgCLVPct /= float64(g.Graded)
				}
				result = append(result, *g)
			}

			if len(result) == 0 {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"groups": result,
					"note":   "no recorded bets found in this window — use 'bets record' to log one, then 'bets grade' once the market closes",
				}, flags)
			}
			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{"groups": result}, flags)
		},
	}
	cmd.Flags().StringVar(&flagSince, "since", "", "Only include bets placed within this window (e.g. 30d, 7d); default: all recorded bets")
	cmd.Flags().StringVar(&flagGroupBy, "group-by", "none", "Group results by: none, market, or book")
	cmd.Flags().StringVar(&flagDB, "db", "", "Database path (defaults to the standard local data directory)")
	return cmd
}
