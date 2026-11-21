package roomid

import "strings"

func Normalize(roomID string) string {
	trimmed := strings.TrimSpace(roomID)
	if trimmed == "" {
		return ""
	}

	uppercase := strings.ToUpper(trimmed)
	if suffix, ok := strings.CutPrefix(uppercase, "С1."); ok {
		return "C1." + suffix
	}
	switch uppercase {
	case "ATRIUM", "COWORKING", "FULLGYM", "DININGHALL":
		return uppercase
	}
	if strings.HasPrefix(uppercase, "C1.") || strings.HasPrefix(uppercase, "K.") {
		return uppercase
	}
	if suffix, ok := strings.CutPrefix(uppercase, "К."); ok {
		return "K." + suffix
	}

	return trimmed
}

func NormalizeList(roomIDs []string) []string {
	if len(roomIDs) == 0 {
		return nil
	}

	result := make([]string, 0, len(roomIDs))
	seen := make(map[string]struct{}, len(roomIDs))

	for _, roomID := range roomIDs {
		normalized := Normalize(roomID)
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}

		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	if len(result) == 0 {
		return nil
	}

	return result
}
