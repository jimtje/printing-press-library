// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"fmt"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newSportsCmd(flags))
	})
}

func newSportsCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "sports",
		Short:       "sports subcommands: list",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newSportsListCmd(flags))
	return cmd
}

type sportRow struct {
	SpID              int    `json:"spid"`
	Name              string `json:"nam"`
	Active            bool   `json:"act"`
	Enabled           bool   `json:"enabled"`
	DefaultMarketType *int   `json:"default_market_type,omitempty"`
	Order             int    `json:"ord"`
}

func newSportsListCmd(flags *rootFlags) *cobra.Command {
	var flagSiteID int
	var flagDomainID int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sports in BookmakersReview's own catalog",
		Long: "List sports via getSportsWithSettingsV2. The top-level 'sports' GraphQL field is a broken " +
			"federation passthrough on BookmakersReview's own backend (confirmed live: it unconditionally " +
			"errors demanding internal sitid/did context that isn't a usable public arg) — this command uses " +
			"getSportsWithSettingsV2 instead, which works correctly and returns each sport's spid alongside its " +
			"site-specific settings (enabled, default market type, display order). Defaults to sitid=5/did=1 " +
			"(BookmakersReview's own scope, matching 'sportsbooks list').",
		Example:     "  bookmakersreview-pp-cli sports list --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			var raw struct {
				Sports []struct {
					SpID     int    `json:"spid"`
					Name     string `json:"nam"`
					Active   bool   `json:"act"`
					Settings struct {
						Enabled bool `json:"enabled"`
						MTID    *int `json:"mtid"`
						Ord     int  `json:"ord"`
					} `json:"settings"`
				} `json:"getSportsWithSettingsV2"`
			}
			// Literal values inlined — see leagues.go's comment on why named
			// GraphQL $variables are avoided against this backend.
			query := fmt.Sprintf(`{getSportsWithSettingsV2(sitid:%d,did:%d){spid nam act settings{enabled mtid ord}}}`,
				flagSiteID, flagDomainID)
			if err := c.Query(ctx, query, nil, &raw); err != nil {
				return apiErr(err)
			}

			results := make([]sportRow, 0, len(raw.Sports))
			for _, s := range raw.Sports {
				results = append(results, sportRow{
					SpID:              s.SpID,
					Name:              s.Name,
					Active:            s.Active,
					Enabled:           s.Settings.Enabled,
					DefaultMarketType: s.Settings.MTID,
					Order:             s.Settings.Ord,
				})
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), results, flags)
			}
			for _, r := range results {
				cmd.Printf("%d\t%s\t(active=%t)\n", r.SpID, r.Name, r.Active)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagSiteID, "site-id", bmr.DefaultSiteID, "Site scope id (5 = BookmakersReview's own offshore catalog)")
	cmd.Flags().IntVar(&flagDomainID, "domain-id", bmr.DefaultDomainID, "Domain scope id (affiliate/tracking-link scope)")
	return cmd
}
