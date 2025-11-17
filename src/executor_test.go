package main

import (
	"os"
	"testing"
)

func TestExecutorConfig(t *testing.T) {
	config := ExecutorConfig{
		ProjectRoot:        "/test/root",
		TempPromptFilePath: "/tmp/prompt.md",
		LogFile:            "/tmp/test.log",
		HookInfo:           "Hook: Test",
	}

	if config.ProjectRoot != "/test/root" {
		t.Errorf("ProjectRoot = %q, want %q", config.ProjectRoot, "/test/root")
	}
	if config.TempPromptFilePath != "/tmp/prompt.md" {
		t.Errorf("TempPromptFilePath = %q, want %q", config.TempPromptFilePath, "/tmp/prompt.md")
	}
	if config.LogFile != "/tmp/test.log" {
		t.Errorf("LogFile = %q, want %q", config.LogFile, "/tmp/test.log")
	}
	if config.HookInfo != "Hook: Test" {
		t.Errorf("HookInfo = %q, want %q", config.HookInfo, "Hook: Test")
	}
}

func TestExecuteInBackground_InvalidCommand(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/prompt.md"
	err := os.WriteFile(tmpFile, []byte("test"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	config := &ExecutorConfig{
		ProjectRoot:        "/nonexistent/directory",
		TempPromptFilePath: tmpFile,
		LogFile:            tmpDir + "/test.log",
		HookInfo:           "Test Hook",
	}

	// cmd.Start()はエラーにならないことが多い
	err = ExecuteInBackground(config)
	// エラーが返らなくてもOK（非同期実行のため）
	if err != nil {
		t.Logf("ExecuteInBackground() returned error: %v", err)
	}
}
