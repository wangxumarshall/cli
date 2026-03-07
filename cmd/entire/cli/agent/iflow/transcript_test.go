package iflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTranscriptFromBytes(t *testing.T) {
	transcript := `{"type": "user", "timestamp": "2024-01-01T00:00:00Z", "message": {}}
{"type": "assistant", "timestamp": "2024-01-01T00:00:01Z", "message": {}}
{"type": "tool_use", "timestamp": "2024-01-01T00:00:02Z", "tool_use": {"id": "tool-1", "name": "write_file", "input": {"file_path": "test.txt"}}}`

	lines, err := ParseTranscriptFromBytes([]byte(transcript))
	if err != nil {
		t.Fatalf("ParseTranscriptFromBytes failed: %v", err)
	}

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

	if lines[2].ToolUse == nil {
		t.Fatal("Expected tool use on line 3")
	}

	if lines[2].ToolUse.Name != "write_file" {
		t.Errorf("Expected tool name 'write_file', got %q", lines[2].ToolUse.Name)
	}
}

func TestParseTranscriptFromBytes_Empty(t *testing.T) {
	lines, err := ParseTranscriptFromBytes([]byte(""))
	if err != nil {
		t.Fatalf("ParseTranscriptFromBytes failed: %v", err)
	}

	if len(lines) != 0 {
		t.Errorf("Expected 0 lines for empty input, got %d", len(lines))
	}
}

func TestParseTranscriptFromBytes_InvalidJSON(t *testing.T) {
	// Invalid JSON lines should be skipped
	transcript := `{"type": "user", "timestamp": "2024-01-01T00:00:00Z"}
invalid json line
{"type": "assistant", "timestamp": "2024-01-01T00:00:01Z"}`

	lines, err := ParseTranscriptFromBytes([]byte(transcript))
	if err != nil {
		t.Fatalf("ParseTranscriptFromBytes failed: %v", err)
	}

	if len(lines) != 2 {
		t.Errorf("Expected 2 valid lines, got %d", len(lines))
	}
}

func TestExtractModifiedFiles(t *testing.T) {
	lines := []TranscriptLine{
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:00Z",
			ToolUse: &ToolUse{
				ID:   "tool-1",
				Name: "write_file",
				Input: json.RawMessage(`{"file_path": "file1.txt"}`),
			},
		},
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:01Z",
			ToolUse: &ToolUse{
				ID:   "tool-2",
				Name: "replace",
				Input: json.RawMessage(`{"path": "file2.txt"}`),
			},
		},
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:02Z",
			ToolUse: &ToolUse{
				ID:   "tool-3",
				Name: "read_file",
				Input: json.RawMessage(`{"file_path": "file3.txt"}`),
			},
		},
	}

	files := ExtractModifiedFiles(lines)

	if len(files) != 2 {
		t.Errorf("Expected 2 modified files, got %d", len(files))
	}

	// Check that file1.txt and file2.txt are in the list
	found := make(map[string]bool)
	for _, f := range files {
		found[f] = true
	}

	if !found["file1.txt"] {
		t.Error("Expected file1.txt in modified files")
	}
	if !found["file2.txt"] {
		t.Error("Expected file2.txt in modified files")
	}
	if found["file3.txt"] {
		t.Error("file3.txt should not be in modified files (read_file is not a modification tool)")
	}
}

func TestExtractModifiedFiles_Duplicates(t *testing.T) {
	lines := []TranscriptLine{
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:00Z",
			ToolUse: &ToolUse{
				ID:   "tool-1",
				Name: "write_file",
				Input: json.RawMessage(`{"file_path": "same.txt"}`),
			},
		},
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:01Z",
			ToolUse: &ToolUse{
				ID:   "tool-2",
				Name: "write_file",
				Input: json.RawMessage(`{"file_path": "same.txt"}`),
			},
		},
	}

	files := ExtractModifiedFiles(lines)

	if len(files) != 1 {
		t.Errorf("Expected 1 unique file, got %d", len(files))
	}

	if files[0] != "same.txt" {
		t.Errorf("Expected 'same.txt', got %q", files[0])
	}
}

