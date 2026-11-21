package util

import (
	"encoding/json"
	"strings"
)

// plateNode represents a node in the Plate JSON tree.
type plateNode struct {
	Type     string      `json:"type,omitempty"`
	Text     *string     `json:"text,omitempty"`
	Fallback string      `json:"fallback,omitempty"`
	Children []plateNode `json:"children,omitempty"`
}

// ExtractPlainText walks a Plate JSON tree and extracts all text content.
// For tg_emoji nodes, it uses the fallback Unicode emoji.
// Returns empty string for nil, null, or invalid JSON input.
func ExtractPlainText(plateJSON json.RawMessage) string {
	if len(plateJSON) == 0 {
		return ""
	}

	var nodes []plateNode
	if err := json.Unmarshal(plateJSON, &nodes); err != nil {
		return ""
	}

	var parts []string
	walkNodes(nodes, &parts)

	// Join parts and trim trailing whitespace
	result := strings.Join(parts, "")
	return strings.TrimRight(result, "\n ")
}

func walkNodes(nodes []plateNode, parts *[]string) {
	for _, node := range nodes {
		if node.Text != nil {
			*parts = append(*parts, *node.Text)
		} else if node.Type == "tg_emoji" && node.Fallback != "" {
			*parts = append(*parts, node.Fallback)
		} else if len(node.Children) > 0 {
			walkNodes(node.Children, parts)
			// Add newline after block-level nodes (not list items or list item content)
			if node.Type != "" && node.Type != "lic" && node.Type != "ul" && node.Type != "ol" {
				*parts = append(*parts, "\n")
			}

		}
	}
}
