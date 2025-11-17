package main

import (
	"regexp"
	"strings"
)

// Section represents a markdown section with its level and content
type Section struct {
	Level     int // 1 for #, 2 for ##, etc.
	Title     string
	Content   string
	StartLine int
	EndLine   int
}

// ParseSections parses markdown content and extracts sections
func ParseSections(content string) []Section {
	lines := strings.Split(content, "\n")
	var sections []Section
	var currentSection *Section

	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	for i, line := range lines {
		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			// Save previous section
			if currentSection != nil {
				currentSection.EndLine = i - 1
				sections = append(sections, *currentSection)
			}

			// Start new section
			level := len(matches[1])
			title := strings.TrimSpace(matches[2])
			currentSection = &Section{
				Level:     level,
				Title:     title,
				Content:   line + "\n",
				StartLine: i,
			}
		} else if currentSection != nil {
			currentSection.Content += line + "\n"
		}
	}

	// Save last section
	if currentSection != nil {
		currentSection.EndLine = len(lines) - 1
		sections = append(sections, *currentSection)
	}

	return sections
}

// FindSectionByTitle finds a section by its title (case-insensitive, level 2)
func FindSectionByTitle(sections []Section, title string) *Section {
	normalizedTitle := strings.ToLower(strings.TrimSpace(title))
	for i := range sections {
		if sections[i].Level == 2 && strings.ToLower(strings.TrimSpace(sections[i].Title)) == normalizedTitle {
			return &sections[i]
		}
	}
	return nil
}

// InsertIntoSection inserts new content into existing CLAUDE.md at appropriate sections
// nolint:gocyclo,gocritic // Complex logic for section-based insertion is necessary
func InsertIntoSection(existingContent, suggestionContent string) string {
	// If existing content is empty, return suggestion as-is
	if existingContent == "" {
		return suggestionContent
	}

	// Parse existing CLAUDE.md sections
	existingSections := ParseSections(existingContent)

	// Parse suggestion sections
	suggestionSections := ParseSections(suggestionContent)

	if len(suggestionSections) == 0 {
		// No sections found, append to end
		return appendContent(existingContent, suggestionContent)
	}

	// Build a map of level-2 sections and collect their subsections
	type SectionGroup struct {
		Section     *Section
		Subsections []Section
		EndLine     int
	}

	existingGroups := make(map[string]*SectionGroup)
	for i := 0; i < len(existingSections); i++ {
		if existingSections[i].Level == 2 {
			key := strings.ToLower(strings.TrimSpace(existingSections[i].Title))
			group := &SectionGroup{
				Section: &existingSections[i],
			}
			// Collect subsections
			endLine := existingSections[i].EndLine
			for j := i + 1; j < len(existingSections); j++ {
				if existingSections[j].Level <= 2 {
					break
				}
				group.Subsections = append(group.Subsections, existingSections[j])
				endLine = existingSections[j].EndLine
			}
			group.EndLine = endLine
			existingGroups[key] = group
		}
	}

	// Group suggestion sections similarly
	suggestionGroups := make(map[string][]Section)
	var newLevel2Sections []string
	hasLevel2Sections := false
	for i := 0; i < len(suggestionSections); i++ {
		if suggestionSections[i].Level == 2 {
			hasLevel2Sections = true
			key := strings.ToLower(strings.TrimSpace(suggestionSections[i].Title))
			var subsections []Section
			// Collect subsections
			for j := i + 1; j < len(suggestionSections); j++ {
				if suggestionSections[j].Level <= 2 {
					break
				}
				subsections = append(subsections, suggestionSections[j])
			}
			suggestionGroups[key] = subsections

			// If section doesn't exist in original, collect for appending
			if _, exists := existingGroups[key]; !exists {
				// Build the full section content
				sectionContent := suggestionSections[i].Content
				for _, sub := range subsections {
					sectionContent += sub.Content
				}
				newLevel2Sections = append(newLevel2Sections, sectionContent)
			}
		}
	}

	// If no level-2 sections found, just append to end
	if !hasLevel2Sections {
		return appendContent(existingContent, suggestionContent)
	}

	// Insert subsections into existing sections
	result := existingContent
	for key, subsections := range suggestionGroups {
		if group, exists := existingGroups[key]; exists {
			// Build subsection content to insert
			var subsectionContent string
			for _, sub := range subsections {
				subsectionContent += sub.Content
			}

			if subsectionContent == "" {
				continue
			}

			// Find the end of the existing section group in result
			// Use the section content to find insertion point
			fullSectionContent := group.Section.Content
			for _, sub := range group.Subsections {
				fullSectionContent += sub.Content
			}

			insertPos := strings.Index(result, fullSectionContent)
			if insertPos != -1 {
				endPos := insertPos + len(fullSectionContent)

				before := result[:endPos]
				after := result[endPos:]

				// Ensure proper spacing
				if !strings.HasSuffix(before, "\n\n") {
					if strings.HasSuffix(before, "\n") {
						before += "\n"
					} else {
						before += "\n\n"
					}
				}

				result = before + subsectionContent + after
			}
		}
	}

	// Append new level-2 sections at the end
	if len(newLevel2Sections) > 0 {
		if !strings.HasSuffix(result, "\n") {
			result += "\n"
		}
		result += "\n" + strings.Join(newLevel2Sections, "\n")
	}

	return result
}

// extractSubsectionContent extracts content after the first line (section header)
func extractSubsectionContent(sectionContent string) string {
	lines := strings.Split(sectionContent, "\n")
	if len(lines) <= 1 {
		return ""
	}
	return strings.Join(lines[1:], "\n")
}

// appendContent appends new content to existing content with proper spacing
func appendContent(existing, newContent string) string {
	if !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	return existing + "\n" + newContent
}
