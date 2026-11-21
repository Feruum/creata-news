package sundayopen

import (
	"encoding/json"
	"strings"
)

// publicDescriptionSQL projects the same visible text as publicDescription.
// It traverses editor children/content only; attributes, type names, property
// names and other serializer metadata never become searchable content.
// Paths preserve document order, with block separators ordered after children.
const publicDescriptionSQL = `(WITH RECURSIVE description_nodes(value, node_path) AS (
	SELECT e.description::jsonb, ARRAY[]::integer[]
	UNION ALL
	SELECT child.value, parent.node_path || child.position::integer
	FROM description_nodes AS parent
	CROSS JOIN LATERAL jsonb_array_elements(CASE
		WHEN jsonb_typeof(parent.value) = 'array' THEN parent.value
		WHEN jsonb_typeof(parent.value->'text') = 'string' THEN '[]'::jsonb
		WHEN parent.value->>'type' = 'tg_emoji' AND COALESCE(parent.value->>'fallback', '') <> '' THEN '[]'::jsonb
		WHEN jsonb_typeof(parent.value->'children') = 'array' THEN parent.value->'children'
		WHEN jsonb_typeof(parent.value->'content') = 'array' THEN parent.value->'content'
		ELSE '[]'::jsonb
	END) WITH ORDINALITY AS child(value, position)
)
SELECT COALESCE(string_agg(CASE
	WHEN jsonb_typeof(value) = 'string' THEN value #>> '{}'
	WHEN jsonb_typeof(value->'text') = 'string' THEN value->>'text'
	WHEN value->>'type' = 'tg_emoji' AND COALESCE(value->>'fallback', '') <> '' THEN value->>'fallback'
	WHEN COALESCE(value->>'type', '') NOT IN ('', 'lic', 'ul', 'ol')
		AND (jsonb_typeof(value->'children') = 'array' OR jsonb_typeof(value->'content') = 'array') THEN E'\n'
	ELSE NULL
END, '' ORDER BY CASE
	WHEN jsonb_typeof(value) = 'string' OR jsonb_typeof(value->'text') = 'string'
		OR (value->>'type' = 'tg_emoji' AND COALESCE(value->>'fallback', '') <> '') THEN node_path
	ELSE node_path || 2147483647
END), '') FROM description_nodes)`

func publicDescription(raw json.RawMessage) string {
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		return string(raw)
	}
	var text strings.Builder
	appendDescription(&text, document)
	return strings.TrimRight(text.String(), "\n ")
}

func appendDescription(text *strings.Builder, value any) {
	switch node := value.(type) {
	case string:
		text.WriteString(node)
	case []any:
		for _, child := range node {
			appendDescription(text, child)
		}
	case map[string]any:
		if content, ok := node["text"].(string); ok {
			text.WriteString(content)
			return
		}
		kind, _ := node["type"].(string)
		if fallback, ok := node["fallback"].(string); kind == "tg_emoji" && ok && fallback != "" {
			text.WriteString(fallback)
			return
		}
		children, ok := node["children"].([]any)
		if !ok {
			children, _ = node["content"].([]any)
		}
		if len(children) > 0 {
			appendDescription(text, children)
			if kind != "" && kind != "lic" && kind != "ul" && kind != "ol" {
				text.WriteByte('\n')
			}
		}
	}
}
