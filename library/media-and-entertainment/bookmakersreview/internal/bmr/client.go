// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

// Package bmr wraps the generated internal/client.Client to speak GraphQL
// against BookmakersReview's odds-v2 service. This is a hand-written,
// non-generated package: the upstream API is GraphQL-only, so there is no
// OpenAPI surface for the generator to emit typed endpoint commands from.
//
// Rate limiting: every outbound HTTP call in this package goes through
// client.Client.Post (internal/client), which already owns a full
// cliutil.AdaptiveLimiter with 429 detection, adaptive backoff, and
// Retry-After handling (see internal/client/client.go). This package makes
// no raw net/http calls of its own and intentionally does not layer a
// second, independent limiter on top of the generated one — doing so would
// double-throttle requests without adding any real protection.
package bmr

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/client"
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/cliutil"
)

// FallbackBaseURL is a sibling hostname (OddsTrader, another Better
// Collective property) confirmed live to serve the identical backend and
// identical sitid-scoped sportsbook catalogs. Used as an automatic failover
// when the primary bookmakersreview.com host is unreachable at the network
// level.
const FallbackBaseURL = "https://ms.virginia.us-east-1.oddstrader.com/odds-v2/odds-v2-service"

// DefaultSiteID and DefaultDomainID scope every sitid/did-bearing query to
// BookmakersReview's own catalog. Confirmed live: sitid=5 returns exactly 26
// classic offshore sportsbooks (Bovada, MyBookie, BetOnline, Pinnacle, ...)
// with zero US-regulated-book noise, matching BookmakersReview's own
// "offshore sportsbooks" identity; sitid 1-4 return sibling-property catalogs
// (256/434/535/526 books, including state-licensed variants like
// "DraftKingsNJ"). did 1-5 did not change the sitid=5 book count, so did is
// most likely an affiliate/tracking-link scope rather than a catalog filter.
const (
	DefaultSiteID   = 5
	DefaultDomainID = 1
)

// GraphQLError mirrors one entry of a GraphQL response's top-level "errors"
// array. The upstream federation gateway sometimes nests proxy diagnostics
// inside "path" rather than "message", so Error() falls back to the raw
// path when message is empty.
type GraphQLError struct {
	Message string          `json:"message"`
	Path    json.RawMessage `json:"path,omitempty"`
}

func (e GraphQLError) String() string {
	if strings.TrimSpace(e.Message) != "" {
		return e.Message
	}
	if len(e.Path) > 0 {
		return string(e.Path)
	}
	return "unknown graphql error"
}

// QueryError wraps one or more GraphQL-level errors returned alongside HTTP
// 200 (GraphQL reports application errors in the response body, not via
// status code).
type QueryError struct {
	Errors []GraphQLError
}

func (e *QueryError) Error() string {
	parts := make([]string, len(e.Errors))
	for i, ge := range e.Errors {
		parts[i] = ge.String()
	}
	return "bmr graphql error: " + strings.Join(parts, "; ")
}

type envelope struct {
	Data   json.RawMessage `json:"data"`
	Errors []GraphQLError  `json:"errors,omitempty"`
}

// Client executes GraphQL operations against the odds-v2 service through the
// generated HTTP transport (so timeout, dry-run, and transport selection
// stay consistent with every other command).
type Client struct {
	http *client.Client
}

// New wraps an already-constructed generated HTTP client.
func New(c *client.Client) *Client {
	return &Client{http: c}
}

// Query executes a GraphQL query/mutation and decodes the response's "data"
// object into out (typically a struct with json tags matching the queried
// top-level field names). Pass a nil out to execute for side effects/error
// checking only.
func (c *Client) Query(ctx context.Context, query string, variables map[string]any, out any) error {
	raw, err := c.RawQuery(ctx, query, variables)
	if err != nil {
		return err
	}
	if out == nil || len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decoding bmr graphql data: %w", err)
	}
	return nil
}

// RawQuery executes a GraphQL query/mutation and returns the raw "data"
// object for callers that want to decode selectively (e.g. --select support
// on raw passthrough output).
func (c *Client) RawQuery(ctx context.Context, query string, variables map[string]any) (json.RawMessage, error) {
	body := map[string]any{"query": query}
	if len(variables) > 0 {
		body["variables"] = variables
	}

	raw, _, err := c.http.Post(ctx, "/", body)
	if err != nil {
		// Network-level failure (DNS/connection/TLS) against the primary
		// host: retry once against the confirmed-identical OddsTrader
		// sibling host before giving up. A GraphQL-level error (HTTP 200
		// with an "errors" body) is not retried here — it already reached
		// the server and got an answer.
		original := c.http.BaseURL
		if original != FallbackBaseURL {
			c.http.BaseURL = FallbackBaseURL
			raw, _, err = c.http.Post(ctx, "/", body)
			c.http.BaseURL = original
		}
	}
	if err != nil {
		// The generated client.Client already retries 429s internally with
		// its own AdaptiveLimiter (see package doc comment above) and only
		// returns here once retries are exhausted. Surface that distinctly
		// as *cliutil.RateLimitError rather than a generic wrapped error, so
		// callers (including dogfood's whole-slate scan commands) can tell
		// "rate limited" apart from "no data" — empty-on-throttle would
		// otherwise silently corrupt results.
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests {
			return nil, &cliutil.RateLimitError{URL: c.http.RequestBaseURL(), Body: apiErr.Body}
		}
		return nil, fmt.Errorf("bmr graphql request failed: %w", err)
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decoding bmr graphql response: %w", err)
	}
	if len(env.Errors) > 0 {
		return env.Data, &QueryError{Errors: env.Errors}
	}
	return env.Data, nil
}
