// Package main provides suggest-claude-md, a tool that analyzes conversation history
// and generates CLAUDE.md update suggestions.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var version = "1.0.0" // Can be set during build

func main() {
	_ = version // version is set during build via ldflags

	// フラグの定義
	installHook := flag.String("install-hook", "", "Install hooks (user: ~/.claude/settings.json, project: .claude/settings.json)")
	applySuggestion := flag.String("apply", "", "Apply suggestion file to CLAUDE.md")
	showHelp := flag.Bool("help", false, "Show help message")
	flag.Parse()

	// ヘルプ表示
	if *showHelp {
		printHelp()
		return
	}

	// --install-hookが指定された場合
	if *installHook != "" {
		if err := installHooks(*installHook); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		return
	}

	// --applyが指定された場合
	if *applySuggestion != "" {
		if err := applySuggestionFile(*applySuggestion); err != nil {
			fmt.Fprintf(os.Stderr, "❌ %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 通常のフック実行
	if err := run(os.Stdin, os.Stdout, os.Getwd, os.Getenv, time.Now); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("suggest-claude-md - Claude Code CLAUDE.md update suggestion tool")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  suggest-claude-md [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --install-hook <scope>")
	fmt.Println("                    Install hooks for SessionEnd and PreCompact events")
	fmt.Println("                    Scope:")
	fmt.Println("                      user    - Install to ~/.claude/settings.json (all projects)")
	fmt.Println("                      project - Install to .claude/settings.json (current project only)")
	fmt.Println("  --apply <file>   Apply suggestion file to CLAUDE.md")
	fmt.Println("                    Displays existing CLAUDE.md content and proposed changes")
	fmt.Println("                    Prompts for confirmation before applying")
	fmt.Println("  --help           Show this help message")
	fmt.Println("")
	fmt.Println("Normal usage:")
	fmt.Println("  This tool is typically invoked as a Claude Code hook and reads hook input from stdin.")
	fmt.Println("  Suggestions are saved to /tmp/suggest-claude-md-*.md")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  # Install hooks to user settings (all projects)")
	fmt.Println("  suggest-claude-md --install-hook user")
	fmt.Println("")
	fmt.Println("  # Install hooks to project settings (current project only)")
	fmt.Println("  suggest-claude-md --install-hook project")
	fmt.Println("")
	fmt.Println("  # Apply a suggestion file to CLAUDE.md")
	fmt.Println("  suggest-claude-md --apply /tmp/suggest-claude-md-abc123.md")
	fmt.Println("")
	fmt.Println("  # Show help")
	fmt.Println("  suggest-claude-md --help")
}

// run is the main logic that can be tested.
func run(input io.Reader, output io.Writer, getwd func() (string, error), getenv func(string) string, now func() time.Time) error {
	// 再帰実行防止
	if getenv("SUGGEST_CLAUDE_MD_RUNNING") == "1" {
		_, _ = fmt.Fprintln(output, "⚠️  既に実行中のため、スキップします") // nolint:errcheck // Output to user, error not critical
		return nil
	}

	// フック入力の読み取り
	var hookInput HookInput
	decoder := json.NewDecoder(input)
	if err := decoder.Decode(&hookInput); err != nil {
		return fmt.Errorf("❌ フック入力の読み取りに失敗: %w", err)
	}

	// transcript_pathの検証
	if hookInput.TranscriptPath == "" {
		return fmt.Errorf("❌ transcript_pathが空です")
	}

	// ~をホームディレクトリに展開
	transcriptPath := ExpandTilde(hookInput.TranscriptPath)

	// ファイルの存在確認
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		return fmt.Errorf("❌ ファイルが存在しません: %s", transcriptPath)
	}

	// PROJECT_ROOTの取得
	projectRoot, err := getwd()
	if err != nil {
		return fmt.Errorf("❌ カレントディレクトリの取得に失敗: %w", err)
	}

	// CONVERSATION_IDの抽出
	conversationID := strings.TrimSuffix(filepath.Base(transcriptPath), filepath.Ext(transcriptPath))

	// TIMESTAMPの生成
	timestamp := now().Format("20060102-150405")

	// ログファイルと提案ファイルのパス
	logFile := fmt.Sprintf("/tmp/suggest-claude-md-%s-%s.log", conversationID, timestamp)
	suggestionFile := fmt.Sprintf("/tmp/suggest-claude-md-%s-%s.md", conversationID, timestamp)

	_, _ = fmt.Fprintln(output, "🤖 会話履歴を分析中...") // nolint:errcheck // Output to user, error not critical
	hookInfo := fmt.Sprintf("Hook: %s (trigger: %s)", hookInput.HookEventName, hookInput.Trigger)
	_, _ = fmt.Fprintln(output, hookInfo)                      // nolint:errcheck // Output to user, error not critical
	_, _ = fmt.Fprintf(output, "📋 バックグラウンドで実行中...\n")          // nolint:errcheck // Output to user, error not critical
	_, _ = fmt.Fprintf(output, "ログファイル: %s\n", logFile)        // nolint:errcheck // Output to user, error not critical
	_, _ = fmt.Fprintf(output, "提案ファイル: %s\n", suggestionFile) // nolint:errcheck // Output to user, error not critical

	// 会話履歴の抽出
	conversationHistory, err := ExtractConversationHistory(transcriptPath)
	if err != nil {
		return fmt.Errorf("❌ 会話履歴の抽出に失敗: %w", err)
	}

	if conversationHistory == "" {
		_, _ = fmt.Fprintln(output, "⚠️  会話履歴が空のため、スキップします") // nolint:errcheck // Output to user, error not critical
		return nil
	}

	// 既存のCLAUDE.mdを読み込む
	claudeMdPath := filepath.Join(projectRoot, "CLAUDE.md")
	var existingClaudeMd string
	if content, readErr := os.ReadFile(claudeMdPath); readErr == nil {
		existingClaudeMd = string(content)
	}

	// プロンプトファイルの生成
	promptContent := GeneratePrompt(DefaultPromptContent, conversationHistory, existingClaudeMd)

	// 一時ファイルの作成
	tempPromptFile, err := os.CreateTemp("", "suggest-claude-md-prompt-*.md")
	if err != nil {
		return fmt.Errorf("❌ 一時ファイルの作成に失敗: %w", err)
	}
	tempPromptFilePath := tempPromptFile.Name()

	if _, err := tempPromptFile.WriteString(promptContent); err != nil {
		_ = tempPromptFile.Close()        // nolint:errcheck // Best-effort cleanup in error path
		_ = os.Remove(tempPromptFilePath) // nolint:errcheck // Best-effort cleanup in error path
		return fmt.Errorf("❌ 一時ファイルへの書き込みに失敗: %w", err)
	}
	_ = tempPromptFile.Close() // nolint:errcheck // File is read-only from here

	// バックグラウンド実行
	config := &ExecutorConfig{
		ProjectRoot:        projectRoot,
		TempPromptFilePath: tempPromptFilePath,
		LogFile:            logFile,
		HookInfo:           hookInfo,
		SuggestionFile:     suggestionFile,
	}

	if err := ExecuteInBackground(config); err != nil {
		_ = os.Remove(tempPromptFilePath) // nolint:errcheck // Best-effort cleanup in error path
		return fmt.Errorf("❌ バックグラウンド実行の開始に失敗: %w", err)
	}

	_, _ = fmt.Fprintf(output, "\n✅ バックグラウンドで実行を開始しました\n")                              // nolint:errcheck // Output to user, error not critical
	_, _ = fmt.Fprintf(output, "   完了時にmacOS通知でお知らせします\n")                              // nolint:errcheck // Output to user, error not critical
	_, _ = fmt.Fprintf(output, "   結果: cat %s\n", logFile)                              // nolint:errcheck // Output to user, error not critical
	_, _ = fmt.Fprintf(output, "   適用: suggest-claude-md --apply %s\n", suggestionFile) // nolint:errcheck // Output to user, error not critical

	return nil
}

// applySuggestionFile applies a suggestion file to CLAUDE.md after user confirmation
func applySuggestionFile(suggestionPath string) error {
	return applySuggestionFileWithInput(suggestionPath, os.Stdin)
}

// applySuggestionFileWithInput applies a suggestion file with a custom input reader (for testing)
func applySuggestionFileWithInput(suggestionPath string, input io.Reader) error {
	// 提案ファイルの存在確認
	suggestionPath = ExpandTilde(suggestionPath)
	if _, err := os.Stat(suggestionPath); os.IsNotExist(err) {
		return fmt.Errorf("提案ファイルが存在しません: %s", suggestionPath)
	}

	// 提案ファイルを読み込む
	suggestionContent, err := os.ReadFile(suggestionPath)
	if err != nil {
		return fmt.Errorf("提案ファイルの読み込みに失敗: %w", err)
	}

	// CLAUDE.mdのパスを取得
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("カレントディレクトリの取得に失敗: %w", err)
	}
	claudeMdPath := filepath.Join(cwd, "CLAUDE.md")

	// 既存のCLAUDE.mdを読み込む（存在しない場合は空文字列）
	var existingContent string
	if _, err := os.Stat(claudeMdPath); err == nil {
		content, readErr := os.ReadFile(claudeMdPath)
		if readErr != nil {
			return fmt.Errorf("CLAUDE.mdの読み込みに失敗: %w", readErr)
		}
		existingContent = string(content)
	}

	// 既存のCLAUDE.mdの内容を表示
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("📄 現在のCLAUDE.md")
	fmt.Println("=" + strings.Repeat("=", 79))
	if existingContent == "" {
		fmt.Println("(ファイルは存在しません)")
	} else {
		fmt.Println(existingContent)
	}
	fmt.Println()

	// 提案内容を表示
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println("✨ 追加する提案内容")
	fmt.Println("=" + strings.Repeat("=", 79))
	fmt.Println(string(suggestionContent))
	fmt.Println()

	// 確認プロンプト
	fmt.Print("この内容をCLAUDE.mdに追記しますか? (yes/no): ")

	// inputから1行読み取る
	scanner := bufio.NewScanner(input)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("入力の読み取りに失敗: %w", err)
		}
		return fmt.Errorf("入力がありません")
	}
	response := scanner.Text()

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "yes" && response != "y" {
		fmt.Println("❌ キャンセルしました")
		return nil
	}

	// CLAUDE.mdに追記
	var newContent string
	if existingContent == "" {
		newContent = string(suggestionContent)
	} else {
		// 既存の内容の末尾に改行がない場合は追加
		if !strings.HasSuffix(existingContent, "\n") {
			existingContent += "\n"
		}
		newContent = existingContent + "\n" + string(suggestionContent)
	}

	if err := os.WriteFile(claudeMdPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("CLAUDE.mdへの書き込みに失敗: %w", err)
	}

	fmt.Printf("✅ CLAUDE.mdを更新しました: %s\n", claudeMdPath)
	fmt.Printf("   提案ファイル: %s\n", suggestionPath)

	return nil
}
