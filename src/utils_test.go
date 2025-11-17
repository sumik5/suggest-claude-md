package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func() string
	}{
		{
			name:  "tilde at start",
			input: "~/test/path",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "test", "path")
			},
		},
		{
			name:  "no tilde",
			input: "/absolute/path",
			expected: func() string {
				return "/absolute/path"
			},
		},
		{
			name:  "tilde in middle",
			input: "/path/~/test",
			expected: func() string {
				return "/path/~/test"
			},
		},
		{
			name:  "only tilde",
			input: "~",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return home
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: func() string {
				return ""
			},
		},
		{
			name:  "tilde with relative path",
			input: "~/Documents/test.txt",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "Documents", "test.txt")
			},
		},
		{
			name:  "tilde with slash only",
			input: "~/",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTilde(tt.input)
			expected := tt.expected()
			if result != expected {
				t.Errorf("ExpandTilde(%q) = %q, want %q", tt.input, result, expected)
			}
		})
	}
}

// TestExpandTildeWithInvalidHome tests the error handling when HOME environment variable is not set
func TestExpandTildeWithInvalidHome(t *testing.T) {
	// Save current HOME environment variable
	originalHome := os.Getenv("HOME")
	originalUserProfile := os.Getenv("USERPROFILE")

	// Unset HOME and USERPROFILE to simulate error condition
	os.Unsetenv("HOME")
	os.Unsetenv("USERPROFILE")

	// Restore after test
	defer func() {
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		}
		if originalUserProfile != "" {
			os.Setenv("USERPROFILE", originalUserProfile)
		}
	}()

	input := "~/test/path"
	result := ExpandTilde(input)

	// When os.UserHomeDir() fails, the function should return the original path
	if result != input {
		t.Errorf("ExpandTilde(%q) with invalid HOME = %q, want %q", input, result, input)
	}
}
