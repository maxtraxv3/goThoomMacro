package vosk

import "testing"

func TestParseResultText(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{`{"text": "hello world"}`, "hello world"},
		{`{"partial": "hello wor"}`, "hello wor"},
		{`{"text": "", "partial": ""}`, ""},
		{`   {"text": "  hi  "}   `, "  hi  "},
		{"", ""},
		{"not json", "not json"},
	}
	for _, tc := range tests {
		if got := parseResultText(tc.in); got != tc.want {
			t.Errorf("parseResultText(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
