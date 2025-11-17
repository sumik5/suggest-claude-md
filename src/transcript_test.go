package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractConversationHistory(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		fileContent string
		wantContain string
		wantErr     bool
	}{
		{
			name: "valid transcript with string content",
			fileContent: `{"message":{"role":"user","content":"Hello"}}
{"message":{"role":"assistant","content":"Hi there"}}`,
			wantContain: "### user",
			wantErr:     false,
		},
		{
			name: "valid transcript with array content",
			fileContent: `{"message":{"role":"user","content":[{"type":"text","text":"Hello"}]}}
{"message":{"role":"assistant","content":[{"type":"text","text":"Hi"}]}}`,
			wantContain: "### user",
			wantErr:     false,
		},
		{
			name: "empty lines should be skipped",
			fileContent: `{"message":{"role":"user","content":"Hello"}}

{"message":{"role":"assistant","content":"Hi"}}`,
			wantContain: "### user",
			wantErr:     false,
		},
		{
			name:        "invalid json should be skipped",
			fileContent: `invalid json line`,
			wantContain: "",
			wantErr:     false,
		},
		{
			name: "empty content should be skipped",
			fileContent: `{"message":{"role":"user","content":""}}
{"message":{"role":"assistant","content":"Response"}}`,
			wantContain: "### assistant",
			wantErr:     false,
		},
		{
			name: "multiple messages with mixed content types",
			fileContent: `{"message":{"role":"user","content":"Question 1"}}
{"message":{"role":"assistant","content":[{"type":"text","text":"Answer 1"}]}}
{"message":{"role":"user","content":[{"type":"text","text":"Question 2"}]}}
{"message":{"role":"assistant","content":"Answer 2"}}`,
			wantContain: "Question 1",
			wantErr:     false,
		},
		{
			name: "empty array content should be skipped",
			fileContent: `{"message":{"role":"user","content":[]}}
{"message":{"role":"assistant","content":"Response"}}`,
			wantContain: "### assistant",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(tmpDir, tt.name+".jsonl")
			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0o600)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			result, err := ExtractConversationHistory(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractConversationHistory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantContain != "" && !strings.Contains(result, tt.wantContain) {
				t.Errorf("ExtractConversationHistory() result does not contain %q, got: %q", tt.wantContain, result)
			}

			// 空のコンテンツがスキップされることを確認
			if tt.name == "empty content should be skipped" {
				if strings.Contains(result, "### user") {
					t.Error("Empty user content should be skipped")
				}
			}
		})
	}
}

func TestExtractConversationHistory_FileNotFound(t *testing.T) {
	_, err := ExtractConversationHistory("/nonexistent/file.jsonl")
	if err == nil {
		t.Error("ExtractConversationHistory() should return error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "ファイルを開けません") {
		t.Errorf("Error message should contain 'ファイルを開けません', got: %v", err)
	}
}

func TestExtractTextContent(t *testing.T) {
	tests := []struct {
		name    string
		content interface{}
		want    string
	}{
		{
			name:    "string content",
			content: "Hello, world!",
			want:    "Hello, world!",
		},
		{
			name: "array content with text",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Hello"},
				map[string]interface{}{"type": "text", "text": "World"},
			},
			want: "Hello\nWorld",
		},
		{
			name: "array content with non-text items",
			content: []interface{}{
				map[string]interface{}{"type": "image", "data": "..."},
				map[string]interface{}{"type": "text", "text": "Hello"},
			},
			want: "Hello",
		},
		{
			name: "array content with empty text",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": ""},
				map[string]interface{}{"type": "text", "text": "Hello"},
			},
			want: "Hello",
		},
		{
			name:    "nil content",
			content: nil,
			want:    "",
		},
		{
			name:    "empty array",
			content: []interface{}{},
			want:    "",
		},
		{
			name: "array with invalid type field",
			content: []interface{}{
				map[string]interface{}{"type": 123, "text": "Hello"},
			},
			want: "",
		},
		{
			name: "array with missing type field",
			content: []interface{}{
				map[string]interface{}{"text": "Hello"},
			},
			want: "",
		},
		{
			name: "array with non-text type",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": "Valid"},
				map[string]interface{}{"type": "code", "text": "Should be skipped"},
			},
			want: "Valid",
		},
		{
			name: "array with invalid text field type",
			content: []interface{}{
				map[string]interface{}{"type": "text", "text": 123},
			},
			want: "",
		},
		{
			name: "array with non-map items",
			content: []interface{}{
				"invalid item",
				map[string]interface{}{"type": "text", "text": "Hello"},
			},
			want: "Hello",
		},
		{
			name:    "integer content",
			content: 123,
			want:    "",
		},
		{
			name:    "boolean content",
			content: true,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextContent(tt.content)
			if result != tt.want {
				t.Errorf("extractTextContent() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestExtractConversationHistory_ScannerError(t *testing.T) {
	// Scanner errors are difficult to trigger in unit tests, testing normal case
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")
	content := `{"message":{"role":"user","content":"Test"}}`
	err := os.WriteFile(tmpFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	result, err := ExtractConversationHistory(tmpFile)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(result, "### user") {
		t.Errorf("Expected result to contain '### user', got: %q", result)
	}
}

func TestExtractConversationHistory_Formatting(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "format.jsonl")
	content := `{"message":{"role":"user","content":"Question"}}
{"message":{"role":"assistant","content":"Answer"}}`
	err := os.WriteFile(tmpFile, []byte(content), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	result, err := ExtractConversationHistory(tmpFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// フォーマットの確認
	expected := "### user\n\nQuestion\n\n### assistant\n\nAnswer"
	if result != expected {
		t.Errorf("Expected format:\n%q\nGot:\n%q", expected, result)
	}
}
