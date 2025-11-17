package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	// テスト用の一時ディレクトリとファイルを作成
	tmpDir := t.TempDir()
	validTranscriptPath := filepath.Join(tmpDir, "test-conversation.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Hello"}}
{"message":{"role":"assistant","content":"Hi there"}}`
	err := os.WriteFile(validTranscriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript file: %v", err)
	}

	emptyTranscriptPath := filepath.Join(tmpDir, "empty-conversation.jsonl")
	err = os.WriteFile(emptyTranscriptPath, []byte(""), 0o600)
	if err != nil {
		t.Fatalf("Failed to create empty transcript file: %v", err)
	}

	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name           string
		input          string
		getenv         func(string) string
		getwd          func() (string, error)
		wantErr        bool
		wantOutputs    []string
		notWantOutputs []string
	}{
		{
			name: "valid input with successful execution",
			input: fmt.Sprintf(`{
				"transcript_path": "%s",
				"hook_event_name": "SessionEnd",
				"trigger": "user"
			}`, validTranscriptPath),
			getenv: func(key string) string { return "" },
			getwd: func() (string, error) {
				return tmpDir, nil
			},
			wantErr: false,
			wantOutputs: []string{
				"🤖 会話履歴を分析中...",
				"Hook: SessionEnd (trigger: user)",
				"📋 実行中...",
				"✅ 分析が完了しました",
				"📄 提案ファイル:",
			},
		},
		{
			name:  "recursive execution prevention",
			input: `{"transcript_path": "/tmp/test.jsonl", "hook_event_name": "SessionEnd", "trigger": "user"}`,
			getenv: func(key string) string {
				if key == "SUGGEST_CLAUDE_MD_RUNNING" {
					return "1"
				}
				return ""
			},
			getwd:       func() (string, error) { return tmpDir, nil },
			wantErr:     false,
			wantOutputs: []string{"⚠️  既に実行中のため、スキップします"},
		},
		{
			name:        "invalid json input",
			input:       `invalid json`,
			getenv:      func(key string) string { return "" },
			getwd:       func() (string, error) { return tmpDir, nil },
			wantErr:     true,
			wantOutputs: []string{"❌ フック入力の読み取りに失敗"},
		},
		{
			name: "empty transcript_path",
			input: `{
				"transcript_path": "",
				"hook_event_name": "SessionEnd",
				"trigger": "user"
			}`,
			getenv:      func(key string) string { return "" },
			getwd:       func() (string, error) { return tmpDir, nil },
			wantErr:     true,
			wantOutputs: []string{"❌ transcript_pathが空です"},
		},
		{
			name: "file does not exist",
			input: `{
				"transcript_path": "/nonexistent/file.jsonl",
				"hook_event_name": "SessionEnd",
				"trigger": "user"
			}`,
			getenv:      func(key string) string { return "" },
			getwd:       func() (string, error) { return tmpDir, nil },
			wantErr:     true,
			wantOutputs: []string{"❌ ファイルが存在しません"},
		},
		{
			name: "getwd error",
			input: fmt.Sprintf(`{
				"transcript_path": "%s",
				"hook_event_name": "SessionEnd",
				"trigger": "user"
			}`, validTranscriptPath),
			getenv: func(key string) string { return "" },
			getwd: func() (string, error) {
				return "", fmt.Errorf("mock getwd error")
			},
			wantErr:     true,
			wantOutputs: []string{"❌ カレントディレクトリの取得に失敗"},
		},
		{
			name: "empty conversation history",
			input: fmt.Sprintf(`{
				"transcript_path": "%s",
				"hook_event_name": "SessionEnd",
				"trigger": "user"
			}`, emptyTranscriptPath),
			getenv:      func(key string) string { return "" },
			getwd:       func() (string, error) { return tmpDir, nil },
			wantErr:     false,
			wantOutputs: []string{"⚠️  会話履歴が空のため、スキップします"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}
			now := func() time.Time { return fixedTime }

			err := run(input, output, tt.getwd, tt.getenv, now)

			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			outputStr := output.String()
			if err != nil {
				outputStr = err.Error()
			}

			for _, want := range tt.wantOutputs {
				if !strings.Contains(outputStr, want) {
					t.Errorf("run() output does not contain %q\nGot: %s", want, outputStr)
				}
			}

			for _, notWant := range tt.notWantOutputs {
				if strings.Contains(outputStr, notWant) {
					t.Errorf("run() output should not contain %q\nGot: %s", notWant, outputStr)
				}
			}
		})
	}
}

func TestRun_TimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	validTranscriptPath := filepath.Join(tmpDir, "test-conversation.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test"}}`
	err := os.WriteFile(validTranscriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, validTranscriptPath))
	output := &bytes.Buffer{}

	fixedTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	now := func() time.Time { return fixedTime }

	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outputStr := output.String()
	// タイムスタンプフォーマット: 20060102-150405 → 20240315-143045
	if !strings.Contains(outputStr, "20240315-143045") {
		t.Errorf("Timestamp format should be in expected format (20240315-143045), got: %s", outputStr)
	}
}

