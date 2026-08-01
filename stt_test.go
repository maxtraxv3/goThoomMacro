package main

import "testing"

func TestSttFindCommand(t *testing.T) {
	sttCommandsMu.Lock()
	sttCommands = map[string]string{
		"say":           "{text}",
		"yell":          "\\YELL {text}",
		"think":         "\\THINK {text}",
		"think to":      "\\THINKTO {text}",
		"think to clan": "\\THINKCLAN {text}",
		"pose":          "\\POSE {text}",
		"status":        "\\STATUS",
		"attack orc":    "attack orc",
	}
	sttCommandsMu.Unlock()

	tests := []struct {
		spoken string
		phrase string
		cmd    string
		rest   string
		ok     bool
	}{
		{"yell help", "yell", "\\YELL {text}", "help", true},
		{"YELL help", "yell", "\\YELL {text}", "help", true},
		{"say hello everyone", "say", "{text}", "hello everyone", true},
		{"think to clan everyone rise", "think to clan", "\\THINKCLAN {text}", "everyone rise", true},
		{"think to bob hi", "think to", "\\THINKTO {text}", "bob hi", true},
		{"think deeply", "think", "\\THINK {text}", "deeply", true},
		{"thinking about things", "", "", "", false}, // word boundary: no space after "think"
		{"status", "status", "\\STATUS", "", true},
		{"attack orc", "attack orc", "attack orc", "", true},
		{"attack the orc", "", "", "", false}, // phrase must start the utterance
		{"pose", "pose", "\\POSE {text}", "", true},
		{"nothing relevant", "", "", "", false},
	}
	for _, tc := range tests {
		phrase, cmd, rest, ok := sttFindCommand(tc.spoken)
		if phrase != tc.phrase || cmd != tc.cmd || rest != tc.rest || ok != tc.ok {
			t.Errorf("sttFindCommand(%q) = (%q, %q, %q, %v), want (%q, %q, %q, %v)",
				tc.spoken, phrase, cmd, rest, ok, tc.phrase, tc.cmd, tc.rest, tc.ok)
		}
	}
}

func TestSttBuildCommand(t *testing.T) {
	// The {text} placeholder is replaced with everything spoken after the phrase.
	cmd := "\\YELL {text}"
	rest := "help me"
	got := sttTemplate(cmd, rest)
	want := "\\YELL help me"
	if got != want {
		t.Fatalf("sttTemplate(%q, %q) = %q, want %q", cmd, rest, got, want)
	}
	// Commands without a placeholder stay unchanged.
	if got := sttTemplate("\\STATUS", "ignored"); got != "\\STATUS" {
		t.Fatalf("sttTemplate without placeholder = %q, want \\STATUS", got)
	}
}

func TestSttParseCommands(t *testing.T) {
	// Inline '#' comments and leading indentation must not leak into the
	// command value.
	content := "# header comment\n" +
		"   say={text}                      # \"say hello all\" -> sends \"hello all\"\n" +
		"yell=\\YELL {text}               # \"yell help\" -> \"\\YELL help\"\n" +
		"think to clan=\\THINKCLAN {text} # clan private message\n" +
		"status=\\STATUS\n" +
		"# fully commented line\n"
	got := parseSTTCommands(content)
	want := map[string]string{
		"say":           "{text}",
		"yell":          "\\YELL {text}",
		"think to clan": "\\THINKCLAN {text}",
		"status":        "\\STATUS",
	}
	if len(got) != len(want) {
		t.Fatalf("parseSTTCommands returned %d entries, want %d: %v", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseSTTCommands[%q] = %q, want %q", k, got[k], v)
		}
	}
}
