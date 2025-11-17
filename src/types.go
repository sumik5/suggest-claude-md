// Package main provides suggest-claude-md, a tool that analyzes conversation history
// and generates CLAUDE.md update suggestions.
package main

// HookInput represents the JSON input from Claude Code hook.
type HookInput struct {
	TranscriptPath string `json:"transcript_path"`
	HookEventName  string `json:"hook_event_name"`
	Trigger        string `json:"trigger"`
}

// Message represents a single message in the conversation.
type Message struct {
	Message MessageContent `json:"message"`
}

// MessageContent contains the role and content of a message.
type MessageContent struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// ContentItem represents a single content item with type and text.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}