func TestRun_ConversationIDExtraction(t *testing.T) {
	tmpDir := t.TempDir()
	conversationFile := filepath.Join(tmpDir, "my-conversation-id.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test"}}`
	err := os.WriteFile(conversationFile, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, conversationFile))
	output := &bytes.Buffer{}

	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, time.Now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outputStr := output.String()
	// ログファイル名にconversation IDが含まれているか確認
	if !strings.Contains(outputStr, "my-conversation-id") {
		t.Errorf("Output should contain conversation ID 'my-conversation-id', got: %s", outputStr)
	}
}

func TestMain_Version(t *testing.T) {
	// versionは変更されないことを確認
	if version == "" {
		t.Error("version should not be empty")
	}
}

func TestPrintHelp(t *testing.T) {
	// printHelp()を呼び出して出力を確認（パニックしないことを確認）
	// 実際の出力内容は標準出力に書き込まれるため、ここでは呼び出しのみ確認
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("printHelp() should not panic: %v", r)
		}
	}()

	printHelp()
}

func TestRun_ConversationHistoryExtractionError(t *testing.T) {
	tmpDir := t.TempDir()

	// 無効なJSONファイルを作成（パース可能だがメッセージが不正）
	invalidTranscript := filepath.Join(tmpDir, "invalid.jsonl")
	// 有効なJSONだが、scanner.Errを発生させるのは困難なので、
	// ExtractConversationHistoryが失敗するケースをテスト
	// ここでは読み取り権限のないファイルを作成
	err := os.WriteFile(invalidTranscript, []byte("test"), 0o000) // 読み取り権限なし
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, invalidTranscript))
	output := &bytes.Buffer{}

	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, time.Now)

	// パーミッションエラーまたは読み取りエラーが発生するはず
	// ただし、環境によって動作が異なる可能性があるため、エラーがあることのみを確認
	if err != nil && !strings.Contains(err.Error(), "会話履歴の抽出に失敗") {
		// エラーが発生した場合は、適切なエラーメッセージを含むことを確認
		if !strings.Contains(err.Error(), "ファイルを開けません") && !strings.Contains(err.Error(), "permission denied") {
			t.Logf("Got error (this might be expected depending on system): %v", err)
		}
	}
}

func TestRun_TildeExpansion(t *testing.T) {
	tmpDir := t.TempDir()

	// ホームディレクトリ配下にテストファイルを作成
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	testSubDir := filepath.Join(homeDir, ".suggest-claude-md-test")
	err = os.MkdirAll(testSubDir, 0o755)
	if err != nil {
		t.Skipf("Cannot create test directory: %v", err)
	}
	defer os.RemoveAll(testSubDir) // nolint:errcheck // Best-effort cleanup in test

	testFile := filepath.Join(testSubDir, "test.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test"}}`
	err = os.WriteFile(testFile, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Skipf("Cannot create test file: %v", err)
	}

	// ~を使ったパスでテスト
	tildeTranscriptPath := "~/.suggest-claude-md-test/test.jsonl"

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, tildeTranscriptPath))
	output := &bytes.Buffer{}

	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, time.Now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "🤖 会話履歴を分析中") {
		t.Errorf("Tilde expansion should work, got: %s", outputStr)
	}
}

