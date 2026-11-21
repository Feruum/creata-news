package util

import (
	"encoding/json"
	"testing"
)

func TestExtractPlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple paragraph",
			input:    `[{"type":"p","children":[{"text":"Hello world"}]}]`,
			expected: "Hello world",
		},
		{
			name:     "multiple paragraphs",
			input:    `[{"type":"p","children":[{"text":"First"}]},{"type":"p","children":[{"text":"Second"}]}]`,
			expected: "First\nSecond",
		},
		{
			name:     "bold and italic marks",
			input:    `[{"type":"p","children":[{"text":"Normal "},{"text":"bold","bold":true},{"text":" text"}]}]`,
			expected: "Normal bold text",
		},
		{
			name:     "heading",
			input:    `[{"type":"h1","children":[{"text":"Title"}]},{"type":"p","children":[{"text":"Body"}]}]`,
			expected: "Title\nBody",
		},
		{
			name:     "tg_emoji with fallback",
			input:    `[{"type":"p","children":[{"type":"tg_emoji","emojiId":"123","fallback":"👍","children":[{"text":""}]},{"text":" Great"}]}]`,
			expected: "👍 Great",
		},
		{
			name:     "nested list",
			input:    `[{"type":"ul","children":[{"type":"li","children":[{"type":"lic","children":[{"text":"Item 1"}]}]},{"type":"li","children":[{"type":"lic","children":[{"text":"Item 2"}]}]}]}]`,
			expected: "Item 1\nItem 2",
		},
		{
			name:     "empty value",
			input:    `[{"type":"p","children":[{"text":""}]}]`,
			expected: "",
		},
		{
			name:     "null input",
			input:    "null",
			expected: "",
		},
		{
			name:     "invalid JSON",
			input:    "not json",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractPlainText(json.RawMessage(tt.input))
			if result != tt.expected {
				t.Errorf("ExtractPlainText(%s) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
