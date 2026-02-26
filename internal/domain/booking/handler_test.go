package booking

import (
	"testing"
	"time"
)

func TestParseBookingDateTime(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 10, 0, 0, 0, now.Location())

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		wantToday bool
	}{
		{name: "rfc3339", input: "2026-02-26T10:00:00Z", wantErr: false},
		{name: "iso without seconds", input: "2026-02-26T10:00", wantErr: false},
		{name: "datetime with space", input: "2026-02-26 10:00", wantErr: false},
		{name: "time only", input: "10:00", wantErr: false, wantToday: true},
		{name: "invalid", input: "not-a-time", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBookingDateTime(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}

			if tt.wantToday {
				if got.Year() != today.Year() || got.Month() != today.Month() || got.Day() != today.Day() {
					t.Fatalf("expected today's date, got %s", got.Format(time.RFC3339))
				}
				if got.Hour() != today.Hour() || got.Minute() != today.Minute() {
					t.Fatalf("expected time %02d:%02d, got %02d:%02d", today.Hour(), today.Minute(), got.Hour(), got.Minute())
				}
			}
		})
	}
}
