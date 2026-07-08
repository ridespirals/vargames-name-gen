package igdb

import "testing"

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		done, total int
		want        string
	}{
		{0, 0, ""},
		{0, 100, " (0.0%)"},
		{50, 200, " (25.0%)"},
		{200, 200, " (100.0%)"},
		{250, 200, " (100.0%)"},
	}
	for _, tc := range tests {
		if got := formatPercent(tc.done, tc.total); got != tc.want {
			t.Errorf("formatPercent(%d, %d) = %q, want %q", tc.done, tc.total, got, tc.want)
		}
	}
}
