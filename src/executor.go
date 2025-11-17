package main

import (
	"fmt"
	"os"
	"os/exec"
)

// ExecutorConfig holds configuration for background execution.
type ExecutorConfig struct {
	ProjectRoot        string
	TempPromptFilePath string
	LogFile            string
	HookInfo           string
	SuggestionFile     string // 提案ファイルのパス
}

// ExecuteInBackground executes Claude CLI in background.
func ExecuteInBackground(config *ExecutorConfig) error {
	shellScript := fmt.Sprintf(`
		cd '%s' || exit 1
		export SUGGEST_CLAUDE_MD_RUNNING=1

		# claudeコマンドを実行し、出力をログファイルと提案ファイルに保存
		claude --dangerously-skip-permissions --output-format text --print < '%s' | tee '%s' > '%s' 2>&1

		# ログファイルにフック情報を追記
		{
			echo ""
			echo "---"
			echo ""
			echo "## フック実行情報"
			echo ""
			echo "%s"
			echo ""
			echo "---"
			echo ""
			echo "## 実際に渡したプロンプト全文"
			echo ""
			cat '%s'
		} >> '%s'

		# macOS通知を送信（--applyコマンドを含める）
		osascript -e 'display notification "提案ファイル: %s\n適用: suggest-claude-md --apply %s" with title "CLAUDE.md更新提案" subtitle "分析が完了しました"'

		# 一時ファイル削除
		rm -f '%s'
	`, config.ProjectRoot, config.TempPromptFilePath, config.SuggestionFile, config.LogFile,
		config.HookInfo, config.TempPromptFilePath, config.LogFile,
		config.SuggestionFile, config.SuggestionFile, config.TempPromptFilePath)

	cmd := exec.Command("sh", "-c", shellScript)
	cmd.Env = append(os.Environ(), "SUGGEST_CLAUDE_MD_RUNNING=1")

	// Startで非同期実行（main関数終了後も継続）
	return cmd.Start()
}
