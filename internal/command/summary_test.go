package command

import (
	"testing"
	"time"
)

func TestResolveSummaryDateWithArg(t *testing.T) {
	t.Parallel()

	d, err := resolveSummaryDate("2026-03-06")
	if err != nil {
		t.Fatalf("resolveSummaryDate failed: %v", err)
	}

	if d.Year() != 2026 || d.Month() != time.March || d.Day() != 6 {
		t.Fatalf("unexpected date: %v", d)
	}
	if d.Hour() != 0 || d.Minute() != 0 || d.Second() != 0 {
		t.Fatalf("expected midnight date, got %v", d)
	}
}

func TestResolveSummaryDateInvalid(t *testing.T) {
	t.Parallel()

	if _, err := resolveSummaryDate("2026-99-99"); err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestFormatHHMMSS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		seconds int64
		want    string
	}{
		{seconds: 0, want: "00:00:00"},
		{seconds: 59, want: "00:00:59"},
		{seconds: 60, want: "00:01:00"},
		{seconds: 3600, want: "01:00:00"},
		{seconds: 3661, want: "01:01:01"},
		{seconds: -10, want: "00:00:00"},
	}

	for _, tc := range cases {
		got := formatHHMMSS(tc.seconds)
		if got != tc.want {
			t.Fatalf("formatHHMMSS(%d) = %s, want %s", tc.seconds, got, tc.want)
		}
	}
}
