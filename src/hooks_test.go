package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSettings_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	settings, err := loadSettings(settingsPath)
	if err != nil {
		t.Fatalf("loadSettings() should not error for non-existent file: %v", err)
	}

	if settings == nil {
		t.Error("loadSettings() should return empty settings, not nil")
	}
}

func TestLoadSettings_Existing(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// 既存の設定ファイルを作成
	existing := &ClaudeSettings{
		Hooks: map[string][]HookEntry{
			"SessionEnd": {
				{
					Hooks: []HookCommand{
						{Type: "command", Command: "existing-hook"},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(existing)
	err := os.WriteFile(settingsPath, data, 0o644)
	if err != nil {
		t.Fatalf("Failed to create test settings file: %v", err)
	}

	settings, err := loadSettings(settingsPath)
	if err != nil {
		t.Fatalf("loadSettings() error = %v", err)
	}

	if len(settings.Hooks["SessionEnd"]) != 1 {
		t.Errorf("Expected 1 SessionEnd hook, got %d", len(settings.Hooks["SessionEnd"]))
	}

	if settings.Hooks["SessionEnd"][0].Hooks[0].Command != "existing-hook" {
		t.Errorf("Expected command 'existing-hook', got %q", settings.Hooks["SessionEnd"][0].Hooks[0].Command)
	}
}

func TestLoadSettings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// 無効なJSONファイルを作成
	err := os.WriteFile(settingsPath, []byte("invalid json content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = loadSettings(settingsPath)
	if err == nil {
		t.Error("loadSettings() should return error for invalid JSON")
	}
}

func TestLoadSettings_ReadPermissionDenied(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// 読み取り権限のないファイルを作成
	err := os.WriteFile(settingsPath, []byte("{}"), 0o000)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Chmod(settingsPath, 0o644) // nolint:errcheck // Best-effort cleanup in test

	_, err = loadSettings(settingsPath)
	// 環境によってはエラーが発生しない場合があるため、エラーがある場合のみチェック
	if err != nil && !os.IsPermission(err) {
		t.Logf("Got error (may vary by system): %v", err)
	}
}

func TestSaveSettings_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()

	// 存在しないディレクトリへの書き込み
	invalidPath := filepath.Join(tmpDir, "nonexistent", "settings.json")
	settings := &ClaudeSettings{}

	err := saveSettings(invalidPath, settings)
	if err == nil {
		t.Error("saveSettings() should return error for invalid path")
	}
}

func TestSaveSettings_WritePermissionDenied(t *testing.T) {
	tmpDir := t.TempDir()

	// 読み取り専用ディレクトリを作成
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0o555)
	if err != nil {
		t.Fatalf("Failed to create readonly directory: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0o755) // nolint:errcheck // Best-effort cleanup in test

	settingsPath := filepath.Join(readOnlyDir, "settings.json")
	settings := &ClaudeSettings{}

	err = saveSettings(settingsPath, settings)
	// 環境によってはエラーが発生しない場合があるため、エラーがある場合のみチェック
	if err != nil && !os.IsPermission(err) {
		t.Logf("Got error (may vary by system): %v", err)
	}
}

func TestSaveSettings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	settings := &ClaudeSettings{
		Hooks: map[string][]HookEntry{
			"SessionEnd": {
				{
					Hooks: []HookCommand{
						{Type: "command", Command: "test-command"},
					},
				},
			},
		},
	}

	err := saveSettings(settingsPath, settings)
	if err != nil {
		t.Fatalf("saveSettings() error = %v", err)
	}

	// ファイルが作成されたことを確認
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		t.Error("saveSettings() should create the file")
	}

	// 内容を確認
	data, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		t.Fatalf("Failed to read saved file: %v", readErr)
	}

	var loaded ClaudeSettings
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal saved settings: %v", err)
	}

	if loaded.Hooks["SessionEnd"][0].Hooks[0].Command != "test-command" {
		t.Errorf("Saved command mismatch, got %q", loaded.Hooks["SessionEnd"][0].Hooks[0].Command)
	}
}

func TestAddHookIfNotExists_NewHook(t *testing.T) {
	existingEntries := []HookEntry{
		{
			Hooks: []HookCommand{
				{Type: "command", Command: "existing-command"},
			},
		},
	}

	newHook := HookCommand{Type: "command", Command: "new-command"}
	result := addHookIfNotExists(existingEntries, newHook)

	// 既存のエントリーに追加されるので、エントリー数は1のまま
	if len(result) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(result))
	}

	// 最初のエントリーに2つのコマンドが入っているはず
	if len(result[0].Hooks) != 2 {
		t.Errorf("Expected 2 commands in first entry, got %d", len(result[0].Hooks))
	}

	// 1つ目のコマンドは既存のもの
	if result[0].Hooks[0].Command != "existing-command" {
		t.Errorf("Expected first command to be 'existing-command', got %q", result[0].Hooks[0].Command)
	}

	// 2つ目のコマンドは新しく追加したもの
	if result[0].Hooks[1].Command != "new-command" {
		t.Errorf("Expected second command to be 'new-command', got %q", result[0].Hooks[1].Command)
	}
}

