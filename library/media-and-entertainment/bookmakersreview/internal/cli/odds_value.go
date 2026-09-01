// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// pp:data-source live

package cli

import (
	"fmt"
	"sort"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

// valueRow is one book's price on one selection, judged against the
// vig-free fair-value probability derived for that selection.
type valueRow struct {
	Selection          string  `json:"selection"`
	Book               string  `json:"book"`
	Price              float64 `json:"price"`
	American           int     `json:"american"`
	FairProbabilityPct float64 `json:"fair_probability_pct"`
	ExpectedValuePct   float64 `json:"expected_value_pct"`
	PositiveEV         bool    `json:"positive_ev"`
}

func newNovelOddsValueCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int
	var flagBooks string

	cmd := &cobra.Command{
		Use:   "value",
		Short: "See which sportsbook's current price beats the vig-free fair line before you bet.",
		Long: "Devigs the market's own average current price per selection to estimate a fair (vig-free) " +
			"win probability, then compares every book's individual price against it. " +
			"This is the standard 'average the market, remove the overround' devig method, used when no single " +
			"designated sharp book (e.g. Pinnacle) has a price on this particular market. " +
			"Use this to find whether a price is actually positive-EV. Do NOT use it to just find the numerically " +
			"highest price across books; use 'odds best' for that.",
		Example:     "  bookmakersreview-pp-cli odds value --event 4802244 --market 1 --agent",
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

			linesQuery := fmt.Sprintf(`query { currentLines(eid: [%d], mtid: [%d], paid: %s) }`, flagEvent, flagMarket, intListLiteral(books))
			var linesResp map[string][]bmr.Line
			if err := c.Query(ctx, linesQuery, nil, &linesResp); err != nil {
				return apiErr(fmt.Errorf("fetching current lines: %w", err))
			}
			lines := linesResp["currentLines"]
			if len(lines) == 0 {
				return apiErr(fmt.Errorf("no current lines for event %d market %d across the requested books", flagEvent, flagMarket))
			}

			// Devig by averaging the market's own prices per selection, then
			// normalizing implied probabilities to remove the overround
			// (vig). This is the standard fallback devig method when no
			// single designated sharp book (e.g. Pinnacle) has a price on
			// this particular market — confirmed live that Pinnacle does
			// not cover every event BookmakersReview tracks.
			priceSum := map[int]float64{}
			priceCount := map[int]int{}
			for _, l := range lines {
				if l.Price <= 0 {
					continue
				}
				priceSum[l.BOID] += l.Price
				priceCount[l.BOID]++
			}

			impliedProb := make(map[int]float64, len(priceSum))
			var overround float64
			for boid, sum := range priceSum {
				avgPrice := sum / float64(priceCount[boid])
				p := impliedProbability(avgPrice)
				impliedProb[boid] = p
				overround += p
			}
			if overround <= 0 {
				return apiErr(fmt.Errorf("could not compute a devigged fair line for event %d market %d", flagEvent, flagMarket))
			}
			fairProb := make(map[int]float64, len(impliedProb))
			for boid, p := range impliedProb {
				fairProb[boid] = p / overround
			}

			boidNames, _ := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)
			paidNames, _ := resolveSportsbookNames(ctx, c, bmr.DefaultSiteID, bmr.DefaultDomainID)

			rows := make([]valueRow, 0, len(lines))
			for _, line := range lines {
				fp, known := fairProb[line.BOID]
				if !known || line.Price <= 0 {
					continue
				}
				// EV fraction = fair probability * decimal price - 1.
				ev := fp*line.Price - 1
				rows = append(rows, valueRow{
					Selection:          boidNames[line.BOID],
					Book:               paidNames[line.PAID],
					Price:              line.Price,
					American:           line.American,
					FairProbabilityPct: fp * 100,
					ExpectedValuePct:   ev * 100,
					PositiveEV:         ev > 0,
				})
			}
			sort.Slice(rows, func(i, j int) bool { return rows[i].ExpectedValuePct > rows[j].ExpectedValuePct })

			return printJSONFiltered(cmd.OutOrStdout(), rows, flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	cmd.Flags().StringVar(&flagBooks, "books", "", "Comma-separated sportsbook provider-account ids to check (default: BookmakersReview's full tracked list)")
	return cmd
}
