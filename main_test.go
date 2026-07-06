package main

import "testing"

func TestParseDirection(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"v", directionVertical, false},
		{"vertical", directionVertical, false},
		{"vertically", directionVertical, false},
		{"V", directionVertical, false},
		{"h", directionHorizontal, false},
		{"horizontal", directionHorizontal, false},
		{"H", directionHorizontal, false},
		{"diagonal", "", true},
		{"", "", true},
		{"vv", "", true},
	}
	for _, tt := range tests {
		got, err := parseDirection(tt.in)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDirection(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("parseDirection(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsIDToken(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"5", true},
		{"35267", true},
		{"W35267", true},
		{"w35267", true},
		{"T6", true},
		{"P2", true},
		{"npm", false},
		{"run", false},
		{"", false},
		{"P", false},
		{"5x", false},
	}
	for _, tt := range tests {
		if got := isIDToken(tt.in); got != tt.want {
			t.Errorf("isIDToken(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestEscapeForAppleScript(t *testing.T) {
	tests := []struct{ in, want string }{
		{`plain`, `plain`},
		{`a"b`, `a\"b`},
		{`a\b`, `a\\b`},
		{`\"`, `\\\"`}, // backslash-then-quote order: \ -> \\, then " -> \"
		{``, ``},
	}
	for _, tt := range tests {
		if got := escapeForAppleScript(tt.in); got != tt.want {
			t.Errorf("escapeForAppleScript(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParsePaneAddress(t *testing.T) {
	tests := []struct {
		name                          string
		args                          []string
		wantWindow, wantTab, wantPane int
		wantRest                      int
		wantOK                        bool
	}{
		{"empty", nil, 0, 0, 0, 0, false},
		{"one arg", []string{"5"}, 0, 0, 0, 0, false},
		{"tab pane", []string{"5", "2"}, 0, 5, 2, 2, true},
		{"prefixed tab pane", []string{"T5", "P2"}, 0, 5, 2, 2, true},
		{"tab pane trailing lines", []string{"5", "2", "100"}, 0, 5, 2, 2, true},
		{"window tab pane", []string{"35267", "6", "3"}, 35267, 6, 3, 3, true},
		{"prefixed window", []string{"W35267", "6", "3"}, 35267, 6, 3, 3, true},
		{"all prefixed", []string{"W35267", "T6", "P2"}, 35267, 6, 2, 3, true},
		{"window-like first, 2 tokens, incomplete", []string{"200", "5"}, 0, 0, 0, 0, false},
		{"W-prefix, 2 tokens, incomplete", []string{"W35267", "6"}, 0, 0, 0, 0, false},
		{"window-like, 3 tokens, invalid third", []string{"W35267", "6", "notanum"}, 0, 0, 0, 0, false},
		{"boundary 100 is tab", []string{"100", "5", "2"}, 0, 100, 5, 2, true},
		{"boundary 101 is window", []string{"101", "5", "2"}, 101, 5, 2, 3, true},
		{"command, no target", []string{"npm", "run", "dev"}, 0, 0, 0, 0, false},
		{"tab pane then command", []string{"5", "2", "npm", "run"}, 0, 5, 2, 2, true},
		{"window tab pane then command", []string{"W1", "T2", "P3", "echo"}, 1, 2, 3, 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, tab, pane, rest, ok := parsePaneAddress(tt.args)
			if w != tt.wantWindow || tab != tt.wantTab || pane != tt.wantPane || rest != tt.wantRest || ok != tt.wantOK {
				t.Errorf("parsePaneAddress(%v) = (w=%d tab=%d pane=%d rest=%d ok=%v), want (w=%d tab=%d pane=%d rest=%d ok=%v)",
					tt.args, w, tab, pane, rest, ok, tt.wantWindow, tt.wantTab, tt.wantPane, tt.wantRest, tt.wantOK)
			}
		})
	}
}