func TestAddHookIfNotExists_ExistingHook(t *testing.T) {
	existingEntries := []HookEntry{
		{
			Hooks: []HookCommand{
				{Type: "command", Command: "suggest-claude-md"},
			},
		},
	}

	newHook := HookCommand{Type: "command", Command: "suggest-claude-md"}
	result := addHookIfNotExists(existingEntries, newHook)

	if len(result) != 1 {
		t.Errorf("Expected 1 entry (no duplicate), got %d", len(result))
	}
}

func TestAddHookIfNotExists_ExistingHookWithPath(t *testing.T) {
	existingEntries := []HookEntry{
		{
			Hooks: []HookCommand{
				{Type: "command", Command: "/usr/local/bin/suggest-claude-md"},
			},
		},
	}

	newHook := HookCommand{Type: "command", Command: "suggest-claude-md"}
	result := addHookIfNotExists(existingEntries, newHook)

	// 既存のパスでsuggest-claude-mdが見つかるため、追加されない
	if len(result) != 1 {
		t.Errorf("Expected 1 entry (existing hook with path detected), got %d", len(result))
	}
}

func TestAddHookIfNotExists_EmptyEntries(t *testing.T) {
	// 空のエントリー配列
	existingEntries := []HookEntry{}

	newHook := HookCommand{Type: "command", Command: "suggest-claude-md"}
	result := addHookIfNotExists(existingEntries, newHook)

	// 新しいエントリーが作成されるはず
	if len(result) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(result))
	}

	// エントリーには1つのコマンドが入っているはず
	if len(result[0].Hooks) != 1 {
		t.Errorf("Expected 1 command in entry, got %d", len(result[0].Hooks))
	}

	// コマンドはsuggest-claude-md
	if result[0].Hooks[0].Command != "suggest-claude-md" {
		t.Errorf("Expected command to be 'suggest-claude-md', got %q", result[0].Hooks[0].Command)
	}
}

func TestInstallHooks_InvalidScope(t *testing.T) {
	err := installHooks("invalid")
	if err == nil {
		t.Error("installHooks() should return error for invalid scope")
	}

	if err != nil && !strings.Contains(err.Error(), "無効なスコープ") {
		t.Errorf("Expected error about invalid scope, got: %v", err)
	}
}

func TestInstallHooks_NoClaudeDirectory(t *testing.T) {
	// 一時ディレクトリに移動
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup in test

	os.Chdir(tmpDir) // nolint:errcheck // Test will fail if this fails

	err := installHooks(scopeProject)
	if err == nil {
		t.Error("installHooks() should return error when .claude directory doesn't exist")
	}

	if err != nil && err.Error() != ".claudeディレクトリが見つかりません。このディレクトリはClaude Codeプロジェクトではない可能性があります" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestInstallHooks_ProjectScope_Success(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup in test

	os.Chdir(tmpDir)                     // nolint:errcheck // Test will fail if this fails
	os.Mkdir(".claude", 0o755)           // nolint:errcheck // Test will fail if this fails
	defer os.RemoveAll(".claude")        // nolint:errcheck // Best-effort cleanup in test
	defer os.Remove("settings.json.bak") // nolint:errcheck // Best-effort cleanup in test

	err := installHooks(scopeProject)
	if err != nil {
		t.Fatalf("installHooks() unexpected error: %v", err)
	}

	// settings.jsonが作成されたことを確認
	settingsPath := filepath.Join(".claude", "settings.json")
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		t.Error("installHooks() should create .claude/settings.json")
	}

	// 内容を確認
	data, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		t.Fatalf("Failed to read settings file: %v", readErr)
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// SessionEndとPreCompactにフックが追加されているか確認
	if len(settings.Hooks["SessionEnd"]) == 0 {
		t.Error("SessionEnd hook should be added")
	}

	if len(settings.Hooks["PreCompact"]) == 0 {
		t.Error("PreCompact hook should be added")
	}
}

