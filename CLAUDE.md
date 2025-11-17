# suggest-claude-md - プロジェクト概要

Claude Codeのセッション終了時に会話履歴を分析し、CLAUDE.md更新提案を生成するGo製フックツール。

## 開発コマンド

```bash
# ビルド
mise run build

# テスト実行
mise run test

# カバレッジ確認
mise run coverage

# Lint実行
mise run lint

# フォーマット
mise run fmt

# すべてのチェック実行
mise run check
```

## アーキテクチャ

### 全体フロー

```
1. Claude Code Hook Trigger (SessionEnd/PreCompact)
   ↓
2. main.go: フック入力受信（JSON経由）
   ↓
3. transcript.go: 会話履歴の抽出・パース
   ↓
4. prompt.go: 既存CLAUDE.md読込 + プロンプト生成
   ↓
5. executor.go: バックグラウンドでClaude CLI実行
   ↓
6. 提案ファイル保存: /tmp/suggest-claude-md-{id}-{timestamp}.md
   ↓
7. macOS通知: --applyコマンドを表示
   ↓
8. ユーザー確認: suggest-claude-md --apply {提案ファイル}
   ↓
9. main.go (applySuggestionFile): 既存CLAUDE.mdと提案内容を表示
   ↓
10. ユーザー確認後にCLAUDE.mdへ追記
```

### 2段階承認ワークフロー

**Stage 1: バックグラウンド生成**
- フック実行時に自動的にバックグラウンドで実行
- 既存のCLAUDE.mdを読み込み、重複を避けて新しい提案を生成
- 提案ファイルを `/tmp/suggest-claude-md-*.md` に保存
- ログファイルを `/tmp/suggest-claude-md-*.log` に保存
- macOS通知で完了を通知（--applyコマンドを含む）

**Stage 2: 対話的適用**
- `--apply` コマンドで提案ファイルを指定
- 既存のCLAUDE.mdと提案内容を両方表示
- yes/y で承認、それ以外でキャンセル
- 承認時にCLAUDE.mdへ追記（既存内容の末尾に追加）

### モジュール構成

| ファイル | 責務 |
|---------|------|
| **main.go** | エントリーポイント、CLI引数解析、メインロジック、--apply実装 |
| **hooks.go** | フック自動インストール（user/projectスコープ）、settings.json操作 |
| **transcript.go** | Claude Code会話履歴のJSON解析、role/content抽出 |
| **prompt.go** | Claude CLIへのプロンプト生成、既存CLAUDE.md考慮 |
| **executor.go** | バックグラウンド実行（cmd.Start）、teeでログ/提案ファイル分岐保存 |
| **utils.go** | チルダ展開（`~/` → ホームディレクトリ） |
| **types.go** | データ構造定義（HookInput, Message, Settings） |

## 重要な設計パターン

### 1. 依存性注入によるテスタビリティ

main.goの`run()`関数は、外部依存を関数引数で受け取る設計：

```go
func run(
    input io.Reader,      // 標準入力（テスト時はモック可能）
    output io.Writer,     // 標準出力（テスト時はモック可能）
    getwd func() (string, error),  // カレントディレクトリ取得
    getenv func(string) string,    // 環境変数取得
    now func() time.Time,          // 現在時刻取得
) error
```

これにより、テスト時に制御可能な依存関係を注入できる。

### 2. テスト可能な入力処理

`applySuggestionFile()` は公開関数、`applySuggestionFileWithInput()` はテスト用内部関数：

```go
// 本番用: os.Stdinを使用
func applySuggestionFile(suggestionPath string) error {
    return applySuggestionFileWithInput(suggestionPath, os.Stdin)
}

// テスト用: io.Readerを受け取る
func applySuggestionFileWithInput(suggestionPath string, input io.Reader) error {
    scanner := bufio.NewScanner(input)  // テスト時はstrings.Readerを注入可能
    // ...
}
```

### 3. バックグラウンド実行とプロセス独立性

