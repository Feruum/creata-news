package perms

import (
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
)

// SetPlatformRole replaces both platform grants in one atomic graph mutation.
// An empty role revokes platform administration and staff access.
func (c *Client) SetPlatformRole(ctx context.Context, userID, role string) error {
	if role != "" && role != platformRoleAdministrator && role != platformRoleStaff {
		return fmt.Errorf("unsupported platform role: %s", role)
	}
	updates := make([]*pb.RelationshipUpdate, 0, 2)
	for _, relation := range []string{platformRoleAdministrator, platformRoleStaff} {
		operation := pb.RelationshipUpdate_OPERATION_DELETE
		if role == relation {
			operation = pb.RelationshipUpdate_OPERATION_TOUCH
		}
		updates = append(updates, &pb.RelationshipUpdate{Operation: operation, Relationship: &pb.Relationship{Resource: &pb.ObjectReference{ObjectType: "platform", ObjectId: c.platformID}, Relation: relation, Subject: &pb.SubjectReference{Object: &pb.ObjectReference{ObjectType: "user", ObjectId: userID}}}})
	}
	_, err := c.client.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{Updates: updates})
	return err
}