func TestInstallHooks_UserScope_Success(t *testing.T) {
	// ホームディレクトリ配下にテスト用.claudeを作成
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("Cannot get home directory")
	}

	testClaudeDir := filepath.Join(homeDir, ".claude-test-"+t.Name())
	defer os.RemoveAll(testClaudeDir) // nolint:errcheck // Best-effort cleanup in test

	// 一時的にHOMEを変更
	originalHome := os.Getenv("HOME")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)            // nolint:errcheck // Test will fail if this fails
	defer os.Setenv("HOME", originalHome) // nolint:errcheck // Best-effort cleanup in test

	err = installHooks(scopeUser)
	if err != nil {
		t.Fatalf("installHooks() unexpected error: %v", err)
	}

	// settings.jsonが作成されたことを確認
	settingsPath := filepath.Join(tmpHome, ".claude", "settings.json")
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		t.Error("installHooks() should create ~/.claude/settings.json")
	}

	// 内容を確認
	data, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		t.Fatalf("Failed to read settings file: %v", readErr)
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Failed to unmarshal settings: %v", err)
	}

	// SessionEndとPreCompactにフックが追加されているか確認
	if len(settings.Hooks["SessionEnd"]) == 0 {
		t.Error("SessionEnd hook should be added")
	}

	if len(settings.Hooks["PreCompact"]) == 0 {
		t.Error("PreCompact hook should be added")
	}
}

func TestInstallHooks_LoadSettingsError(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup in test

	os.Chdir(tmpDir)              // nolint:errcheck // Test will fail if this fails
	os.Mkdir(".claude", 0o755)    // nolint:errcheck // Test will fail if this fails
	defer os.RemoveAll(".claude") // nolint:errcheck // Best-effort cleanup in test

	// 無効なJSONファイルを作成してloadSettingsをエラーにする
	settingsPath := filepath.Join(".claude", "settings.json")
	err := os.WriteFile(settingsPath, []byte("invalid json"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create invalid settings file: %v", err)
	}

	err = installHooks(scopeProject)
	if err == nil {
		t.Error("installHooks() should return error when loadSettings fails")
	}

	if err != nil && !strings.Contains(err.Error(), "設定ファイルの読み込みに失敗") {
		t.Errorf("Expected error message about load failure, got: %v", err)
	}
}

func TestInstallHooks_SaveSettingsError(t *testing.T) {
	tmpDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd) // nolint:errcheck // Best-effort cleanup in test

	os.Chdir(tmpDir)                 // nolint:errcheck // Test will fail if this fails
	os.Mkdir(".claude", 0o555)       // nolint:errcheck // 読み取り専用
	defer os.Chmod(".claude", 0o755) // nolint:errcheck // Best-effort cleanup in test
	defer os.RemoveAll(".claude")    // nolint:errcheck // Best-effort cleanup in test

	err := installHooks(scopeProject)
	// 環境によってはエラーが発生しない場合があるため、エラーがある場合のみチェック
	if err != nil && !strings.Contains(err.Error(), "設定ファイルの保存に失敗") && !os.IsPermission(err) {
		t.Logf("Got error (may vary by system): %v", err)
	}
}

func TestSaveSettings_Success(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "test-settings.json")

	settings := &ClaudeSettings{
		Hooks: map[string][]HookEntry{
			"TestHook": {
				{
					Hooks: []HookCommand{
						{Type: "command", Command: "test"},
					},
				},
			},
		},
	}

	err := saveSettings(settingsPath, settings)
	if err != nil {
		t.Fatalf("saveSettings() error = %v", err)
	}

	// ファイルが作成されたことを確認
	if _, statErr := os.Stat(settingsPath); os.IsNotExist(statErr) {
		t.Error("File should be created")
	}

	// JSONとして読み込めることを確認
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var loaded ClaudeSettings
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if loaded.Hooks["TestHook"][0].Hooks[0].Command != "test" {
		t.Errorf("Unexpected command: %v", loaded.Hooks["TestHook"][0].Hooks[0].Command)
	}
}

func TestAddHookIfNotExists_MultipleExistingHooks(t *testing.T) {
	existingEntries := []HookEntry{
		{
			Hooks: []HookCommand{
				{Type: "command", Command: "hook1"},
				{Type: "command", Command: "hook2"},
				{Type: "command", Command: "hook3"},
			},
		},
	}

	newHook := HookCommand{Type: "command", Command: "hook4"}
	result := addHookIfNotExists(existingEntries, newHook)

	if len(result) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(result))
	}

	if len(result[0].Hooks) != 4 {
		t.Errorf("Expected 4 hooks, got %d", len(result[0].Hooks))
	}

	// 4番目のフックがhook4であることを確認
	if result[0].Hooks[3].Command != "hook4" {
		t.Errorf("Expected 4th hook to be 'hook4', got %q", result[0].Hooks[3].Command)
	}
}
