// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	registerNovelCommand(func(root *cobra.Command, flags *rootFlags) {
		addNovelCommandIfAbsent(root, newWeatherCmd(flags))
	})
}

type weatherDay struct {
	Date        string  `json:"date"`
	MaxTempC    float64 `json:"maxtempC"`
	MaxTempF    float64 `json:"maxtempF"`
	MinTempC    float64 `json:"mintempC"`
	MinTempF    float64 `json:"mintempF"`
	TotalSnowCm float64 `json:"totalSnow_cm"`
	SunHour     float64 `json:"sunHour"`
	UVIndex     int     `json:"uvIndex"`
}

// eventWeather mirrors the real (introspection-confirmed after a live
// field-name miss) shape: Event.weather returns a WeatherOutput wrapper
// whose OWN "weather" field is the list of per-day forecast entries — the
// wrapper is not itself a per-day record. When there is no weather data,
// the resolver serializes the wrapper as an empty JSON array `[]` instead
// of `null` (confirmed live) — raw json.RawMessage plus manual dispatch
// below handles both shapes.
type eventWeather struct {
	EID     int             `json:"eid"`
	Weather json.RawMessage `json:"weather"`
}

// weatherDays extracts the per-day forecast list from an eventWeather's raw
// Weather field, tolerating the empty-array-instead-of-null quirk.
func weatherDays(raw json.RawMessage) ([]weatherDay, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}
	if trimmed[0] == '[' {
		// Empty-array-for-no-data quirk; a populated array in this
		// position has never been observed live and would not match the
		// WeatherOutput object shape regardless, so treat any array here
		// as "no data".
		return nil, nil
	}
	var wrapper struct {
		Days []weatherDay `json:"weather"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return nil, err
	}
	return wrapper.Days, nil
}

func newWeatherCmd(flags *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:         "weather",
		Short:       "weather subcommands: get",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE:        parentNoSubcommandRunE(flags),
	}
	cmd.AddCommand(newWeatherGetCmd(flags))
	return cmd
}

func newWeatherGetCmd(flags *rootFlags) *cobra.Command {
	var flagEvent int

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get forecasted weather for an outdoor event",
		Long: "Get forecasted weather for an event. A nil result is a legitimate empty result — most " +
			"indoor-sport or domed-venue events have no weather data, not an error.",
		Example:     "  bookmakersreview-pp-cli weather get --event 4802244 --json",
		Annotations: map[string]string{"mcp:read-only": "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if dryRunOK(flags) {
				return nil
			}
			if !cmd.Flags().Changed("event") {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--event is required"))
			}
			c, err := newBMRClient(flags)
			if err != nil {
				return err
			}
			ctx, cancel := boundCtx(cmd.Context(), flags)
			defer cancel()

			query := fmt.Sprintf(`query {
				events(eid: %s) {
					eid
					weather { weather { date maxtempC maxtempF mintempC mintempF totalSnow_cm sunHour uvIndex } }
				}
			}`, intLiteralList([]int{flagEvent}))
			var result struct {
				Events []eventWeather `json:"events"`
			}
			if err := c.Query(ctx, query, nil, &result); err != nil {
				return apiErr(err)
			}
			if len(result.Events) == 0 {
				return notFoundErr(fmt.Errorf("no event found with eid %d", flagEvent))
			}
			ev := result.Events[0]
			days, err := weatherDays(ev.Weather)
			if err != nil {
				return apiErr(fmt.Errorf("decoding weather data: %w", err))
			}
			if len(days) == 0 {
				if !wantsHumanTable(cmd.OutOrStdout(), flags) {
					return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
						"eid":     ev.EID,
						"note":    "no weather data for this event — likely an indoor sport or a domed venue",
						"weather": make([]weatherDay, 0),
					}, flags)
				}
				cmd.Println("no weather data for this event — likely an indoor sport or a domed venue")
				return nil
			}
			if !wantsHumanTable(cmd.OutOrStdout(), flags) {
				return printJSONFiltered(cmd.OutOrStdout(), map[string]any{
					"eid":     ev.EID,
					"weather": days,
				}, flags)
			}
			for _, d := range days {
				cmd.Printf("%s\thigh %.0fF/%.0fC low %.0fF/%.0fC\tUV %d\n",
					d.Date, d.MaxTempF, d.MaxTempC, d.MinTempF, d.MinTempC, d.UVIndex)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&flagEvent, "event", 0, "Event id (required)")
	return cmd
}