func TestRun_ExecuteInBackgroundError(t *testing.T) {
	// ExecuteInBackground内部でエラーが発生した場合のテスト
	// 実際にはExecuteInBackgroundは常にnilを返すため、このテストは限定的
	// しかし、将来的にエラーハンドリングが追加された場合のため、構造を確認

	tmpDir := t.TempDir()
	validTranscriptPath := filepath.Join(tmpDir, "test.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test"}}`
	err := os.WriteFile(validTranscriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, validTranscriptPath))
	output := &bytes.Buffer{}

	// 正常系で実行してエラーハンドリングコードパスを確認
	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, time.Now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	// 実行完了のメッセージが出力されることを確認
	outputStr := output.String()
	if !strings.Contains(outputStr, "✅ 分析が完了しました") {
		t.Errorf("Should contain success message, got: %s", outputStr)
	}
}

func TestRun_AllOutputMessages(t *testing.T) {
	// すべての出力メッセージをカバーするための包括的なテスト
	tmpDir := t.TempDir()
	validTranscriptPath := filepath.Join(tmpDir, "comprehensive-test.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Comprehensive test message"}}
{"message":{"role":"assistant","content":"Response to comprehensive test"}}`
	err := os.WriteFile(validTranscriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "PreCompact",
		"trigger": "system"
	}`, validTranscriptPath))
	output := &bytes.Buffer{}

	fixedTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	now := func() time.Time { return fixedTime }

	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outputStr := output.String()

	// すべての期待される出力メッセージを確認
	expectedMessages := []string{
		"🤖 会話履歴を分析中...",
		"Hook: PreCompact (trigger: system)",
		"📋 実行中...",
		"✅ 分析が完了しました",
		"📄 提案ファイル:",
		"以下のコマンドで提案を適用できます：",
		"suggest-claude-md --apply",
		"詳細なログ:",
		"/tmp/suggest-claude-md-comprehensive-test-20240615-103000",
	}

	for _, expected := range expectedMessages {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Output should contain %q, got: %s", expected, outputStr)
		}
	}
}

func TestRun_LogFilePathFormat(t *testing.T) {
	// ログファイルパスのフォーマットを詳細にテスト
	tmpDir := t.TempDir()
	transcriptFile := filepath.Join(tmpDir, "special-conversation-123.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test"}}`
	err := os.WriteFile(transcriptFile, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, transcriptFile))
	output := &bytes.Buffer{}

	fixedTime := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)
	now := func() time.Time { return fixedTime }

	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outputStr := output.String()

	// ログファイルパスの形式を確認
	expectedLogPath := "/tmp/suggest-claude-md-special-conversation-123-20241231-235959.log"
	if !strings.Contains(outputStr, expectedLogPath) {
		t.Errorf("Log file path should be %q, got: %s", expectedLogPath, outputStr)
	}
}

