// Package traeagent implements the Agent interface for Trae Agent.
package traeagent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// TrajectoryEvent represents a single event in Trae Agent's trajectory

type TrajectoryEvent struct {
	Type       string          `json:"type"`
	Timestamp  string          `json:"timestamp"`
	EventID    string          `json:"event_id"`
	Content    json.RawMessage `json:"content"`
	ToolName   string          `json:"tool_name,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`
	ToolOutput json.RawMessage `json:"tool_output,omitempty"`
	ModelName  string          `json:"model_name,omitempty"`
	Prompt     string          `json:"prompt,omitempty"`
	Response   string          `json:"response,omitempty"`
	TokenUsage json.RawMessage `json:"token_usage,omitempty"`
}

// Trajectory represents the complete trajectory of a Trae Agent session
type Trajectory struct {
	SessionID string            `json:"session_id"`
	StartTime string            `json:"start_time"`
	EndTime   string            `json:"end_time,omitempty"`
	Events    []TrajectoryEvent `json:"events"`
}

// ParseTrajectory parses raw JSON content into a Trajectory object
func ParseTrajectory(data []byte) (*Trajectory, error) {
	var trajectory Trajectory
	if err := json.Unmarshal(data, &trajectory); err != nil {
		return nil, fmt.Errorf("failed to parse trajectory: %w", err)
	}
	return &trajectory, nil
}

// extractModifiedFiles extracts files modified by tool calls from trajectory
func extractModifiedFiles(data []byte) ([]string, error) {
	trajectory, err := ParseTrajectory(data)
	if err != nil {
		return []string{}, err // Return empty slice with error if parsing fails
	}

	fileSet := make(map[string]bool)
	var files []string

	for _, event := range trajectory.Events {
		// Check for tool execution events
		if event.Type == "tool_execution" || event.Type == "tool_result" {
			// Check if it's a file modification tool
			isModifyTool := false
			for _, name := range FileModificationTools {
				if event.ToolName == name {
					isModifyTool = true
					break
				}
			}

			if !isModifyTool {
				continue
			}

			// Try to extract file path from tool input
			var toolInput struct {
				FilePath string `json:"file_path,omitempty"`
				Path     string `json:"path,omitempty"`
			}

			if err := json.Unmarshal(event.ToolInput, &toolInput); err == nil {
				file := toolInput.FilePath
				if file == "" {
					file = toolInput.Path
				}

				if file != "" && !fileSet[file] {
					fileSet[file] = true
					files = append(files, file)
				}
			}
		}
	}

	return files, nil
}

// FileModificationTools lists the tools that modify files
var FileModificationTools = []string{
	"str_replace_based_edit_tool",
	"edit_tool",
	"write_file",
	"update_file",
	"delete_file",
}

// TranscriptAnalyzer interface implementation

// GetTranscriptPosition returns the current event count of a Trae Agent trajectory.
// Trae Agent uses JSON format with an events array, so position is the number of events.
func (t *TraeAgent) GetTranscriptPosition(path string) (int, error) {
	if path == "" {
		return 0, nil
	}

	file, err := os.Open(path) //nolint:gosec // Path comes from Trae Agent trajectory location
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to open trajectory file: %w", err)
	}
	defer file.Close()

	// Read the entire file and parse it to get the event count
	data, err := os.ReadFile(path) //nolint:gosec // Path comes from Trae Agent trajectory location
	if err != nil {
		return 0, fmt.Errorf("failed to read trajectory file: %w", err)
	}

	var trajectory Trajectory
	if err := json.Unmarshal(data, &trajectory); err != nil {
		return 0, fmt.Errorf("failed to parse trajectory: %w", err)
	}

	return len(trajectory.Events), nil
}

// ExtractModifiedFilesFromOffset extracts files modified since a given event index.
// For Trae Agent (JSON format), offset is the starting event index.
func (t *TraeAgent) ExtractModifiedFilesFromOffset(path string, startOffset int) (files []string, currentPosition int, err error) {
	if path == "" {
		return nil, 0, nil
	}

	file, openErr := os.Open(path) //nolint:gosec // Path comes from Trae Agent trajectory location
	if openErr != nil {
		return nil, 0, fmt.Errorf("failed to open trajectory file: %w", openErr)
	}
	defer file.Close()

	// Read the entire file and parse it
	data, err := os.ReadFile(path) //nolint:gosec // Path comes from Trae Agent trajectory location
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read trajectory file: %w", err)
	}

	var trajectory Trajectory
	if err := json.Unmarshal(data, &trajectory); err != nil {
		return nil, 0, fmt.Errorf("failed to parse trajectory: %w", err)
	}

	currentPosition = len(trajectory.Events)
	if startOffset >= currentPosition {
		return nil, currentPosition, nil
	}

	// Extract events from startOffset onwards
	relevantEvents := trajectory.Events[startOffset:]

	// Create a new trajectory with only relevant events
	partialTrajectory := Trajectory{
		SessionID: trajectory.SessionID,
		StartTime: trajectory.StartTime,
		EndTime:   trajectory.EndTime,
		Events:    relevantEvents,
	}

	// Serialize partial trajectory and extract modified files
	partialData, err := json.Marshal(partialTrajectory)
	if err != nil {
		return nil, currentPosition, fmt.Errorf("failed to marshal partial trajectory: %w", err)
	}

	modifiedFiles, err := ExtractModifiedFiles(partialData)
	if err != nil {
		return nil, currentPosition, fmt.Errorf("failed to extract modified files: %w", err)
	}

	return modifiedFiles, currentPosition, nil
}

