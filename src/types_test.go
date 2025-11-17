package main

import (
	"encoding/json"
	"testing"
)

func TestHookInput(t *testing.T) {
	jsonData := `{
		"transcript_path": "/path/to/transcript.jsonl",
		"hook_event_name": "SessionEnd",
		"trigger": "test"
	}`

	var hookInput HookInput
	err := json.Unmarshal([]byte(jsonData), &hookInput)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if hookInput.TranscriptPath != "/path/to/transcript.jsonl" {
		t.Errorf("TranscriptPath = %q, want %q", hookInput.TranscriptPath, "/path/to/transcript.jsonl")
	}
	if hookInput.HookEventName != "SessionEnd" {
		t.Errorf("HookEventName = %q, want %q", hookInput.HookEventName, "SessionEnd")
	}
	if hookInput.Trigger != "test" {
		t.Errorf("Trigger = %q, want %q", hookInput.Trigger, "test")
	}
}

func TestMessage(t *testing.T) {
	jsonData := `{
		"message": {
			"role": "user",
			"content": "Hello"
		}
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if msg.Message.Role != "user" {
		t.Errorf("Role = %q, want %q", msg.Message.Role, "user")
	}

	content, ok := msg.Message.Content.(string)
	if !ok {
		t.Error("Content should be string")
	}
	if content != "Hello" {
		t.Errorf("Content = %q, want %q", content, "Hello")
	}
}

func TestMessageContent_ArrayContent(t *testing.T) {
	jsonData := `{
		"message": {
			"role": "assistant",
			"content": [{"type": "text", "text": "Response"}]
		}
	}`

	var msg Message
	err := json.Unmarshal([]byte(jsonData), &msg)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if msg.Message.Role != "assistant" {
		t.Errorf("Role = %q, want %q", msg.Message.Role, "assistant")
	}
}

func TestContentItem(t *testing.T) {
	jsonData := `{
		"type": "text",
		"text": "Hello, world!"
	}`

	var item ContentItem
	err := json.Unmarshal([]byte(jsonData), &item)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if item.Type != "text" {
		t.Errorf("Type = %q, want %q", item.Type, "text")
	}
	if item.Text != "Hello, world!" {
		t.Errorf("Text = %q, want %q", item.Text, "Hello, world!")
	}
}