executor.goは`cmd.Start()`を使用し、親プロセス終了後も子プロセスが継続：

```go
cmd := exec.Command("sh", "-c", shellScript)
cmd.Env = append(os.Environ(), "SUGGEST_CLAUDE_MD_RUNNING=1")
return cmd.Start()  // Wait()せず、非同期実行
```

環境変数 `SUGGEST_CLAUDE_MD_RUNNING=1` で再帰実行を防止。

### 4. フック設定のマージ処理

hooks.goは既存のフック設定を保持しながら新しいフックを追加：

```go
// 既存のhooks配列に追加（重複チェック済み）
currentHooks = append(currentHooks, newHook)
```

これにより、他のフックと共存可能。

## Claude Code Hooksインテグレーション

### フックイベントの選択理由

- **SessionEnd**: 通常のセッション終了時に実行
- **PreCompact**: トークン上限によるコンパクション前に実行

両方で実行することで、どのような終了方法でも会話履歴を確実に保存。

### フック入力形式

Claude Codeは以下のJSON形式で標準入力にデータを渡す：

```json
{
  "hook_event_name": "SessionEnd",
  "trigger": "manual",
  "transcript_path": "~/.local/share/claude/session_transcripts/{conversation_id}.jsonl"
}
```

transcript_pathのJSONLファイルから会話履歴を抽出。

### スコープ管理

`--install-hook` は2つのスコープをサポート：

- **user**: `~/.claude/settings.json` - 全プロジェクト共通
- **project**: `.claude/settings.json` - プロジェクト固有（推奨）

## テスト戦略

- **テーブル駆動テスト**: 複数ケースを効率的にテスト
- **カバレッジ目標**: 85%以上（現在86.6%）
- **エラーパステスト**: ファイルが存在しない、読み込み失敗、ユーザー入力なしなど
- **統合テスト**: 一時ディレクトリを使用した実際のファイル操作テスト

### テスト実行環境

```bash
# 通常のテスト
go test -v ./src

# カバレッジ付き
go test -coverprofile=coverage.out ./src
go tool cover -func=coverage.out

# Lint（Docker経由）
docker run --rm -v $(pwd):/app -w /app golangci/golangci-lint:v1.62.2 golangci-lint run
```

## プロンプトの埋め込み

`prompt.go`の`DefaultPromptContent`にプロンプトを埋め込み：

- バイナリにコンパイル時に含まれる
- 外部ファイル不要でポータビリティ向上
- 既存CLAUDE.mdは実行時に動的に読み込み

## トラブルシューティング

### フックが実行されない

1. settings.jsonの場所を確認（user vs project）
2. バイナリが PATH に存在するか確認
3. `.claude`ディレクトリが存在するか確認

### 重複実行の防止

環境変数 `SUGGEST_CLAUDE_MD_RUNNING=1` をチェック：

```go
if getenv("SUGGEST_CLAUDE_MD_RUNNING") == "1" {
    fmt.Fprintln(output, "⚠️  既に実行中のため、スキップします")
    return nil
}
```

### ログとデバッグ

- **提案ファイル**: `/tmp/suggest-claude-md-{id}-{timestamp}.md`
- **ログファイル**: `/tmp/suggest-claude-md-{id}-{timestamp}.log`
- ログには実行情報、プロンプト全文、Claude CLIの出力が含まれる

## 開発のベストプラクティス

1. **新機能追加時**: 必ずテストケースを追加し、カバレッジを維持
2. **エラーハンドリング**: すべてのエラーに`fmt.Errorf`でコンテキストを付与
3. **Lint対応**: `mise run lint`でgoconst, gofmt, gosec等をチェック
4. **nolint コメント**: 正当な理由がある場合のみ使用し、理由を明記
5. **定数使用**: マジックストリング（"user", "project"等）は必ず定数化

## ライセンス

MIT License - 詳細は [LICENSE](LICENSE) 参照
