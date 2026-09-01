// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/mvanhorn/printing-press-library/library/media-and-entertainment/bookmakersreview/internal/bmr"

	"github.com/spf13/cobra"
)

// addNovelCommandIfAbsent adds cmd to parent unless a command with the same
// Name() is already registered — a real implementation replacing a TODO
// scaffold of the same name wins, and re-running this from more than one
// hook is safe.
func addNovelCommandIfAbsent(parent *cobra.Command, cmd *cobra.Command) {
	for _, existing := range parent.Commands() {
		if existing.Name() == cmd.Name() {
			return
		}
	}
	parent.AddCommand(cmd)
}

// newBMRClient builds the hand-written GraphQL client on top of the
// generated HTTP transport, so timeout/dry-run/base-URL config stay
// consistent with every other command.
func newBMRClient(flags *rootFlags) (*bmr.Client, error) {
	c, err := flags.newClient()
	if err != nil {
		return nil, err
	}
	return bmr.New(c), nil
}
