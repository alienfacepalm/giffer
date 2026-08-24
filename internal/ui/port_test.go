package ui

import "testing"

func TestPortMatches(t *testing.T) {
	tests := []struct {
		local  string
		needle string
		want   bool
	}{
		{"127.0.0.1:8765", ":8765", true},
		{"0.0.0.0:8765", ":8765", true},
		{"[::1]:8765", ":8765", true},
		{"127.0.0.1:18765", ":8765", false},
		{"127.0.0.1:87650", ":8765", false},
		{"127.0.0.1:876", ":8765", false},
	}
	for _, tt := range tests {
		if got := portMatches(tt.local, tt.needle); got != tt.want {
			t.Errorf("portMatches(%q, %q) = %v, want %v", tt.local, tt.needle, got, tt.want)
		}
	}
}
