package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	commandName  = "suggest-claude-md"
	scopeUser    = "user"
	scopeProject = "project"
)

// ClaudeSettings represents the structure of .claude/settings.json
type ClaudeSettings struct {
	Hooks map[string][]HookEntry `json:"hooks,omitempty"`
}

// HookEntry represents a hook entry in settings.json
type HookEntry struct {
	Hooks []HookCommand `json:"hooks"`
}

// HookCommand represents a hook command
type HookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// installHooks installs suggest-claude-md hooks to settings.json
func installHooks(scope string) error {
	// スコープの検証
	var settingsPath string
	switch scope {
	case scopeUser:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("ホームディレクトリの取得に失敗: %w", err)
		}
		claudeDir := filepath.Join(homeDir, ".claude")
		// .claudeディレクトリが存在しない場合は作成
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			if err := os.MkdirAll(claudeDir, 0o750); err != nil {
				return fmt.Errorf(".claudeディレクトリの作成に失敗: %w", err)
			}
		}
		settingsPath = filepath.Join(claudeDir, "settings.json")
	case scopeProject:
		claudeDir := ".claude"
		if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
			return fmt.Errorf(".claudeディレクトリが見つかりません。このディレクトリはClaude Codeプロジェクトではない可能性があります")
		}
		settingsPath = filepath.Join(claudeDir, "settings.json")
	default:
		return fmt.Errorf("無効なスコープ: %s (有効な値: user, project)", scope)
	}

	// 実行可能ファイルのパスを取得
	execPath, err := exec.LookPath(commandName)
	if err != nil {
		// フルパスが見つからない場合は、コマンド名を使用
		execPath = commandName
	}

	// 既存の設定を読み込む
	settings, err := loadSettings(settingsPath)
	if err != nil {
		return fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}

	// hooksが初期化されていない場合は初期化
	if settings.Hooks == nil {
		settings.Hooks = make(map[string][]HookEntry)
	}

	// SessionEndとPreCompactにフックを追加
	hookCommand := HookCommand{
		Type:    "command",
		Command: execPath,
	}

	// SessionEndに追加
	settings.Hooks["SessionEnd"] = addHookIfNotExists(settings.Hooks["SessionEnd"], hookCommand)

	// PreCompactに追加
	settings.Hooks["PreCompact"] = addHookIfNotExists(settings.Hooks["PreCompact"], hookCommand)

	// 設定を保存
	if err := saveSettings(settingsPath, settings); err != nil {
		return fmt.Errorf("設定ファイルの保存に失敗: %w", err)
	}

	scopeLabel := map[string]string{
		scopeUser:    "ユーザー設定（全プロジェクト共通）",
		scopeProject: "プロジェクト設定（現在のプロジェクトのみ）",
	}[scope]

	fmt.Println("✅ フックのインストールが完了しました")
	fmt.Printf("   スコープ: %s\n", scopeLabel)
	fmt.Printf("   設定ファイル: %s\n", settingsPath)
	fmt.Printf("   コマンド: %s\n", execPath)
	fmt.Println("\n登録されたフック:")
	fmt.Println("  - SessionEnd: 通常のセッション終了時")
	fmt.Println("  - PreCompact: トークン上限によるコンパクション前")

	return nil
}

// loadSettings loads settings from .claude/settings.json
func loadSettings(path string) (*ClaudeSettings, error) {
	// ファイルが存在しない場合は空の設定を返す
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ClaudeSettings{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// saveSettings saves settings to .claude/settings.json
func saveSettings(path string, settings *ClaudeSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// addHookIfNotExists adds a hook command if it doesn't already exist
func addHookIfNotExists(entries []HookEntry, hookCmd HookCommand) []HookEntry {
	// 既存のエントリーをチェックして、suggest-claude-mdが既に存在するか確認
	for _, entry := range entries {
		for _, cmd := range entry.Hooks {
			basename := filepath.Base(cmd.Command)
			if cmd.Type == hookCmd.Type && (cmd.Command == hookCmd.Command ||
				basename == commandName ||
				basename == commandName+".exe") {
				// 既に存在する場合はそのまま返す
				fmt.Println("⚠️  suggest-claude-mdフックは既に登録されています")
				return entries
			}
		}
	}

	// エントリーが既に存在する場合は、最初のエントリーに追加
	if len(entries) > 0 {
		entries[0].Hooks = append(entries[0].Hooks, hookCmd)
		return entries
	}

	// エントリーが空の場合は、新しいエントリーとして追加
	newEntry := HookEntry{
		Hooks: []HookCommand{hookCmd},
	}

	return append(entries, newEntry)
}
