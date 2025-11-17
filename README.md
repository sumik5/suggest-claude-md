# suggest-claude-md

Claude Code のセッション終了時に会話履歴を分析し、CLAUDE.md 更新提案を生成する Go 製フックツール。

## 概要

このツールは、Claude Code のフック機能を使用して、会話終了時に自動的に会話履歴を分析し、`CLAUDE.md` の更新提案を生成します。

## インストール

### 前提条件

- Go 1.24 以上
- macOS（通知機能を使用）
- Claude CLI（`claude` コマンドがインストールされていること）

### ビルド

```bash
mise run build
```

または直接：

```bash
go build -o build/suggest-claude-md ./src
```

バイナリは `build/suggest-claude-md` に生成されます。

**注**: プロンプトはバイナリに埋め込まれているため、外部ファイルは不要です。

### フックのインストール

#### 自動インストール（推奨）

プロジェクトディレクトリで以下のコマンドを実行すると、`.claude/settings.json`に自動的にフックが追加されます：

```bash
suggest-claude-md --install-hook
```

このコマンドは：
- `.claude/settings.json`に`SessionEnd`と`PreCompact`フックを追加
- 既存のフック設定を保持
- `.claude`ディレクトリが存在しない場合はエラーを表示

#### 手動設定

手動でフックを設定する場合は、`.claude/settings.json`に以下を追加します：

**設定ファイルの場所**:
- **ユーザ設定**: `~/.claude/settings.json`（全プロジェクト共通）
- **プロジェクト設定**: `.claude/settings.json`（推奨）

**設定内容**:

```json
{
  "hooks": {
    "SessionEnd": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "suggest-claude-md"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "suggest-claude-md"
          }
        ]
      }
    ]
  }
}
```

#### フックイベントの説明

- **SessionEnd**: 通常のセッション終了時に会話履歴を分析し、CLAUDE.md更新提案を生成
- **PreCompact**: トークン上限によるコンパクション前に会話履歴を分析・保存

両方のイベントで実行することで、どのようなセッション終了方法でも確実に会話履歴を保存できます。

## 使用方法

### ヘルプの表示

```bash
suggest-claude-md --help
```

### フックのインストール

```bash
# プロジェクトディレクトリで実行
cd /path/to/your/claude-project
suggest-claude-md --install-hook
```

### 通常の実行

通常は Claude Code のフックとして自動的に実行されます。手動実行する場合は標準入力からフック情報を渡す必要があります。

## ライセンス

このプロジェクトは MIT License のもとで公開されています。詳細は [LICENSE](LICENSE) ファイルを参照してください。