// TranscriptChunker interface implementation

// ChunkTranscript splits a JSON trajectory into chunks if it exceeds maxSize.
// For JSON format, we split the events array into chunks while preserving valid JSON.
func (t *TraeAgent) ChunkTranscript(_ context.Context, content []byte, maxSize int) ([][]byte, error) {
	// If content is smaller than maxSize, return as single chunk
	if len(content) <= maxSize {
		return [][]byte{content}, nil
	}

	// Parse the trajectory to split events
	var trajectory Trajectory
	if err := json.Unmarshal(content, &trajectory); err != nil {
		return nil, fmt.Errorf("failed to parse trajectory: %w", err)
	}

	var chunks [][]byte
	currentChunk := Trajectory{
		SessionID: trajectory.SessionID,
		StartTime: trajectory.StartTime,
		Events:    []TrajectoryEvent{},
	}

	for _, event := range trajectory.Events {
		// Add event to current chunk
		currentChunk.Events = append(currentChunk.Events, event)

		// Check if current chunk exceeds maxSize
		chunkData, err := json.Marshal(currentChunk)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal chunk: %w", err)
		}

		if len(chunkData) > maxSize {
			// Remove the last event (it caused the overflow)
			currentChunk.Events = currentChunk.Events[:len(currentChunk.Events)-1]

			// Marshal and add the chunk
			finalChunkData, err := json.Marshal(currentChunk)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal final chunk: %w", err)
			}
			chunks = append(chunks, finalChunkData)

			// Start a new chunk with the last event
			currentChunk = Trajectory{
				SessionID: trajectory.SessionID,
				StartTime: trajectory.StartTime,
				Events:    []TrajectoryEvent{event},
			}
		}
	}

	// Add the remaining chunk
	if len(currentChunk.Events) > 0 {
		chunkData, err := json.Marshal(currentChunk)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal remaining chunk: %w", err)
		}
		chunks = append(chunks, chunkData)
	}

	return chunks, nil
}

// ReassembleTranscript combines chunks back into a single trajectory.
// For JSON format, we merge the events arrays from all chunks.
func (t *TraeAgent) ReassembleTranscript(chunks [][]byte) ([]byte, error) {
	if len(chunks) == 0 {
		return []byte("{}"), nil
	}

	// Parse the first chunk to get the base trajectory
	var mergedTrajectory Trajectory
	if err := json.Unmarshal(chunks[0], &mergedTrajectory); err != nil {
		return nil, fmt.Errorf("failed to parse first chunk: %w", err)
	}

	// Merge events from remaining chunks
	for i := 1; i < len(chunks); i++ {
		var chunk Trajectory
		if err := json.Unmarshal(chunks[i], &chunk); err != nil {
			return nil, fmt.Errorf("failed to parse chunk %d: %w", i, err)
		}
		mergedTrajectory.Events = append(mergedTrajectory.Events, chunk.Events...)
	}

	// Set end time from the last chunk if available
	var lastChunk Trajectory
	if err := json.Unmarshal(chunks[len(chunks)-1], &lastChunk); err == nil {
		mergedTrajectory.EndTime = lastChunk.EndTime
	}

	// Serialize the merged trajectory
	mergedData, err := json.Marshal(mergedTrajectory)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged trajectory: %w", err)
	}

	return mergedData, nil
}

// ExtractAllUserPrompts extracts all user prompts from a trajectory.
func ExtractAllUserPrompts(data []byte) ([]string, error) {
	trajectory, err := ParseTrajectory(data)
	if err != nil {
		return nil, err
	}

	var prompts []string
	for _, event := range trajectory.Events {
		if event.Type == "user_message" || event.Type == "user" {
			var content struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(event.Content, &content); err == nil && content.Text != "" {
				prompts = append(prompts, content.Text)
			}
		}
		// Also check for prompt field directly in event
		if event.Prompt != "" {
			prompts = append(prompts, event.Prompt)
		}
	}

	return prompts, nil
}

// ExtractLastAssistantMessage extracts the last assistant message from a trajectory.
func ExtractLastAssistantMessage(data []byte) (string, error) {
	trajectory, err := ParseTrajectory(data)
	if err != nil {
		return "", err
	}

	// Iterate in reverse to find the last assistant message
	for i := len(trajectory.Events) - 1; i >= 0; i-- {
		event := trajectory.Events[i]
		if event.Type == "assistant_message" || event.Type == "assistant" {
			var content struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(event.Content, &content); err == nil && content.Text != "" {
				return content.Text, nil
			}
		}
		// Also check for response field directly in event
		if event.Response != "" {
			return event.Response, nil
		}
	}

	return "", nil
}
