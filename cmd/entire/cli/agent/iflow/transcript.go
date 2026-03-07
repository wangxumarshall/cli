package iflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// ParseTranscriptFromBytes parses JSONL transcript data into TranscriptLines.
func ParseTranscriptFromBytes(data []byte) ([]TranscriptLine, error) {
	var lines []TranscriptLine
	scanner := bufio.NewScanner(bytesReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry TranscriptLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed lines
			continue
		}
		lines = append(lines, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan transcript: %w", err)
	}
	return lines, nil
}

// ExtractModifiedFiles extracts file paths from tools that modify files.
func ExtractModifiedFiles(lines []TranscriptLine) []string {
	fileSet := make(map[string]struct{})

	for _, line := range lines {
		if line.ToolUse == nil {
			continue
		}

		if !isFileModificationTool(line.ToolUse.Name) {
			continue
		}

		filePath := extractFilePathFromToolInput(line.ToolUse.Input)
		if filePath != "" {
			fileSet[filePath] = struct{}{}
		}
	}

	// Convert set to sorted slice
	files := make([]string, 0, len(fileSet))
	for file := range fileSet {
		files = append(files, file)
	}
	return files
}

// isFileModificationTool checks if the tool name indicates a file modification operation.
func isFileModificationTool(toolName string) bool {
	for _, t := range FileModificationTools {
		if toolName == t {
			return true
		}
	}
	return false
}

// extractFilePathFromToolInput extracts the file path from tool input JSON.
func extractFilePathFromToolInput(input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}

	// Try FileWriteToolInput structure
	var writeInput FileWriteToolInput
	if err := json.Unmarshal(input, &writeInput); err == nil {
		if writeInput.FilePath != "" {
			return writeInput.FilePath
		}
		if writeInput.Path != "" {
			return writeInput.Path
		}
	}

	// Try FileEditToolInput structure
	var editInput FileEditToolInput
	if err := json.Unmarshal(input, &editInput); err == nil {
		if editInput.FilePath != "" {
			return editInput.FilePath
		}
		if editInput.Path != "" {
			return editInput.Path
		}
	}

	// Try generic map extraction
	var generic map[string]interface{}
	if err := json.Unmarshal(input, &generic); err == nil {
		// Check common field names
		for _, key := range []string{"file_path", "path", "filepath", "file", "filename"} {
			if val, ok := generic[key]; ok {
				if str, ok := val.(string); ok && str != "" {
					return str
				}
			}
		}
	}

	return ""
}

// bytesReader creates an io.Reader from bytes (helper for testing).
func bytesReader(data []byte) io.Reader {
	return strings.NewReader(string(data))
}

// SerializeTranscript converts TranscriptLines back to JSONL format.
func SerializeTranscript(lines []TranscriptLine) ([]byte, error) {
	var result strings.Builder
	encoder := json.NewEncoder(&result)
	encoder.SetEscapeHTML(false)

	for _, line := range lines {
		if err := encoder.Encode(line); err != nil {
			return nil, fmt.Errorf("failed to encode transcript line: %w", err)
		}
	}

	return []byte(result.String()), nil
}

// FindCheckpointLine finds the line index containing a specific tool result.
// Used for checkpoint rewind operations.
func FindCheckpointLine(lines []TranscriptLine, toolUseID string) (int, bool) {
	for i, line := range lines {
		if line.ToolResult != nil && line.ToolResult.ToolUseID == toolUseID {
			return i, true
		}
	}
	return 0, false
}

// TruncateAtLine returns a new transcript truncated at the given line index (inclusive).
func TruncateAtLine(lines []TranscriptLine, lineIndex int) []TranscriptLine {
	if lineIndex < 0 || lineIndex >= len(lines) {
		return lines
	}
	return lines[:lineIndex+1]
}

// TruncateAtToolResult returns a new transcript truncated at the given tool result.
func TruncateAtToolResult(lines []TranscriptLine, toolUseID string) ([]TranscriptLine, bool) {
	lineIndex, found := FindCheckpointLine(lines, toolUseID)
	if !found {
		return nil, false
	}
	return TruncateAtLine(lines, lineIndex), true
}

// GetLastToolUse finds the last tool use in the transcript.
func GetLastToolUse(lines []TranscriptLine) *ToolUse {
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].ToolUse != nil {
			return lines[i].ToolUse
		}
	}
	return nil
}

// GetToolResultForUse finds the tool result for a given tool use ID.
func GetToolResultForUse(lines []TranscriptLine, toolUseID string) *ToolResult {
	for _, line := range lines {
		if line.ToolResult != nil && line.ToolResult.ToolUseID == toolUseID {
			return line.ToolResult
		}
	}
	return nil
}

// CountToolUses counts the occurrences of a specific tool in the transcript.
func CountToolUses(lines []TranscriptLine, toolName string) int {
	count := 0
	for _, line := range lines {
		if line.ToolUse != nil && line.ToolUse.Name == toolName {
			count++
		}
	}
	return count
}

// ReadTranscriptFile reads and parses a transcript file.
func ReadTranscriptFile(path string) ([]TranscriptLine, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read transcript file: %w", err)
	}
	return ParseTranscriptFromBytes(data)
}

// WriteTranscriptFile writes transcript lines to a file.
func WriteTranscriptFile(path string, lines []TranscriptLine) error {
	data, err := SerializeTranscript(lines)
	if err != nil {
		return fmt.Errorf("failed to serialize transcript: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write transcript file: %w", err)
	}

	return nil
}

// MergeTranscripts merges multiple transcripts into one.
// Handles deduplication of lines based on timestamp and content.
func MergeTranscripts(transcripts [][]TranscriptLine) []TranscriptLine {
	seen := make(map[string]struct{})
	var result []TranscriptLine

	for _, transcript := range transcripts {
		for _, line := range transcript {
			// Create a key for deduplication
			key := line.Timestamp + string(line.Message)
			if _, exists := seen[key]; !exists {
				seen[key] = struct{}{}
				result = append(result, line)
			}
		}
	}

	return result
}
