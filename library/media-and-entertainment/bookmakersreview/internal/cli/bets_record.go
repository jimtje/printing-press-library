// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source local

package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/store"

	"github.com/spf13/cobra"
)

func newNovelBetsRecordCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int
	var flagBOID int
	var flagSelection string
	var flagPrice float64
	var flagBook int
	var flagStake float64
	var flagOutcome string
	var flagDB string

	cmd := &cobra.Command{
		Use:   "record",
		Short: "Log your own wager (event, market, price, book, timestamp) to a local ledger.",
		Long: "Records a personal bet to the local database for later closing-line-value grading via 'bets grade'. " +
			"This is your own data, not BookmakersReview's — no API endpoint provides it.",
		Example:     "  bookmakersreview-pp-cli bets record --event 4802244 --market 1 --selection \"Southampton FC\" --price 2.5 --book 9 --stake 50",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") || !cmd.Flags().Changed("market") || !cmd.Flags().Changed("price") || !cmd.Flags().Changed("book") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event, --market, --price, and --book are required"))
			}
			if !cmd.Flags().Changed("boid") && !cmd.Flags().Changed("selection") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("one of --boid or --selection is required to identify which side you bet"))
			}
			if flagOutcome != "" && flagOutcome != "win" && flagOutcome != "loss" && flagOutcome != "push" {
				return usageErr(fmt.Errorf("--outcome must be one of: win, loss, push"))
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}

			boid := flagBOID
			selectionName := flagSelection
			if !cmd.Flags().Changed("boid") {
				names, err := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)
				if err != nil {
					return apiErr(fmt.Errorf("resolving --selection to a betting option id: %w", err))
				}
				matches := 0
				for id, name := range names {
					if strings.EqualFold(name, flagSelection) {
						boid = id
						selectionName = name
						matches++
					}
				}
				if matches == 0 {
					return usageErr(fmt.Errorf("no betting option named %q found for event %d market %d — check 'events get %d' or pass --boid directly", flagSelection, flagEvent, flagMarket, flagEvent))
				}
				if matches > 1 {
					return usageErr(fmt.Errorf("more than one betting option matched %q — pass --boid directly", flagSelection))
				}
			} else {
				names, err := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)
				if err == nil {
					selectionName = names[boid]
				}
			}

			bookName := ""
			if paidNames, err := resolveSportsbookNames(ctx, c, bmr.DefaultSiteID, bmr.DefaultDomainID); err == nil {
				bookName = paidNames[flagBook]
			}

			if flagDB == "" {
				flagDB = defaultDBPath("bookmakersreview-pp-cli")
			}
			db, err := store.OpenWithContext(ctx, flagDB)
			if err != nil {
				return fmt.Errorf("opening database: %w", err)
			}
			defer db.Close()
			if err := db.EnsureBetsTable(); err != nil {
				return err
			}

			res, err := db.DB().ExecContext(ctx, `
				INSERT INTO bmr_bets (event_id, market_id, boid, selection, price, book_paid, book_name, stake, outcome, placed_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				flagEvent, flagMarket, boid, selectionName, flagPrice, flagBook, bookName, nullableFloat(flagStake, cmd.Flags().Changed("stake")), nullableString(flagOutcome), time.Now().UnixMilli(),
			)
			if err != nil {
				return fmt.Errorf("recording bet: %w", err)
			}
			id, _ := res.LastInsertId()

			return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
				"bet_id":    id,
				"event_id":  flagEvent,
				"market_id": flagMarket,
				"selection": selectionName,
				"price":     flagPrice,
				"book":      bookName,
				"stake":     flagStake,
			}, flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id you bet on (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id you bet on (see 'markets list')")
	cmd.Flags().IntVar(&flagBOID, "boid", 0, "Betting option id you bet (see 'events get' or bettingOptionsByEvent); alternative to --selection")
	cmd.Flags().StringVar(&flagSelection, "selection", "", "Selection name you bet, e.g. a team name or \"Draw\" (resolved to a boid automatically); alternative to --boid")
	cmd.Flags().Float64Var(&flagPrice, "price", 0, "Decimal price you got (e.g. 2.5)")
	cmd.Flags().IntVar(&flagBook, "book", 0, "Sportsbook provider-account id you bet at (see 'sportsbooks list')")
	cmd.Flags().Float64Var(&flagStake, "stake", 0, "Optional stake amount, for your own records only")
	cmd.Flags().StringVar(&flagOutcome, "outcome", "", "Optional self-reported result once known: win, loss, or push (used by 'bets report' for win-rate stats)")
	cmd.Flags().StringVar(&flagDB, "db", "", "Database path (defaults to the standard local data directory)")
	return cmd
}

func nullableFloat(v float64, changed bool) any {
	if !changed {
		return nil
	}
	return v
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}
