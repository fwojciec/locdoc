package locdoc

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Section represents a heading in a markdown document.
type Section struct {
	Level  int    `json:"level"`
	Title  string `json:"title"`
	Anchor string `json:"anchor"`
}

// ExtractSections parses markdown and returns all headings (H1-H6).
// It generates URL-safe anchors and handles duplicates with numeric suffixes.
func ExtractSections(markdown string) []Section {
	if markdown == "" {
		return nil
	}

	// Remove code blocks to avoid matching # in code
	cleaned := removeCodeBlocks(markdown)

	// Match markdown headings: # through ######
	headingRe := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	matches := headingRe.FindAllStringSubmatch(cleaned, -1)

	if len(matches) == 0 {
		return nil
	}

	sections := make([]Section, 0, len(matches))
	anchorCounts := make(map[string]int)

	for _, match := range matches {
		level := len(match[1])
		title := strings.TrimSpace(match[2])
		baseAnchor := generateAnchor(title)

		// Handle duplicates
		anchor := baseAnchor
		if count, exists := anchorCounts[baseAnchor]; exists {
			anchor = baseAnchor + "-" + strconv.Itoa(count)
			anchorCounts[baseAnchor]++
		} else {
			anchorCounts[baseAnchor] = 1
		}

		sections = append(sections, Section{
			Level:  level,
			Title:  title,
			Anchor: anchor,
		})
	}

	return sections
}

// removeCodeBlocks removes fenced code blocks from markdown.
func removeCodeBlocks(s string) string {
	codeBlockRe := regexp.MustCompile("(?s)```.*?```")
	return codeBlockRe.ReplaceAllString(s, "")
}

// generateAnchor creates a URL-safe anchor from a title.
// Converts to lowercase, replaces spaces with hyphens, removes special chars.
func generateAnchor(title string) string {
	var sb strings.Builder
	prevHyphen := false

	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(r)
			prevHyphen = false
		} else if unicode.IsSpace(r) || r == '-' {
			if !prevHyphen && sb.Len() > 0 {
				sb.WriteRune('-')
				prevHyphen = true
			}
		}
	}

	result := sb.String()
	// Trim trailing hyphen
	return strings.TrimSuffix(result, "-")
}
