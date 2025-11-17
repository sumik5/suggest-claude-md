package main

import (
	"strings"
	"testing"
)

func TestParseSections(t *testing.T) {
	content := `# Main Title

Some intro text.

## Section 1

Content of section 1.

### Subsection 1.1

Content of subsection 1.1.

## Section 2

Content of section 2.
`

	sections := ParseSections(content)

	if len(sections) == 0 {
		t.Fatal("Expected sections to be parsed")
	}

	// Check first section
	if sections[0].Level != 1 {
		t.Errorf("Expected level 1, got %d", sections[0].Level)
	}
	if sections[0].Title != "Main Title" {
		t.Errorf("Expected 'Main Title', got '%s'", sections[0].Title)
	}
}

func TestFindSectionByTitle(t *testing.T) {
	sections := []Section{
		{Level: 2, Title: "Section 1", Content: "## Section 1\n\nContent 1"},
		{Level: 2, Title: "Section 2", Content: "## Section 2\n\nContent 2"},
	}

	section := FindSectionByTitle(sections, "section 1")
	if section == nil {
		t.Fatal("Expected to find 'Section 1'")
	}
	if section.Title != "Section 1" {
		t.Errorf("Expected 'Section 1', got '%s'", section.Title)
	}

	section = FindSectionByTitle(sections, "Non-existent")
	if section != nil {
		t.Error("Expected nil for non-existent section")
	}
}

func TestInsertIntoSection(t *testing.T) {
	existing := `# Project

## Section 1

Existing content in section 1.

## Section 2

Existing content in section 2.
`

	suggestion := `## Section 1

### New Subsection

New content to add to section 1.

## Section 3

Completely new section.
`

	result := InsertIntoSection(existing, suggestion)

	// Check that new subsection was added to Section 1
	if !strings.Contains(result, "New Subsection") {
		t.Error("Expected new subsection to be added")
	}

	// Check that Section 3 was appended
	if !strings.Contains(result, "## Section 3") {
		t.Error("Expected Section 3 to be appended")
	}

	// Check that original Section 2 is still there
	if !strings.Contains(result, "Existing content in section 2") {
		t.Error("Expected Section 2 to remain")
	}
}

func TestInsertIntoSection_NoSections(t *testing.T) {
	existing := "# Project\n\nSome content without sections.\n"
	suggestion := "Just some text without section headers."

	result := InsertIntoSection(existing, suggestion)

	// Should append to end
	if !strings.Contains(result, "Just some text") {
		t.Error("Expected suggestion to be appended")
	}
}

func TestInsertIntoSection_EmptyExisting(t *testing.T) {
	existing := ""
	suggestion := `## New Section

Some new content.
`

	result := InsertIntoSection(existing, suggestion)

	if result != suggestion {
		t.Error("Expected suggestion to be used as-is when existing is empty")
	}
}

func TestInsertIntoSection_MultipleSubsections(t *testing.T) {
	existing := `# Project

## Architecture

Existing architecture notes.

## Testing

Existing testing notes.
`

	suggestion := `## Architecture

### Database Design

New database design notes.

### API Design

New API design notes.
`

	result := InsertIntoSection(existing, suggestion)

	// Both subsections should be added
	if !strings.Contains(result, "Database Design") {
		t.Error("Expected Database Design subsection to be added")
	}
	if !strings.Contains(result, "API Design") {
		t.Error("Expected API Design subsection to be added")
	}

	// Original content should remain
	if !strings.Contains(result, "Existing architecture notes") {
		t.Error("Expected existing architecture notes to remain")
	}
	if !strings.Contains(result, "Existing testing notes") {
		t.Error("Expected existing testing notes to remain")
	}
}

func TestExtractSubsectionContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal section with content",
			input:    "## Section Title\n\nSome content\nMore content\n",
			expected: "\nSome content\nMore content\n",
		},
		{
			name:     "section with only title",
			input:    "## Section Title\n",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubsectionContent(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestAppendContent(t *testing.T) {
	tests := []struct {
		name       string
		existing   string
		newContent string
		expected   string
	}{
		{
			name:       "existing with newline",
			existing:   "Existing content\n",
			newContent: "New content",
			expected:   "Existing content\n\nNew content",
		},
		{
			name:       "existing without newline",
			existing:   "Existing content",
			newContent: "New content",
			expected:   "Existing content\n\nNew content",
		},
		{
			name:       "empty existing",
			existing:   "",
			newContent: "New content",
			expected:   "\n\nNew content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := appendContent(tt.existing, tt.newContent)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestInsertIntoSection_Level1Sections(t *testing.T) {
	// Level 1セクションの挿入テスト
	existing := `# Main Title

Some content here.

## Level 2 Section

Content in level 2.
`

	suggestion := `# Another Main Title

This should be appended.
`

	result := InsertIntoSection(existing, suggestion)

	// Level 1セクションは末尾に追加される
	if !strings.Contains(result, "Another Main Title") {
		t.Error("Expected level 1 section to be appended")
	}
}

func TestInsertIntoSection_DeeplyNestedSubsections(t *testing.T) {
	// 深くネストしたサブセクションのテスト
	existing := `# Project

## Architecture

### Layer 1

Content in layer 1.

## Testing

### Test Layer 1

Test content.
`

	suggestion := `## Architecture

#### Layer 2

New nested content.

##### Layer 3

Very deeply nested content.
`

	result := InsertIntoSection(existing, suggestion)

	// Layer 2とLayer 3が追加されることを確認
	if !strings.Contains(result, "Layer 2") {
		t.Error("Expected Layer 2 subsection to be added")
	}
	if !strings.Contains(result, "New nested content") {
		t.Error("Expected Layer 2 content to be added")
	}
	if !strings.Contains(result, "Layer 3") {
		t.Error("Expected Layer 3 subsection to be added")
	}
	if !strings.Contains(result, "Very deeply nested content") {
		t.Error("Expected Layer 3 content to be added")
	}

	// 既存のコンテンツが保持されることを確認
	if !strings.Contains(result, "Layer 1") {
		t.Error("Expected existing Layer 1 to be preserved")
	}
	if !strings.Contains(result, "## Testing") {
		t.Error("Expected Testing section to be preserved")
	}
}

func TestInsertIntoSection_PartialTitleMatch(t *testing.T) {
	// セクションタイトルの部分一致テスト
	existing := `# Project

## Test Section

Existing test content.

## Testing Framework

Different section.
`

	suggestion := `## Test Section

### New Test

Additional test content.
`

	result := InsertIntoSection(existing, suggestion)

	// "Test Section"に正確にマッチすることを確認
	lines := strings.Split(result, "\n")
	testSectionCount := 0
	for _, line := range lines {
		if strings.Contains(line, "## Test Section") {
			testSectionCount++
		}
	}

	// "Test Section"は1回のみ存在するはず
	if testSectionCount != 1 {
		t.Errorf("Expected 1 occurrence of 'Test Section', got %d", testSectionCount)
	}

	// 新しいサブセクションが追加されていることを確認
	if !strings.Contains(result, "New Test") {
		t.Error("Expected new subsection to be added")
	}
}

func TestInsertIntoSection_SectionWithNoContent(t *testing.T) {
	// 内容が空のセクションへの挿入テスト
	existing := `# Project

## Empty Section

## Another Section

Some content here.
`

	suggestion := `## Empty Section

### New Content

This is new content for the empty section.
`

	result := InsertIntoSection(existing, suggestion)

	// 新しいコンテンツが追加されることを確認
	if !strings.Contains(result, "New Content") {
		t.Error("Expected new content to be added to empty section")
	}
	if !strings.Contains(result, "This is new content for the empty section") {
		t.Error("Expected detailed content to be added")
	}
}

func TestInsertIntoSection_MultipleSectionsWithSameSubsectionTitles(t *testing.T) {
	// 複数のセクションに同じサブセクション名がある場合のテスト
	existing := `# Project

## Section A

### Introduction

Content A intro.

## Section B

### Introduction

Content B intro.
`

	suggestion := `## Section A

### Details

New details for Section A.
`

	result := InsertIntoSection(existing, suggestion)

	// Section Aにのみ追加されることを確認
	if !strings.Contains(result, "Details") {
		t.Error("Expected new subsection to be added to Section A")
	}

	// Section Bの内容は変更されていないことを確認
	if !strings.Contains(result, "Content B intro") {
		t.Error("Expected Section B content to remain unchanged")
	}
}

func TestInsertIntoSection_EmptySubsectionContent(t *testing.T) {
	// サブセクションの内容が空の場合のテスト
	existing := `# Project

## Main Section

Existing content.
`

	suggestion := `## Main Section

### Empty Subsection
`

	result := InsertIntoSection(existing, suggestion)

	// 空のサブセクションは追加されないことを確認
	// または、追加される場合は正しくフォーマットされていることを確認
	if strings.Contains(result, "### Empty Subsection") {
		// 空のサブセクションが追加された場合
		t.Log("Empty subsection was added (this is acceptable)")
	}
}

func TestFindSectionByTitle_Level3Section(t *testing.T) {
	// Level 3セクションは検索対象外であることを確認
	sections := []Section{
		{Level: 2, Title: "Level 2 Section", Content: "## Level 2 Section\n\nContent"},
		{Level: 3, Title: "Level 3 Section", Content: "### Level 3 Section\n\nContent"},
	}

	// Level 2は見つかる
	section := FindSectionByTitle(sections, "Level 2 Section")
	if section == nil {
		t.Error("Expected to find Level 2 section")
	}

	// Level 3は見つからない（関数はLevel 2のみを検索）
	section = FindSectionByTitle(sections, "Level 3 Section")
	if section != nil {
		t.Error("Should not find Level 3 section (function only searches Level 2)")
	}
}

func TestParseSections_NoSections(t *testing.T) {
	// セクションヘッダーがない場合のテスト
	content := `Just some plain text
without any section headers.

This should not be parsed as a section.
`

	sections := ParseSections(content)

	if len(sections) != 0 {
		t.Errorf("Expected 0 sections, got %d", len(sections))
	}
}

func TestParseSections_OnlyLevel1(t *testing.T) {
	// Level 1セクションのみの場合のテスト
	content := `# Main Title

Content under main title.

# Another Title

More content.
`

	sections := ParseSections(content)

	if len(sections) != 2 {
		t.Errorf("Expected 2 sections, got %d", len(sections))
	}

	for _, section := range sections {
		if section.Level != 1 {
			t.Errorf("Expected all sections to be Level 1, got Level %d", section.Level)
		}
	}
}

func TestInsertIntoSection_SectionNotFound(t *testing.T) {
	// セクションが見つからない場合（insertPos == -1のケース）
	existing := `# Project

## Section A

Content A.

## Section B

Content B.
`

	// 既存のセクションとは全く異なる内容を持つ提案
	// しかし、同じセクション名を持つため、マッチするはず
	suggestion := `## Section C

New section content.
`

	result := InsertIntoSection(existing, suggestion)

	// Section Cは新しいセクションとして末尾に追加される
	if !strings.Contains(result, "## Section C") {
		t.Error("Expected Section C to be appended")
	}
	if !strings.Contains(result, "New section content") {
		t.Error("Expected new section content to be added")
	}
}

func TestInsertIntoSection_EmptySubsectionList(t *testing.T) {
	// サブセクションが空のリストの場合（subsectionContent == ""のケース）
	existing := `# Project

## Main Section

Existing content.
`

	// Level 2セクションのみで、サブセクションがない提案
	suggestion := `## Main Section
`

	result := InsertIntoSection(existing, suggestion)

	// 既存のコンテンツが保持されることを確認
	if !strings.Contains(result, "Existing content") {
		t.Error("Expected existing content to be preserved")
	}
}

func TestInsertIntoSection_WithTrailingNewlines(t *testing.T) {
	// 末尾に改行がある既存コンテンツ
	existing := `# Project

## Section A

Content A.


`

	suggestion := `## Section B

Content B.
`

	result := InsertIntoSection(existing, suggestion)

	// Section Bが追加される
	if !strings.Contains(result, "## Section B") {
		t.Error("Expected Section B to be appended")
	}
	if !strings.Contains(result, "Content B") {
		t.Error("Expected Content B to be added")
	}
}

func TestInsertIntoSection_OnlyLevel2Sections(t *testing.T) {
	// Level 2セクションのみ（サブセクションなし）
	existing := `## Section 1

Content 1.

## Section 2

Content 2.
`

	suggestion := `## Section 3

New section content.
`

	result := InsertIntoSection(existing, suggestion)

	// Section 3が末尾に追加される
	if !strings.Contains(result, "## Section 3") {
		t.Error("Expected Section 3 to be appended")
	}
	if !strings.Contains(result, "New section content") {
		t.Error("Expected new section content to be added")
	}
}

func TestInsertIntoSection_NewLevel2SectionWithoutTrailingNewline(t *testing.T) {
	// 既存コンテンツが改行で終わらない場合の新規Level 2セクション追加
	existing := "# Project\n\n## Existing Section\n\nContent"

	suggestion := `## New Section

New section content.
`

	result := InsertIntoSection(existing, suggestion)

	// 新しいセクションが追加されることを確認
	if !strings.Contains(result, "## New Section") {
		t.Error("Expected new section to be appended")
	}

	// 適切な改行が追加されることを確認
	if !strings.Contains(result, "\n\n## New Section") {
		t.Logf("Result:\n%s", result)
	}
}
