// Package diff provides edit-diff utilities matching TS harness/tools/edit-diff.ts.
package diff

import (
	"fmt"
	"strings"
	"unicode"
)

// Edit represents a single text replacement.
type Edit struct {
	OldText string
	NewText string
}

// FuzzyMatchResult holds the result of fuzzy text matching.
type FuzzyMatchResult struct {
	Found                 bool
	Index                 int
	MatchLength           int
	UsedFuzzyMatch        bool
	ContentForReplacement string
}

// DetectLineEnding detects the line ending style.
func DetectLineEnding(content string) string {
	if strings.Contains(content, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

// NormalizeToLF converts all line endings to LF.
func NormalizeToLF(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

// RestoreLineEndings converts LF back to original line ending style.
func RestoreLineEndings(text, ending string) string {
	if ending == "\r\n" {
		return strings.ReplaceAll(text, "\n", "\r\n")
	}
	return text
}

// NormalizeForFuzzyMatch normalizes text for fuzzy matching.
func NormalizeForFuzzyMatch(text string) string {
	// Strip trailing whitespace from each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	text = strings.Join(lines, "\n")

	// Smart quotes → ASCII
	replacer := strings.NewReplacer(
		"\u2018", "'", "\u2019", "'", "\u201A", "'", "\u201B", "'",
		"\u201C", "\"", "\u201D", "\"", "\u201E", "\"", "\u201F", "\"",
		"\u2010", "-", "\u2011", "-", "\u2012", "-", "\u2013", "-",
		"\u2014", "-", "\u2015", "-", "\u2212", "-",
		"\u00A0", " ", "\u2002", " ", "\u2003", " ", "\u2004", " ",
		"\u2005", " ", "\u2006", " ", "\u2007", " ", "\u2008", " ",
		"\u2009", " ", "\u200A", " ", "\u202F", " ", "\u205F", " ",
		"\u3000", " ",
	)
	return replacer.Replace(text)
}

// FuzzyFindText finds oldText in content, trying exact match first, then fuzzy.
func FuzzyFindText(content, oldText string) FuzzyMatchResult {
	idx := strings.Index(content, oldText)
	if idx != -1 {
		return FuzzyMatchResult{
			Found: true, Index: idx, MatchLength: len(oldText),
			UsedFuzzyMatch: false, ContentForReplacement: content,
		}
	}

	fuzzyContent := NormalizeForFuzzyMatch(content)
	fuzzyOldText := NormalizeForFuzzyMatch(oldText)
	fuzzyIdx := strings.Index(fuzzyContent, fuzzyOldText)
	if fuzzyIdx == -1 {
		return FuzzyMatchResult{Found: false}
	}

	return FuzzyMatchResult{
		Found: true, Index: fuzzyIdx, MatchLength: len(fuzzyOldText),
		UsedFuzzyMatch: true, ContentForReplacement: fuzzyContent,
	}
}

func countOccurrences(content, oldText string) int {
	fuzzyContent := NormalizeForFuzzyMatch(content)
	fuzzyOldText := NormalizeForFuzzyMatch(oldText)
	return strings.Count(fuzzyContent, fuzzyOldText)
}

func getNotFoundError(path string, editIndex, totalEdits int) error {
	if totalEdits == 1 {
		return fmt.Errorf("could not find the exact text in %s. The old text must match exactly including all whitespace and newlines", path)
	}
	return fmt.Errorf("could not find edits[%d] in %s. The oldText must match exactly", editIndex, path)
}

type matchedEdit struct {
	editIndex   int
	matchIndex  int
	matchLength int
	newText     string
}

// ApplyEdits applies edits to normalized content, returning the result.
func ApplyEdits(normalizedContent string, edits []Edit, path string) (string, error) {
	normalizedEdits := make([]Edit, len(edits))
	for i, e := range edits {
		normalizedEdits[i] = Edit{OldText: NormalizeToLF(e.OldText), NewText: NormalizeToLF(e.NewText)}
	}

	for i, e := range normalizedEdits {
		if len(e.OldText) == 0 {
			return "", fmt.Errorf("oldText must not be empty in %s", path)
		}
		_ = i
	}

	initialMatches := make([]FuzzyMatchResult, len(normalizedEdits))
	usedFuzzy := false
	for i, e := range normalizedEdits {
		initialMatches[i] = FuzzyFindText(normalizedContent, e.OldText)
		if initialMatches[i].UsedFuzzyMatch {
			usedFuzzy = true
		}
	}

	baseContent := normalizedContent
	if usedFuzzy {
		baseContent = NormalizeForFuzzyMatch(normalizedContent)
	}

	var matched []matchedEdit
	for i, e := range normalizedEdits {
		match := FuzzyFindText(baseContent, e.OldText)
		if !match.Found {
			return "", getNotFoundError(path, i, len(normalizedEdits))
		}
		occ := countOccurrences(baseContent, e.OldText)
		if occ > 1 {
			return "", fmt.Errorf("found %d occurrences of the text in %s. The text must be unique. Please provide more context", occ, path)
		}
		matched = append(matched, matchedEdit{i, match.Index, match.MatchLength, e.NewText})
	}

	// Sort and check for overlaps
	for i := 0; i < len(matched); i++ {
		for j := i + 1; j < len(matched); j++ {
			if matched[i].matchIndex > matched[j].matchIndex {
				matched[i], matched[j] = matched[j], matched[i]
			}
		}
	}
	for i := 1; i < len(matched); i++ {
		prev := matched[i-1]
		curr := matched[i]
		if prev.matchIndex+prev.matchLength > curr.matchIndex {
			return "", fmt.Errorf("edits[%d] and edits[%d] overlap. Merge them into one edit", prev.editIndex, curr.editIndex)
		}
	}

	// Apply replacements in reverse order
	result := baseContent
	for i := len(matched) - 1; i >= 0; i-- {
		m := matched[i]
		result = result[:m.matchIndex] + m.newText + result[m.matchIndex+m.matchLength:]
	}

	if result == baseContent {
		return "", fmt.Errorf("no changes made to %s. The replacements produced identical content", path)
	}

	return result, nil
}

// GenerateDiffString generates a display-oriented diff with line numbers.
func GenerateDiffString(oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	var output []string
	oldIdx, newIdx := 0, 0
	firstChangedFound := false

	for oldIdx < len(oldLines) || newIdx < len(newLines) {
		if oldIdx < len(oldLines) && newIdx < len(newLines) && oldLines[oldIdx] == newLines[newIdx] {
			if !firstChangedFound {
				output = append(output, fmt.Sprintf("  %s", oldLines[oldIdx]))
			}
			oldIdx++
			newIdx++
			continue
		}
		firstChangedFound = true

		if oldIdx < len(oldLines) {
			output = append(output, fmt.Sprintf("- %s", oldLines[oldIdx]))
			oldIdx++
		}
		if newIdx < len(newLines) {
			output = append(output, fmt.Sprintf("+ %s", newLines[newIdx]))
			newIdx++
		}
	}

	return strings.Join(output, "\n")
}

// StripBOM removes UTF-8 BOM if present.
func StripBOM(content string) (bom, text string) {
	if strings.HasPrefix(content, "\uFEFF") {
		return "\uFEFF", content[1:]
	}
	return "", content
}

// Ensure unicode is used
var _ = unicode.MaxRune