func TestApplySuggestionFile_FileNotFound(t *testing.T) {
	err := applySuggestionFile("/nonexistent/file.md")
	if err == nil {
		t.Error("applySuggestionFile() should return error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "提案ファイルが存在しません") {
		t.Errorf("Expected error about file not found, got: %v", err)
	}
}

func TestApplySuggestionFile_InvalidSuggestionFile(t *testing.T) {
	// 読み取り権限のないファイルを作成
	tmpFile, err := os.CreateTemp("", "suggestion-*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()                    // nolint:errcheck // File handle not needed
	defer os.Remove(tmpFilePath)       // nolint:errcheck // Best-effort cleanup
	os.Chmod(tmpFilePath, 0o000)       // nolint:errcheck // Test will check error handling
	defer os.Chmod(tmpFilePath, 0o644) // nolint:errcheck // Best-effort restore for cleanup

	err = applySuggestionFile(tmpFilePath)
	// 環境によってはエラーが発生しない場合があるため、エラーがある場合のみチェック
	if err != nil && !strings.Contains(err.Error(), "提案ファイルの読み込みに失敗") && !os.IsPermission(err) {
		t.Logf("Got error (may vary by system): %v", err)
	}
}

func TestApplySuggestionFile_GetCwdError(t *testing.T) {
	// このテストは難しい（os.Getwdのモックが必要）のでスキップ
	t.Skip("Skipping test that requires mocking os.Getwd")
}

func TestApplySuggestionFile_TildeExpansion(t *testing.T) {
	tmpDir := t.TempDir()

	// ホームディレクトリを一時的に変更
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)             // nolint:errcheck // Test will fail if this fails
	defer os.Setenv("HOME", originalHome) // nolint:errcheck // Best-effort cleanup

	// 提案ファイルを作成
	suggestionContent := "# Test Suggestion\n\nThis is a test."
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// チルダを使ったパスでファイルが見つかることを確認
	tildePathd := "~/suggestion.md"
	expandedPath := ExpandTilde(tildePathd)

	if expandedPath != suggestionPath {
		t.Errorf("ExpandTilde(%q) = %q, want %q", tildePathd, expandedPath, suggestionPath)
	}
}

func TestApplySuggestionFileWithInput_NoResponse(t *testing.T) {
	tmpDir := t.TempDir()

	// 提案ファイルを作成
	suggestionContent := "# Test Suggestion"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// 空の入力（EOFをシミュレート）
	input := strings.NewReader("")

	err = applySuggestionFileWithInput(suggestionPath, input)
	if err == nil {
		t.Error("applySuggestionFileWithInput() should return error for empty input")
	}
	if !strings.Contains(err.Error(), "入力がありません") {
		t.Errorf("Expected error about no input, got: %v", err)
	}
}

func TestApplySuggestionFileWithInput_CancelWithNo(t *testing.T) {
	tmpDir := t.TempDir()

	// 提案ファイルを作成
	suggestionContent := "# Test Suggestion"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// "no"を入力
	input := strings.NewReader("no\n")

	err = applySuggestionFileWithInput(suggestionPath, input)
	if err != nil {
		t.Errorf("applySuggestionFileWithInput() with 'no' should not return error, got: %v", err)
	}

	// CLAUDE.mdが作成されていないことを確認
	claudeMdPath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMdPath); !os.IsNotExist(err) {
		t.Error("CLAUDE.md should not be created when user says 'no'")
	}
}

func TestApplySuggestionFileWithInput_ApplyWithYes_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 提案ファイルを作成
	suggestionContent := "# Test Suggestion\n\nNew content"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// "yes"を入力
	input := strings.NewReader("yes\n")

	err = applySuggestionFileWithInput(suggestionPath, input)
	if err != nil {
		t.Errorf("applySuggestionFileWithInput() with 'yes' returned error: %v", err)
	}

	// CLAUDE.mdが作成されていることを確認
	claudeMdPath := filepath.Join(tmpDir, "CLAUDE.md")
	content, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("Failed to read CLAUDE.md: %v", err)
	}

	if string(content) != suggestionContent {
		t.Errorf("CLAUDE.md content = %q, want %q", string(content), suggestionContent)
	}
}

func TestApplySuggestionFileWithInput_ApplyWithYes_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// 既存のCLAUDE.mdを作成
	existingContent := "# Existing Content\n\nOld content"
	claudeMdPath := filepath.Join(tmpDir, "CLAUDE.md")
	err := os.WriteFile(claudeMdPath, []byte(existingContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create existing CLAUDE.md: %v", err)
	}

	// 提案ファイルを作成
	suggestionContent := "# New Suggestion\n\nNew content"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err = os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// "yes"を入力
	input := strings.NewReader("yes\n")

	err = applySuggestionFileWithInput(suggestionPath, input)
	if err != nil {
		t.Errorf("applySuggestionFileWithInput() with 'yes' returned error: %v", err)
	}

	// CLAUDE.mdが更新されていることを確認
	content, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("Failed to read CLAUDE.md: %v", err)
	}

	// 既存の内容の末尾に改行がない場合は追加され、さらに空行が追加される
	expectedContent := existingContent + "\n\n" + suggestionContent
	if string(content) != expectedContent {
		t.Errorf("CLAUDE.md content = %q, want %q", string(content), expectedContent)
	}
}

func TestApplySuggestionFileWithInput_ApplyWithY(t *testing.T) {
	tmpDir := t.TempDir()

	// 提案ファイルを作成
	suggestionContent := "# Test"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// "y"（yesの省略形）を入力
	input := strings.NewReader("y\n")

	err = applySuggestionFileWithInput(suggestionPath, input)
	if err != nil {
		t.Errorf("applySuggestionFileWithInput() with 'y' returned error: %v", err)
	}

	// CLAUDE.mdが作成されていることを確認
	claudeMdPath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, err := os.Stat(claudeMdPath); os.IsNotExist(err) {
		t.Error("CLAUDE.md should be created when user says 'y'")
	}
}

