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

// ExecuteSynchronously executes Claude CLI synchronously and outputs to stdout and log file.
func ExecuteSynchronously(config *ExecutorConfig) error {
	shellScript := fmt.Sprintf(`
		cd '%s' || exit 1
		export SUGGEST_CLAUDE_MD_RUNNING=1

		# claudeコマンドを実行し、出力を標準出力とログファイルと提案ファイルに保存
		# teeで標準出力に表示しつつ、提案ファイルにも保存
		claude --dangerously-skip-permissions --output-format text --print < '%s' | tee '%s' '%s'

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

		# 一時ファイル削除
		rm -f '%s'
	`, config.ProjectRoot, config.TempPromptFilePath, config.SuggestionFile, config.LogFile,
		config.HookInfo, config.TempPromptFilePath, config.LogFile,
		config.TempPromptFilePath)

	cmd := exec.Command("sh", "-c", shellScript)
	cmd.Env = append(os.Environ(), "SUGGEST_CLAUDE_MD_RUNNING=1")

	// Runで同期実行（完了を待つ）
	return cmd.Run()
}
