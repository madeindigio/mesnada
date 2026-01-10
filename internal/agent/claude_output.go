// Package agent handles spawning and managing CLI agent processes.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ClaudeStreamEvent represents a single event from Claude CLI's stream-json output.
type ClaudeStreamEvent struct {
	Type      string                `json:"type"`
	SessionID string                `json:"session_id,omitempty"`
	Timestamp string                `json:"timestamp,omitempty"`
	Role      string                `json:"role,omitempty"`
	Content   []ClaudeContentBlock  `json:"content,omitempty"`
	Output    string                `json:"output,omitempty"`
	Status    string                `json:"status,omitempty"`
	DurationMS int64                `json:"duration_ms,omitempty"`
	Name      string                `json:"name,omitempty"`
	Input     json.RawMessage       `json:"input,omitempty"`
}

// ClaudeContentBlock represents a content block in a Claude message.
type ClaudeContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ClaudeOutputParser parses Claude CLI stream-json output and converts it to text.
type ClaudeOutputParser struct {
	lastEventType string
}

// NewClaudeOutputParser creates a new parser.
func NewClaudeOutputParser() *ClaudeOutputParser {
	return &ClaudeOutputParser{}
}

// ParseLine parses a single JSON line from Claude CLI output.
// Returns the formatted text representation of the event.
func (p *ClaudeOutputParser) ParseLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}

	var event ClaudeStreamEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		// If not valid JSON, return as-is
		return line
	}

	return p.formatEvent(&event)
}

// formatEvent formats a Claude stream event into readable text.
func (p *ClaudeOutputParser) formatEvent(event *ClaudeStreamEvent) string {
	var output strings.Builder

	switch event.Type {
	case "init":
		// Session initialization - minimal output
		if event.SessionID != "" {
			output.WriteString(fmt.Sprintf("Session started: %s\n", event.SessionID))
		}

	case "message":
		// Message from assistant or user
		for _, content := range event.Content {
			switch content.Type {
			case "text":
				if content.Text != "" {
					output.WriteString(content.Text)
					if !strings.HasSuffix(content.Text, "\n") {
						output.WriteString("\n")
					}
				}
			case "tool_use":
				// Tool invocation within a message
				output.WriteString(fmt.Sprintf("[Tool: %s]\n", content.Name))
				if len(content.Input) > 0 {
					// Try to format input nicely
					var inputMap map[string]interface{}
					if err := json.Unmarshal(content.Input, &inputMap); err == nil {
						for k, v := range inputMap {
							output.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
						}
					}
				}
			}
		}

	case "tool_use":
		// Standalone tool use event
		output.WriteString(fmt.Sprintf("[Tool: %s]\n", event.Name))
		if len(event.Input) > 0 {
			var inputMap map[string]interface{}
			if err := json.Unmarshal(event.Input, &inputMap); err == nil {
				for k, v := range inputMap {
					// Format based on common tool parameters
					switch k {
					case "command":
						output.WriteString(fmt.Sprintf("  $ %v\n", v))
					case "file_path", "path":
						output.WriteString(fmt.Sprintf("  File: %v\n", v))
					case "content", "new_source":
						// Truncate long content
						str := fmt.Sprintf("%v", v)
						if len(str) > 200 {
							str = str[:200] + "..."
						}
						output.WriteString(fmt.Sprintf("  Content: %s\n", str))
					default:
						output.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
					}
				}
			}
		}

	case "tool_result":
		// Tool execution result
		if event.Output != "" {
			// Prefix each line with output indicator
			lines := strings.Split(event.Output, "\n")
			for _, line := range lines {
				if line != "" {
					output.WriteString(fmt.Sprintf("  > %s\n", line))
				}
			}
		}

	case "result":
		// Final result
		if event.Status != "" {
			output.WriteString(fmt.Sprintf("\n[Status: %s", event.Status))
			if event.DurationMS > 0 {
				output.WriteString(fmt.Sprintf(", Duration: %dms", event.DurationMS))
			}
			output.WriteString("]\n")
		}

	case "error":
		// Error event
		if event.Output != "" {
			output.WriteString(fmt.Sprintf("[Error] %s\n", event.Output))
		}

	default:
		// Unknown event type - output raw content if any
		for _, content := range event.Content {
			if content.Text != "" {
				output.WriteString(content.Text)
				if !strings.HasSuffix(content.Text, "\n") {
					output.WriteString("\n")
				}
			}
		}
	}

	p.lastEventType = event.Type
	return output.String()
}

// ParseMultiLine parses multiple lines of Claude CLI output.
func (p *ClaudeOutputParser) ParseMultiLine(input string) string {
	var output strings.Builder
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		parsed := p.ParseLine(line)
		if parsed != "" {
			output.WriteString(parsed)
		}
	}

	return output.String()
}
