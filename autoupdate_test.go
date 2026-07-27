package main

import "testing"

func TestSemverNewer(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"v1.1.0", "v1.0.0", true},
		{"v1.0.1", "v1.0.0", true},
		{"v2.0.0", "v1.9.9", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.0", "v2.0.0", false},
		{"1.2.3", "1.2.2", true},
		{"v10.20.30", "v9.20.30", true},
		{"dev", "v1.0.0", false},
		{"v1.0.0", "dev", true},
	}
	for _, tt := range tests {
		got := semverNewer(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("semverNewer(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
