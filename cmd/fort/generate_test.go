package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		useSymbols bool
		wantLen    int
		wantErr    bool
	}{
		{"default with symbols", 24, true, 24, false},
		{"no symbols", 16, false, 16, false},
		{"length 1", 1, true, 1, false},
		{"zero length", 0, true, 0, true},
		{"negative length", -5, true, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw, err := generatePassword(tt.length, tt.useSymbols)
			if (err != nil) != tt.wantErr {
				t.Errorf("generatePassword() err=%v, wantErr=%v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if len(pw) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(pw), tt.wantLen)
			}
			if tt.useSymbols {
				for _, c := range pw {
					isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
					isDigit := c >= '0' && c <= '9'
					isSymbol := strings.ContainsRune("!@#$%^&*", c)
					if !isAlpha && !isDigit && !isSymbol {
						t.Errorf("unexpected char in password: %c", c)
					}
				}
			}
		})
	}
}

func TestGeneratePasswordUniqueness(t *testing.T) {
	pw1, err := generatePassword(24, true)
	if err != nil {
		t.Fatal(err)
	}
	pw2, err := generatePassword(24, true)
	if err != nil {
		t.Fatal(err)
	}
	if pw1 == pw2 {
		t.Errorf("two generated passwords are identical: %s", pw1)
	}
}

func TestGenerateCmdFlags(t *testing.T) {
	if generateCmd.Flags().Lookup("length") == nil {
		t.Error("generate missing --length flag")
	}
	if generateCmd.Flags().Lookup("no-symbols") == nil {
		t.Error("generate missing --no-symbols flag")
	}
}

func resetGenerateFlags(t *testing.T) {
	t.Helper()
	if f := generateCmd.Flags().Lookup("length"); f != nil {
		f.Value.Set(f.DefValue)
		f.Changed = false
	}
	if f := generateCmd.Flags().Lookup("no-symbols"); f != nil {
		f.Value.Set(f.DefValue)
		f.Changed = false
	}
}

func TestGenerateCmdExec(t *testing.T) {
	resetGenerateFlags(t)
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"generate"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate: %v", err)
	}
	pw := strings.TrimSpace(buf.String())
	if len(pw) != 24 {
		t.Errorf("default length = %d, want 24", len(pw))
	}
}

func TestGenerateCmdCustomLength(t *testing.T) {
	resetGenerateFlags(t)
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"generate", "-l", "8"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate: %v", err)
	}
	pw := strings.TrimSpace(buf.String())
	if len(pw) != 8 {
		t.Errorf("custom length = %d, want 8", len(pw))
	}
}

func TestGenerateCmdNoSymbols(t *testing.T) {
	resetGenerateFlags(t)
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(io.Discard)
	rootCmd.SetArgs([]string{"generate", "--no-symbols", "-l", "100"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("generate: %v", err)
	}
	pw := strings.TrimSpace(buf.String())
	for _, c := range pw {
		if strings.ContainsRune("!@#$%^&*", c) {
			t.Errorf("no-symbols password contains symbol: %c", c)
		}
	}
}
