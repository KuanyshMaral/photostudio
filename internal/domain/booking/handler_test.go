package booking

import (
	"testing"
	"time"
)

func TestParseBookingDateTime(t *testing.T) {
	now := time.Date(2026, time.February, 26, 11, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		wantTime time.Time
	}{
		{name: "rfc3339", input: "2026-02-26T10:00:00Z", wantErr: false, wantTime: time.Date(2026, time.February, 26, 10, 0, 0, 0, time.UTC)},
		{name: "iso without seconds", input: "2026-02-26T10:00", wantErr: false, wantTime: time.Date(2026, time.February, 26, 10, 0, 0, 0, time.UTC)},
		{name: "datetime with space", input: "2026-02-26 10:00", wantErr: false, wantTime: time.Date(2026, time.February, 26, 10, 0, 0, 0, time.UTC)},
		{name: "time only same day future", input: "12:00", wantErr: false, wantTime: time.Date(2026, time.February, 26, 12, 0, 0, 0, time.UTC)},
		{name: "time only rolls to next day", input: "10:00", wantErr: false, wantTime: time.Date(2026, time.February, 27, 10, 0, 0, 0, time.UTC)},
		{name: "invalid", input: "not-a-time", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBookingDateTimeAt(tt.input, now)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tt.input, err)
			}

			if !tt.wantErr && !got.Equal(tt.wantTime) {
				t.Fatalf("expected %s, got %s", tt.wantTime.Format(time.RFC3339), got.Format(time.RFC3339))
			}
		})
	}
}
