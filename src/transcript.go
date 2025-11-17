package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	contentTypeText = "text"
)

// ExtractConversationHistory extracts conversation history from transcript file.
func ExtractConversationHistory(transcriptPath string) (string, error) {
	file, err := os.Open(transcriptPath)
	if err != nil {
		return "", fmt.Errorf("ファイルを開けません: %w", err)
	}
	defer file.Close() // nolint:errcheck // File is read-only, no need to check close error

	var history strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// JSONパースエラーはスキップ
			continue
		}

		// roleとcontentの抽出
		role := msg.Message.Role
		content := extractTextContent(msg.Message.Content)

		// 空のコンテンツはスキップ
		if content == "" {
			continue
		}

		// フォーマット: ### {role}\n\n{content}\n
		history.WriteString(fmt.Sprintf("### %s\n\n%s\n\n", role, content))
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("ファイルの読み込みエラー: %w", err)
	}

	return strings.TrimSpace(history.String()), nil
}

// extractTextContent extracts text content from content field.
// Content can be either an array of ContentItem or a string.
func extractTextContent(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var texts []string
		for _, item := range v {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, ok := itemMap["type"].(string); ok && itemType == contentTypeText {
					if text, ok := itemMap[contentTypeText].(string); ok && text != "" {
						texts = append(texts, text)
					}
				}
			}
		}
		return strings.Join(texts, "\n")
	default:
		return ""
	}
}
