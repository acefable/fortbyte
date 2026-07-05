package main

import (
	"strings"
	"testing"
)

func TestPromptLine(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"normal", "hello\n", "hello", false},
		{"trimmed", "  world  \n", "world", false},
		{"empty", "\n", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.input)
			var buf strings.Builder
			got, err := promptLine(&buf, r, "prompt: ")
			if (err != nil) != tt.wantErr {
				t.Errorf("promptLine() err=%v, wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("promptLine() = %q, want %q", got, tt.want)
			}
			if !strings.Contains(buf.String(), "prompt:") {
				t.Errorf("prompt not written to writer")
			}
		})
	}
}

func TestPromptLineEOF(t *testing.T) {
	r := strings.NewReader("")
	var buf strings.Builder
	_, err := promptLine(&buf, r, "prompt: ")
	if err == nil {
		t.Error("expected error on EOF")
	}
}

func TestConfirmDeletion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"yes", "yes\n", true},
		{"y", "y\n", true},
		{"no", "no\n", false},
		{"reset", "RESET\n", false}, // only yes/y are accepted
		{"yes caps", "Yes\n", true},
		{"empty", "\n", false},
		{"garbage", "blah\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(tt.in)
			var buf strings.Builder
			got, err := confirmDeletion(&buf, r, "test-secret")
			if err != nil {
				t.Errorf("confirmDeletion() unexpected err: %v", err)
			}
			if got != tt.want {
				t.Errorf("confirmDeletion(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestConfirmDeletionEOF(t *testing.T) {
	r := strings.NewReader("")
	var buf strings.Builder
	_, err := confirmDeletion(&buf, r, "test")
	if err == nil {
		t.Error("expected error on EOF")
	}
}
