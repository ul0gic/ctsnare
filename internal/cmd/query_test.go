package cmd

import (
	"testing"
	"time"
)

func TestParseSince(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{in: "", want: 0},
		{in: "1h", want: time.Hour},
		{in: "90m", want: 90 * time.Minute},
		{in: "24h", want: 24 * time.Hour},
		{in: "7d", want: 7 * 24 * time.Hour},
		{in: "0.5d", want: 12 * time.Hour},
		{in: "d", wantErr: true},
		{in: "xd", wantErr: true},
		{in: "7x", wantErr: true},
		{in: "garbage", wantErr: true},
	}
	for _, tt := range tests {
		got, err := parseSince(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSince(%q): expected error, got %v", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSince(%q): unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseSince(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
