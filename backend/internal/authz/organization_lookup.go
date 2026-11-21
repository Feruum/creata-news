package authz

import (
	"context"
	"strconv"
)

// FilterOrganizations bounds graph lookup results by the current SQL policy.
func (e *Enforcer) FilterOrganizations(ctx context.Context, userID int32, candidates []string, permission string) []int32 {
	result := make([]int32, 0, len(candidates))
	seen := map[int32]bool{}
	for _, candidate := range candidates {
		id, err := strconv.ParseInt(candidate, 10, 32)
		if err != nil || id <= 0 {
			continue
		}
		orgID := int32(id)
		if seen[orgID] {
			continue
		}
		seen[orgID] = true
		if e.CheckOrgPermission(ctx, userID, orgID, permission) {
			result = append(result, orgID)
		}
	}
	return result
}
