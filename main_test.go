package main

import (
	"net/http"
	"slices"
	"testing"
)

func TestFilterOutHopByHopHeaders(t *testing.T) {
	tests := []struct {
		name         string
		input        http.Header
		shouldExist  []string
		shouldBeGone []string
	}{
		{
			name:         "Keep-Alive is deleted",
			input:        http.Header{"Keep-Alive": {"timeout=5"}},
			shouldBeGone: []string{"Keep-Alive"},
		},
		{
			name:        "Content-Type is not deleted",
			input:       http.Header{"Content-Type": {"bxz"}},
			shouldExist: []string{"Content-Type"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filterOutHopByHopHeaders(test.input)
			var headersPresent []string
			for k := range test.input {
				headersPresent = append(headersPresent, k)
			}
			for _, sExist := range test.shouldExist {
				if !slices.Contains(headersPresent, sExist) {
					t.Errorf(
						"Expected header %s to exist, but it was deleted. Headers: %s",
						sExist,
						headersPresent)
				}
			}
			for _, test_gone := range test.shouldBeGone {
				if slices.Contains(headersPresent, test_gone) {
					t.Errorf("Unexpected header %s", test_gone)
				}
			}

		})
	}
}