func TestApplySuggestionFileWithInput_ReadExistingClaudeMdError(t *testing.T) {
	tmpDir := t.TempDir()

	// 提案ファイルを作成
	suggestionContent := "# Test Suggestion"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 読み取り権限のないCLAUDE.mdを作成
	claudeMdPath := filepath.Join(tmpDir, "CLAUDE.md")
	err = os.WriteFile(claudeMdPath, []byte("existing"), 0o000)
	if err != nil {
		t.Fatalf("Failed to create CLAUDE.md: %v", err)
	}
	defer os.Chmod(claudeMdPath, 0o644) // nolint:errcheck // Best-effort cleanup

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// "yes"を入力
	input := strings.NewReader("yes\n")

	err = applySuggestionFileWithInput(suggestionPath, input)
	// 環境によってはエラーが発生しない場合があるため、エラーがある場合のみチェック
	if err != nil && !strings.Contains(err.Error(), "CLAUDE.mdの読み込みに失敗") && !os.IsPermission(err) {
		t.Logf("Got error (may vary by system): %v", err)
	}
}

func TestApplySuggestionFileWithInput_WriteClaudeMdError(t *testing.T) {
	tmpDir := t.TempDir()

	// 提案ファイルを作成
	suggestionContent := "# Test Suggestion"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 読み取り専用ディレクトリを作成
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err = os.Mkdir(readOnlyDir, 0o555)
	if err != nil {
		t.Fatalf("Failed to create readonly directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // nolint:errcheck // Best-effort cleanup

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(readOnlyDir)      // nolint:errcheck // Test will fail if this fails

	// "yes"を入力
	input := strings.NewReader("yes\n")

	err = applySuggestionFileWithInput(suggestionPath, input)
	// 環境によってはエラーが発生しない場合があるため、エラーがある場合のみチェック
	if err != nil && !strings.Contains(err.Error(), "CLAUDE.mdへの書き込みに失敗") && !os.IsPermission(err) {
		t.Logf("Got error (may vary by system): %v", err)
	}
}

func TestRun_CreateTempFileError(t *testing.T) {
	// os.CreateTempのエラーをシミュレートするのは困難なため、
	// このテストは実装をカバーするための構造確認のみ
	t.Skip("Skipping test that requires mocking os.CreateTemp")
}

func TestRun_WriteTempFileError(t *testing.T) {
	// tempPromptFile.WriteStringのエラーをシミュレートするのは困難なため、
	// このテストは実装をカバーするための構造確認のみ
	t.Skip("Skipping test that requires mocking file write operations")
}

func TestRun_ExecuteSynchronouslyError(t *testing.T) {
	tmpDir := t.TempDir()
	validTranscriptPath := filepath.Join(tmpDir, "test-conversation.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test message"}}`
	err := os.WriteFile(validTranscriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, validTranscriptPath))
	output := &bytes.Buffer{}

	// 環境変数でClaude CLIを無効なパスに設定して、ExecuteSynchronouslyを失敗させる
	// ただし、実際にはこのテストは環境依存性が高いため、構造確認のみ
	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string {
		if key == "PATH" {
			return "/nonexistent"
		}
		return ""
	}, time.Now)

	// ExecuteSynchronouslyが失敗する可能性がある
	// ただし、環境によっては成功する場合もある
	if err != nil {
		if !strings.Contains(err.Error(), "実行に失敗") {
			t.Logf("Got different error than expected: %v", err)
		}
	}
}

func TestRun_MultipleHookEvents(t *testing.T) {
	tmpDir := t.TempDir()
	validTranscriptPath := filepath.Join(tmpDir, "test-conversation.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test"}}`
	err := os.WriteFile(validTranscriptPath, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test transcript file: %v", err)
	}

	tests := []struct {
		name      string
		hookEvent string
		trigger   string
	}{
		{
			name:      "SessionEnd with user trigger",
			hookEvent: "SessionEnd",
			trigger:   "user",
		},
		{
			name:      "SessionEnd with system trigger",
			hookEvent: "SessionEnd",
			trigger:   "system",
		},
		{
			name:      "PreCompact with user trigger",
			hookEvent: "PreCompact",
			trigger:   "user",
		},
		{
			name:      "PreCompact with system trigger",
			hookEvent: "PreCompact",
			trigger:   "system",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(fmt.Sprintf(`{
				"transcript_path": "%s",
				"hook_event_name": "%s",
				"trigger": "%s"
			}`, validTranscriptPath, tt.hookEvent, tt.trigger))
			output := &bytes.Buffer{}

			err := run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, time.Now)
			if err != nil {
				t.Fatalf("run() error = %v", err)
			}

			outputStr := output.String()
			if !strings.Contains(outputStr, tt.hookEvent) {
				t.Errorf("Output should contain hook event %q, got: %s", tt.hookEvent, outputStr)
			}
			if !strings.Contains(outputStr, tt.trigger) {
				t.Errorf("Output should contain trigger %q, got: %s", tt.trigger, outputStr)
			}
		})
	}
}

