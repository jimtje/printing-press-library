// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.
// cli-printing-press: novel-scaffold-test
// Novel command scaffold tests. Keep the wiring smoke test and add behavior cases as needed.

package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestNovelOddsValueHelpWires smoke-tests that the odds value command
// resolves at runtime and renders useful --help output. Catches wiring
// regressions (missing AddCommand, panicking RunE on --help, etc.) before
// review. Keep this smoke test when adding behavior-specific cases.
func TestNovelOddsValueHelpWires(t *testing.T) {
	cmd := RootCmd()
	cmd.SetArgs([]string{"odds", "value", "--help"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("odds value --help error = %v (novel command not wired correctly?)", err)
	}
	help := out.String()
	for _, want := range []string{"Usage:", "value"} {
		if !strings.Contains(help, want) {
			t.Fatalf("odds value --help missing %q in output:\n%s", want, help)
		}
	}
}