func TestIsFileModificationTool(t *testing.T) {
	tests := []struct {
		toolName string
		expected bool
	}{
		{"write_file", true},
		{"replace", true},
		{"multi_edit", true},
		{"read_file", false},
		{"run_shell_command", false},
		{"list_directory", false},
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			result := isFileModificationTool(tt.toolName)
			if result != tt.expected {
				t.Errorf("isFileModificationTool(%q) = %v, want %v", tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestExtractFilePathFromToolInput(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "file_path field",
			input:    map[string]interface{}{"file_path": "test.txt"},
			expected: "test.txt",
		},
		{
			name:     "path field",
			input:    map[string]interface{}{"path": "src/main.go"},
			expected: "src/main.go",
		},
		{
			name:     "filepath field",
			input:    map[string]interface{}{"filepath": "docs/readme.md"},
			expected: "docs/readme.md",
		},
		{
			name:     "no file path",
			input:    map[string]interface{}{"content": "hello"},
			expected: "",
		},
		{
			name:     "empty input",
			input:    map[string]interface{}{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := json.Marshal(tt.input)
			result := extractFilePathFromToolInput(data)
			if result != tt.expected {
				t.Errorf("extractFilePathFromToolInput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSerializeTranscript(t *testing.T) {
	lines := []TranscriptLine{
		{
			Type:      "user",
			Timestamp: "2024-01-01T00:00:00Z",
		},
		{
			Type:      "assistant",
			Timestamp: "2024-01-01T00:00:01Z",
		},
	}

	data, err := SerializeTranscript(lines)
	if err != nil {
		t.Fatalf("SerializeTranscript failed: %v", err)
	}

	// Parse it back to verify
	parsed, err := ParseTranscriptFromBytes(data)
	if err != nil {
		t.Fatalf("ParseTranscriptFromBytes failed: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("Expected 2 lines after round-trip, got %d", len(parsed))
	}
}

func TestFindCheckpointLine(t *testing.T) {
	lines := []TranscriptLine{
		{Type: "user", Timestamp: "2024-01-01T00:00:00Z"},
		{
			Type:        "tool_result",
			Timestamp:   "2024-01-01T00:00:01Z",
			ToolResult:  &ToolResult{ToolUseID: "tool-1"},
		},
		{Type: "assistant", Timestamp: "2024-01-01T00:00:02Z"},
		{
			Type:        "tool_result",
			Timestamp:   "2024-01-01T00:00:03Z",
			ToolResult:  &ToolResult{ToolUseID: "tool-2"},
		},
	}

	index, found := FindCheckpointLine(lines, "tool-2")
	if !found {
		t.Error("Expected to find tool-2")
	}
	if index != 3 {
		t.Errorf("Expected index 3, got %d", index)
	}

	_, found = FindCheckpointLine(lines, "non-existent")
	if found {
		t.Error("Expected not to find non-existent tool")
	}
}

func TestTruncateAtLine(t *testing.T) {
	lines := []TranscriptLine{
		{Type: "user", Timestamp: "2024-01-01T00:00:00Z"},
		{Type: "assistant", Timestamp: "2024-01-01T00:00:01Z"},
		{Type: "user", Timestamp: "2024-01-01T00:00:02Z"},
	}

	truncated := TruncateAtLine(lines, 1)
	if len(truncated) != 2 {
		t.Errorf("Expected 2 lines after truncate, got %d", len(truncated))
	}

	// Test out of bounds
	truncated = TruncateAtLine(lines, 10)
	if len(truncated) != 3 {
		t.Errorf("Expected original lines for out of bounds, got %d", len(truncated))
	}
}

func TestReadTranscriptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	content := `{"type": "user", "timestamp": "2024-01-01T00:00:00Z"}
{"type": "assistant", "timestamp": "2024-01-01T00:00:01Z"}`

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	lines, err := ReadTranscriptFile(path)
	if err != nil {
		t.Fatalf("ReadTranscriptFile failed: %v", err)
	}

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}
}

func TestWriteTranscriptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	lines := []TranscriptLine{
		{Type: "user", Timestamp: "2024-01-01T00:00:00Z"},
		{Type: "assistant", Timestamp: "2024-01-01T00:00:01Z"},
	}

	if err := WriteTranscriptFile(path, lines); err != nil {
		t.Fatalf("WriteTranscriptFile failed: %v", err)
	}

	// Read it back
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseTranscriptFromBytes(data)
	if err != nil {
		t.Fatalf("ParseTranscriptFromBytes failed: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("Expected 2 lines after write/read, got %d", len(parsed))
	}
}

func TestMergeTranscripts(t *testing.T) {
	transcript1 := []TranscriptLine{
		{Type: "user", Timestamp: "2024-01-01T00:00:00Z"},
		{Type: "assistant", Timestamp: "2024-01-01T00:00:01Z"},
	}

	transcript2 := []TranscriptLine{
		{Type: "assistant", Timestamp: "2024-01-01T00:00:01Z"}, // Duplicate
		{Type: "user", Timestamp: "2024-01-01T00:00:02Z"},
	}

	merged := MergeTranscripts([][]TranscriptLine{transcript1, transcript2})

	if len(merged) != 3 {
		t.Errorf("Expected 3 unique lines after merge, got %d", len(merged))
	}
}

func TestGetLastToolUse(t *testing.T) {
	lines := []TranscriptLine{
		{Type: "user", Timestamp: "2024-01-01T00:00:00Z"},
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:01Z",
			ToolUse:   &ToolUse{ID: "tool-1", Name: "write_file"},
		},
		{Type: "assistant", Timestamp: "2024-01-01T00:00:02Z"},
	}

	toolUse := GetLastToolUse(lines)
	if toolUse == nil {
		t.Fatal("Expected to find last tool use")
	}
	if toolUse.ID != "tool-1" {
		t.Errorf("Expected tool-1, got %q", toolUse.ID)
	}
}

func TestGetToolResultForUse(t *testing.T) {
	lines := []TranscriptLine{
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:00Z",
			ToolUse:   &ToolUse{ID: "tool-1", Name: "write_file"},
		},
		{
			Type:       "tool_result",
			Timestamp:  "2024-01-01T00:00:01Z",
			ToolResult: &ToolResult{ToolUseID: "tool-1"},
		},
	}

	result := GetToolResultForUse(lines, "tool-1")
	if result == nil {
		t.Fatal("Expected to find tool result")
	}
	if result.ToolUseID != "tool-1" {
		t.Errorf("Expected tool_use_id tool-1, got %q", result.ToolUseID)
	}

	result = GetToolResultForUse(lines, "non-existent")
	if result != nil {
		t.Error("Expected nil for non-existent tool use")
	}
}

func TestCountToolUses(t *testing.T) {
	lines := []TranscriptLine{
		{Type: "user", Timestamp: "2024-01-01T00:00:00Z"},
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:01Z",
			ToolUse:   &ToolUse{ID: "tool-1", Name: "write_file"},
		},
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:02Z",
			ToolUse:   &ToolUse{ID: "tool-2", Name: "write_file"},
		},
		{
			Type:      "tool_use",
			Timestamp: "2024-01-01T00:00:03Z",
			ToolUse:   &ToolUse{ID: "tool-3", Name: "read_file"},
		},
	}

	count := CountToolUses(lines, "write_file")
	if count != 2 {
		t.Errorf("Expected 2 write_file uses, got %d", count)
	}

	count = CountToolUses(lines, "read_file")
	if count != 1 {
		t.Errorf("Expected 1 read_file use, got %d", count)
	}

	count = CountToolUses(lines, "non-existent")
	if count != 0 {
		t.Errorf("Expected 0 uses for non-existent tool, got %d", count)
	}
}
