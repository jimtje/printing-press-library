// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		oddsCmd, _, err := root.Find([]string{"odds"})
		if err != nil {
			return
		}
		addNovelCommandIfAbsent(oddsCmd, newOddsCurrentCmd(flags))
		addNovelCommandIfAbsent(oddsCmd, newOddsOpeningCmd(flags))
		addNovelCommandIfAbsent(oddsCmd, newOddsBestCmd(flags))
		addNovelCommandIfAbsent(oddsCmd, newOddsLiveCmd(flags))
	})
}

// fetchAndEnrichLines runs one of the current/opening/live/history line
// queries (they all share the same argument and return shape) and resolves
// boid/paid into human-readable selection and book names.
func fetchAndEnrichLines(cmd *cobra.Command, flags *rootFlags, queryField string, eid, mtid int, books []int) ([]enrichedLine, error) {
	ctx, cancel := boundCtx(cmd.Context(), flags)
	defer cancel()

	c, err := newBMRClient(flags)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`query { %s(eid: [%d], mtid: [%d], paid: %s) }`, queryField, eid, mtid, intListLiteral(books))
	var resp map[string][]bmr.Line
	if err := c.Query(ctx, query, nil, &resp); err != nil {
		return nil, apiErr(err)
	}

	boidNames, err := resolveBettingOptionNames(ctx, c, eid, mtid)
	if err != nil {
		// Non-fatal: still show prices, just without resolved selection names.
		boidNames = map[int]string{}
	}
	paidNames, err := resolveSportsbookNames(ctx, c, bmr.DefaultSiteID, bmr.DefaultDomainID)
	if err != nil {
		paidNames = map[int]string{}
	}

	return enrichLines(resp[queryField], boidNames, paidNames), nil
}

func newOddsCurrentCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int
	var flagBooks string

	cmd := &cobra.Command{
		Use:         "current",
		Short:       "Current odds for one event and market, across sportsbooks",
		Long:        "Use this to see live current prices for a specific event+market across the sportsbooks you name. Do NOT use it to compare across every book automatically; use 'odds best' for that.",
		Example:     "  bookmakersreview-pp-cli odds current --event 4802244 --market 1 --books 9,20,8 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") || !cmd.Flags().Changed("market") || !cmd.Flags().Changed("books") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event, --market, and --books are required"))
			}
			books, err := parseCSVInts(flagBooks)
			if err != nil {
				return usageErr(err)
			}
			lines, err := fetchAndEnrichLines(cmd, flags, "currentLines", flagEvent, flagMarket, books)
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), lines, flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	cmd.Flags().StringVar(&flagBooks, "books", "", "Comma-separated sportsbook provider-account ids (see 'sportsbooks list')")
	return cmd
}

func newOddsOpeningCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int
	var flagBooks string

	cmd := &cobra.Command{
		Use:         "opening",
		Short:       "Opening odds for one event and market, across sportsbooks",
		Long:        "Use this to see the first posted prices for a specific event+market. Compare against 'odds current' to see how far the line has moved.",
		Example:     "  bookmakersreview-pp-cli odds opening --event 4802244 --market 1 --books 9,20,8 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") || !cmd.Flags().Changed("market") || !cmd.Flags().Changed("books") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event, --market, and --books are required"))
			}
			books, err := parseCSVInts(flagBooks)
			if err != nil {
				return usageErr(err)
			}
			lines, err := fetchAndEnrichLines(cmd, flags, "openingLines", flagEvent, flagMarket, books)
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), lines, flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	cmd.Flags().StringVar(&flagBooks, "books", "", "Comma-separated sportsbook provider-account ids (see 'sportsbooks list')")
	return cmd
}

func newOddsLiveCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int
	var flagBooks string

	cmd := &cobra.Command{
		Use:         "live",
		Short:       "Live in-play odds for one event and market, across sportsbooks",
		Long:        "Use this only for events that are currently in progress. For pre-game prices use 'odds current'.",
		Example:     "  bookmakersreview-pp-cli odds live --event 4802244 --market 1 --books 9,20,8 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") || !cmd.Flags().Changed("market") || !cmd.Flags().Changed("books") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event, --market, and --books are required"))
			}
			books, err := parseCSVInts(flagBooks)
			if err != nil {
				return usageErr(err)
			}
			lines, err := fetchAndEnrichLines(cmd, flags, "liveLines", flagEvent, flagMarket, books)
			if err != nil {
				return err
			}
			return printJSONFiltered(cmd.OutOrStdout(), lines, flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	cmd.Flags().StringVar(&flagBooks, "books", "", "Comma-separated sportsbook provider-account ids (see 'sportsbooks list')")
	return cmd
}

func newOddsBestCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int
	var flagMarket int
	var flagBooks string

	cmd := &cobra.Command{
		Use:         "best",
		Short:       "Best available price per selection across every tracked sportsbook",
		Long:        "Use this to find the highest price for each side of a market across books. It does not check whether that price is actually good value versus the fair line; use 'odds value' for that.",
		Example:     "  bookmakersreview-pp-cli odds best --event 4802244 --market 1 --json",
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
				return apiErr(err)
			}
			boidNames, _ := resolveBettingOptionNames(ctx, c, flagEvent, flagMarket)
			paidNames, _ := resolveSportsbookNames(ctx, c, bmr.DefaultSiteID, bmr.DefaultDomainID)
			lines := enrichLines(resp["bestLines"], boidNames, paidNames)
			return printJSONFiltered(cmd.OutOrStdout(), lines, flags)
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (see 'events list'/'events get')")
	cmd.Flags().IntVar(&flagMarket, "market", 0, "Market type id (see 'markets list'; 1=moneyline/Winner, 2=total, 3=spread/Handicap)")
	cmd.Flags().StringVar(&flagBooks, "books", "", "Comma-separated sportsbook provider-account ids to compare (default: BookmakersReview's full tracked list)")
	return cmd
}
