package perms

import (
	"context"
	"fmt"
	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"io"
)

// ReadRelationships returns the current direct relationships for one resource.
func (c *Client) ReadRelationships(ctx context.Context, resourceType, resourceID string) ([]Relationship, error) {
	stream, err := c.client.ReadRelationships(ctx, &pb.ReadRelationshipsRequest{
		RelationshipFilter: &pb.RelationshipFilter{ResourceType: resourceType, OptionalResourceId: resourceID},
		Consistency:        &pb.Consistency{Requirement: &pb.Consistency_FullyConsistent{FullyConsistent: true}},
	})
	if err != nil {
		return nil, err
	}
	relationships := []Relationship{}
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			return relationships, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read resource relationships: %w", err)
		}
		relationship := response.Relationship
		if relationship.Subject.OptionalRelation != "" {
			return nil, fmt.Errorf("unsupported subject userset in resource relationship")
		}
		relationships = append(relationships, Relationship{Resource: relationship.Resource.ObjectType, ResourceID: relationship.Resource.ObjectId, Relation: relationship.Relation, SubjectType: relationship.Subject.Object.ObjectType, SubjectID: relationship.Subject.Object.ObjectId})
	}
}
