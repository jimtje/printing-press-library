// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

type arbLeg struct {
	Selection string  `json:"selection"`
	Book      string  `json:"book"`
	Price     float64 `json:"price"`
	American  int     `json:"american"`
}

type arbResult struct {
	Legs               []arbLeg `json:"legs"`
	CombinedImpliedPct float64  `json:"combined_implied_probability_pct"`
	ArbitrageExists    bool     `json:"arbitrage_exists"`
	ProfitMarginPct    float64  `json:"profit_margin_pct,omitempty"`
}

func newNovelArbScanCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int
	var flagBooks string

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Find risk-free two-sided price mismatches across sportsbooks for one event.",
		Long: "Takes the single best price for every selection in a market (across the books you name, or " +
			"BookmakersReview's full tracked list by default) and checks whether their combined implied probability " +
			"is under 100% — a guaranteed-profit spread regardless of outcome. Use this to find risk-free arbitrage " +
			"across books. Do NOT use it to evaluate single-side value against fair odds; use 'odds value' for that.",
		Example:     "  bookmakersreview-pp-cli arb scan --event 4802244 --market 1 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") || !cmd.Flags().Changed("market") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event and --market are required"))
			}
			books := defaultBookPAIDs
			if cmd.Flags().Changed("books") {
				var err error
				books, err = parseCSVInts(flagBooks)
				if err != nil {
					return usageErr(err)
				}
			}

			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}

			query := fmt.Sprintf(`query { bestLines(eid: [%d], mtid: [%d], paid: %s) }`, flagEvent, flagMarket, intListLiteral(books))
			var resp map[string][]bmr.Line
			if err := c.Query(ctx, query, nil, &resp); err != nil {
				return apiErr(fmt.Errorf("fetching best lines: %w", err))
			}
			bestLines := resp["bestLines"]
			if len(bestLines) == 0 {
				return apiErr(fmt.Errorf("no lines found for event %d market %d across the requested books", flagEvent, flagMarket))
			}

			boidNames, _ := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)
			paidNames, _ := resolveSportsbookNames(ctx, c, bmr.DefaultSiteID, bmr.DefaultDomainID)

			legs := make([]arbLeg, 0, len(bestLines))
			var combined float64
			for _, l := range bestLines {
				if l.Price <= 0 {
					continue
				}
				legs = append(legs, arbLeg{
					Selection: boidNames[l.BOID],
					Book:      paidNames[l.PAID],
					Price:     l.Price,
					American:  l.American,
				})
				combined += impliedProbability(l.Price)
			}
			if len(legs) < 2 {
				return apiErr(fmt.Errorf("need at least 2 priced selections to check for arbitrage; found %d", len(legs)))
			}

			result := arbResult{
				Legs:               legs,
				CombinedImpliedPct: combined * 100,
				ArbitrageExists:    combined < 1,
			}
			if result.ArbitrageExists {
				result.ProfitMarginPct = (1/combined - 1) * 100
			}

			return printJSONFiltered(cmd.OutOrStdout(), result, flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	cmd.Flags().StringVar(&flagBooks, "books", "", "Comma-separated sportsbook provider-account ids to check (default: BookmakersReview's full tracked list)")
	return cmd
}
