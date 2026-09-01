// Copyright 2026 jim zhou and contributors. Licensed under Apache-2.0. See LICENSE.

package bmr

import "testing"

func TestGraphQLErrorString(t *testing.T) {
	cases := []struct {
		name string
		err  GraphQLError
		want string
	}{
		{
			name: "uses message when present",
			err:  GraphQLError{Message: "boom"},
			want: "boom",
		},
		{
			name: "falls back to raw path when message is empty",
			err:  GraphQLError{Path: []byte(`["consensus","Proxy"]`)},
			want: `["consensus","Proxy"]`,
		},
		{
			name: "falls back to a generic label when both are empty",
			err:  GraphQLError{},
			want: "unknown graphql error",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestQueryErrorError(t *testing.T) {
	err := &QueryError{Errors: []GraphQLError{
		{Message: "first problem"},
		{Message: "second problem"},
	}}
	want := "bmr graphql error: first problem; second problem"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