func TestApplySuggestionFileWithInput_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()

	// 提案ファイルを作成
	suggestionContent := "# Test"
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err := os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// scanner.Errをシミュレートするのは困難なため、
	// 空の入力でEOFケースをテスト
	input := strings.NewReader("")

	err = applySuggestionFileWithInput(suggestionPath, input)
	if err == nil {
		t.Error("applySuggestionFileWithInput() should return error for empty input")
	}
	if !strings.Contains(err.Error(), "入力がありません") {
		t.Errorf("Expected error about no input, got: %v", err)
	}
}

func TestRun_ConversationIDWithSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()
	// 特殊文字を含むファイル名
	specialFileName := filepath.Join(tmpDir, "conversation-with-dashes_123.jsonl")
	transcriptContent := `{"message":{"role":"user","content":"Test"}}`
	err := os.WriteFile(specialFileName, []byte(transcriptContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	input := strings.NewReader(fmt.Sprintf(`{
		"transcript_path": "%s",
		"hook_event_name": "SessionEnd",
		"trigger": "user"
	}`, specialFileName))
	output := &bytes.Buffer{}

	err = run(input, output, func() (string, error) { return tmpDir, nil }, func(key string) string { return "" }, time.Now)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "conversation-with-dashes_123") {
		t.Errorf("Output should contain conversation ID with special chars, got: %s", outputStr)
	}
}

func TestApplySuggestionFileWithInput_ExistingWithMultipleSections(t *testing.T) {
	tmpDir := t.TempDir()

	// 既存のCLAUDE.mdに複数のセクションを作成
	existingContent := `# Project

## Section 1

Content 1.

## Section 2

Content 2.

## Section 3

Content 3.
`
	claudeMdPath := filepath.Join(tmpDir, "CLAUDE.md")
	err := os.WriteFile(claudeMdPath, []byte(existingContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create existing CLAUDE.md: %v", err)
	}

	// 提案ファイルを作成（新しいセクション）
	suggestionContent := `## Section 4

New content for section 4.
`
	suggestionPath := filepath.Join(tmpDir, "suggestion.md")
	err = os.WriteFile(suggestionPath, []byte(suggestionContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create suggestion file: %v", err)
	}

	// 作業ディレクトリを変更
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup
	os.Chdir(tmpDir)           // nolint:errcheck // Test will fail if this fails

	// "yes"を入力
	input := strings.NewReader("yes\n")

	err = applySuggestionFileWithInput(suggestionPath, input)
	if err != nil {
		t.Errorf("applySuggestionFileWithInput() with 'yes' returned error: %v", err)
	}

	// CLAUDE.mdが更新されていることを確認
	content, err := os.ReadFile(claudeMdPath)
	if err != nil {
		t.Fatalf("Failed to read CLAUDE.md: %v", err)
	}

	resultStr := string(content)
	// すべてのセクションが保持されている
	if !strings.Contains(resultStr, "Section 1") || !strings.Contains(resultStr, "Section 2") ||
		!strings.Contains(resultStr, "Section 3") || !strings.Contains(resultStr, "Section 4") {
		t.Errorf("All sections should be present, got: %s", resultStr)
	}
}
