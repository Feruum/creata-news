package perms

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	pb "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/authzed-go/v1"
	"github.com/authzed/grpcutil"
	"github.com/samber/lo"
	"github.com/studyverse/ems-backend/internal/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	client     *authzed.Client
	platformID string
	tracer     *telemetry.SpiceDBTracer
}

const (
	platformRoleAdministrator = "administrator"
	platformRoleStaff         = "staff"
)

// Relationship represents a SpiceDB relationship to write
type Relationship struct {
	Resource    string // e.g., "organization"
	ResourceID  string // e.g., "123"
	Relation    string // e.g., "president"
	SubjectType string // e.g., "user"
	SubjectID   string // e.g., "456"
}

// NewClient creates a new SpiceDB permissions client
func NewClient(endpoint, presharedKey string, insecureMode bool, skipVerifyCA bool, platformID string) (*Client, error) {
	var opts []grpc.DialOption

	if insecureMode {
		opts = append(opts,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpcutil.WithInsecureBearerToken(presharedKey),
		)
	} else {
		// Use TLS with system certs
		var verification = grpcutil.VerifyCA
		if skipVerifyCA {
			verification = grpcutil.SkipVerifyCA
		}
		systemCerts, err := grpcutil.WithSystemCerts(verification)
		if err != nil {
			return nil, err
		}
		opts = append(opts,
			systemCerts,
			grpcutil.WithBearerToken(presharedKey),
		)
	}

	client, err := authzed.NewClient(endpoint, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{client: client, platformID: platformID, tracer: telemetry.NewSpiceDBTracer()}, nil
}

func (c *Client) CheckPermission(ctx context.Context, userID, resourceType, resourceID, permission string) (bool, error) {
	ctx, span := c.tracer.StartCheckPermissionSpan(ctx, userID, resourceType, resourceID, permission)
	defer func() { c.tracer.EndSpan(span, nil) }()

	subject := &pb.SubjectReference{
		Object: &pb.ObjectReference{
			ObjectType: "user",
			ObjectId:   userID,
		},
	}

	resource := &pb.ObjectReference{
		ObjectType: resourceType,
		ObjectId:   resourceID,
	}

	resp, err := c.client.CheckPermission(ctx, &pb.CheckPermissionRequest{
		Resource:   resource,
		Permission: permission,
		Subject:    subject,
		Consistency: &pb.Consistency{
			Requirement: &pb.Consistency_FullyConsistent{FullyConsistent: true},
		},
	})
	if err != nil {
		c.tracer.EndSpan(span, err)
		slog.Error("SpiceDB CheckPermission failed",
			"user_id", userID,
			"resource_type", resourceType,
			"resource_id", resourceID,
			"permission", permission,
			"error", err,
		)
		return false, err
	}

	allowed := resp.Permissionship == pb.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
	c.tracer.EndSpan(span, nil, allowed)

	slog.Info("Permission check result",
		"user_id", userID,
		"resource", resourceType+":"+resourceID,
		"permission", permission,
		"allowed", allowed,
		"permissionship", resp.Permissionship.String(),
	)

	return allowed, nil
}

func (c *Client) WriteRelationships(ctx context.Context, relationships []Relationship) error {
	ctx, span := c.tracer.StartWriteRelationshipsSpan(ctx, len(relationships))
	defer span.End()

	updates := lo.Map(relationships, func(rel Relationship, _ int) *pb.RelationshipUpdate {
		return &pb.RelationshipUpdate{
			Operation: pb.RelationshipUpdate_OPERATION_TOUCH,
			Relationship: &pb.Relationship{
				Resource: &pb.ObjectReference{
					ObjectType: rel.Resource,
					ObjectId:   rel.ResourceID,
				},
				Relation: rel.Relation,
				Subject: &pb.SubjectReference{
					Object: &pb.ObjectReference{
						ObjectType: rel.SubjectType,
						ObjectId:   rel.SubjectID,
					},
				},
			},
		}
	})

	_, err := c.client.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{
		Updates: updates,
	})
	if err != nil {
		c.tracer.EndSpan(span, err)
		slog.Error("SpiceDB WriteRelationships failed", "error", err)
		return err
	}

	slog.Debug("Wrote relationships", "count", len(relationships))
	return nil
}

// CreateOwnerRelationship atomically claims an unowned resource. The
// precondition prevents a second principal from being added as another owner
// during concurrent registration or a guessed-resource attack.
func (c *Client) CreateOwnerRelationship(ctx context.Context, resourceType, resourceID, userID string) error {
	_, err := c.client.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{
		Updates: []*pb.RelationshipUpdate{
			{
				Operation: pb.RelationshipUpdate_OPERATION_TOUCH,
				Relationship: &pb.Relationship{
					Resource: &pb.ObjectReference{ObjectType: resourceType, ObjectId: resourceID},
					Relation: "owner",
					Subject: &pb.SubjectReference{
						Object: &pb.ObjectReference{ObjectType: "user", ObjectId: userID},
					},
				},
			},
		},
		OptionalPreconditions: []*pb.Precondition{
			{
				Operation: pb.Precondition_OPERATION_MUST_NOT_MATCH,
				Filter: &pb.RelationshipFilter{
					ResourceType:       resourceType,
					OptionalResourceId: resourceID,
					OptionalRelation:   "owner",
				},
			},
		},
	})
	if err != nil {
		slog.Warn("Failed to claim resource ownership",
			"resource", resourceType+":"+resourceID,
			"user_id", userID,
			"error", err,
		)
		return err
	}
	return nil
}

// DeleteRelationships removes relationships from SpiceDB
func (c *Client) DeleteRelationships(ctx context.Context, relationships []Relationship) error {
	updates := lo.Map(relationships, func(rel Relationship, _ int) *pb.RelationshipUpdate {
		return &pb.RelationshipUpdate{
			Operation: pb.RelationshipUpdate_OPERATION_DELETE,
			Relationship: &pb.Relationship{
				Resource: &pb.ObjectReference{
					ObjectType: rel.Resource,
					ObjectId:   rel.ResourceID,
				},
				Relation: rel.Relation,
				Subject: &pb.SubjectReference{
					Object: &pb.ObjectReference{
						ObjectType: rel.SubjectType,
						ObjectId:   rel.SubjectID,
					},
				},
			},
		}
	})

	_, err := c.client.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{
		Updates: updates,
	})
	if err != nil {
		slog.Error("SpiceDB DeleteRelationships failed", "error", err)
		return err
	}

	return nil
}

func (c *Client) LookupResources(ctx context.Context, userID, resourceType, permission string) ([]string, error) {
	ctx, span := c.tracer.StartLookupResourcesSpan(ctx, userID, resourceType, permission)
	defer span.End()

	subject := &pb.SubjectReference{
		Object: &pb.ObjectReference{
			ObjectType: "user",
			ObjectId:   userID,
		},
	}

	stream, err := c.client.LookupResources(ctx, &pb.LookupResourcesRequest{
		ResourceObjectType: resourceType,
		Permission:         permission,
		Subject:            subject,
	})
	if err != nil {
		return nil, err
	}

	var resourceIDs []string
	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read resource lookup: %w", err)
		}
		resourceIDs = append(resourceIDs, resp.ResourceObjectId)
	}

	return resourceIDs, nil
}

// LookupSubjects finds all users that have a permission on a resource
// Example: LookupSubjects(ctx, "organization", "123", "create_event") returns all user IDs
func (c *Client) LookupSubjects(ctx context.Context, resourceType, resourceID, permission string) ([]string, error) {
	resource := &pb.ObjectReference{
		ObjectType: resourceType,
		ObjectId:   resourceID,
	}

	stream, err := c.client.LookupSubjects(ctx, &pb.LookupSubjectsRequest{
		Resource:          resource,
		Permission:        permission,
		SubjectObjectType: "user",
	})
	if err != nil {
		return nil, err
	}

	var subjectIDs []string
	for {
		resp, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("read subject lookup: %w", err)
		}
		subjectIDs = append(subjectIDs, resp.Subject.SubjectObjectId)
	}

	return subjectIDs, nil
}

// GetPlatformID returns the configured platform identifier
func (c *Client) GetPlatformID() string {
	return c.platformID
}

// SetupPlatformRelationship sets up a user's relationship to the platform
func (c *Client) SetupPlatformRelationship(ctx context.Context, userID string, role string) error {
	return c.WriteRelationships(ctx, buildPlatformRoleRelationships(c.platformID, userID, role))
}

func (c *Client) DeletePlatformRelationships(ctx context.Context, userID string) error {
	return c.DeleteRelationships(ctx, buildPlatformRoleDeletionRelationships(c.platformID, userID))
}

// SetupOrganizationRelationship sets up a user's relationship to an organization
func (c *Client) SetupOrganizationRelationship(ctx context.Context, organizationID, userID, role string) error {
	return c.WriteRelationships(ctx, []Relationship{
		{
			Resource:    "organization",
			ResourceID:  organizationID,
			Relation:    role, // "president" or "member"
			SubjectType: "user",
			SubjectID:   userID,
		},
	})
}

// LinkOrganizationToPlatform ensures an organization is linked to the platform for permission inheritance
func (c *Client) LinkOrganizationToPlatform(ctx context.Context, organizationID string) error {
	return c.WriteRelationships(ctx, []Relationship{
		{
			Resource:    "organization",
			ResourceID:  organizationID,
			Relation:    "parent_platform",
			SubjectType: "platform",
			SubjectID:   c.platformID,
		},
	})
}

// SetupEventRelationships sets up an event's relationships using the capability-based model
func (c *Client) SetupEventRelationships(ctx context.Context, eventID, orgID, creatorID string) error {
	return c.WriteRelationships(ctx, []Relationship{
		{
			Resource:    "event",
			ResourceID:  eventID,
			Relation:    "host_organization",
			SubjectType: "organization",
			SubjectID:   orgID,
		},
		{
			Resource:    "event",
			ResourceID:  eventID,
			Relation:    "creator",
			SubjectType: "user",
			SubjectID:   creatorID,
		},
	})
}

// IsGlobalStaff checks if user is platform staff or admin
func (c *Client) IsGlobalStaff(ctx context.Context, userID string) (bool, error) {
	return c.CheckPermission(ctx, userID, "platform", c.platformID, "manage_organizations")
}

func buildPlatformRoleRelationships(platformID, userID, role string) []Relationship {
	return []Relationship{{
		Resource:    "platform",
		ResourceID:  platformID,
		Relation:    role,
		SubjectType: "user",
		SubjectID:   userID,
	}}
}

func buildPlatformRoleDeletionRelationships(platformID, userID string) []Relationship {
	return []Relationship{
		{
			Resource:    "platform",
			ResourceID:  platformID,
			Relation:    platformRoleAdministrator,
			SubjectType: "user",
			SubjectID:   userID,
		},
		{
			Resource:    "platform",
			ResourceID:  platformID,
			Relation:    platformRoleStaff,
			SubjectType: "user",
			SubjectID:   userID,
		},
	}
}

// GetManagedOrganizations returns all organization IDs where the user can manage settings
func (c *Client) GetManagedOrganizations(ctx context.Context, userID string) ([]string, error) {
	return c.LookupResources(ctx, userID, "organization", "manage_settings")
}

// GetUserPermissions returns a summary of user permissions for the frontend
func (c *Client) GetUserPermissions(ctx context.Context, userID string) (*UserPermissions, error) {
	isAdmin, _ := c.CheckPermission(ctx, userID, "platform", c.platformID, "manage_system")
	isStaff, _ := c.CheckPermission(ctx, userID, "platform", c.platformID, "manage_organizations")
	managedOrgs, _ := c.GetManagedOrganizations(ctx, userID)

	return &UserPermissions{
		UserID:               userID,
		IsAdmin:              isAdmin,
		IsGlobalStaff:        isStaff,
		ManagedOrganizations: managedOrgs,
	}, nil
}

// UserPermissions represents computed permissions for a user
type UserPermissions struct {
	UserID               string   `json:"user_id"`
	IsAdmin              bool     `json:"is_admin"`
	IsGlobalStaff        bool     `json:"is_global_staff"`
	ManagedOrganizations []string `json:"managed_organizations"`
}

// Capability constants matching the database and proto enum
const (
	CapabilityCreateEvent         = "can_create_event"
	CapabilityManageMembers       = "can_manage_members"
	CapabilityManageSettings      = "can_manage_settings"
	CapabilityViewAnalytics       = "can_view_analytics"
	CapabilityBookRooms           = "can_book_rooms"
	CapabilityApproveBookings     = "can_approve_bookings"
	CapabilityManageRegistrations = "can_manage_registrations"
	CapabilityCheckInAttendees    = "can_check_in_attendees"
)

// AllCapabilities is a list of all available capabilities for validation
var AllCapabilities = []string{
	CapabilityCreateEvent,
	CapabilityManageMembers,
	CapabilityManageSettings,
	CapabilityViewAnalytics,
	CapabilityBookRooms,
	CapabilityApproveBookings,
	CapabilityManageRegistrations,
	CapabilityCheckInAttendees,
}

// SetOrganizationMember adds a user as a member of an organization with specific capabilities
// This replaces the old role-based approach with capability-based tuples
func (c *Client) SetOrganizationMember(ctx context.Context, orgID, userID string, capabilities []string) error {
	desired := map[string]bool{"member": true}
	for _, capability := range capabilities {
		if !lo.Contains(AllCapabilities, capability) {
			return fmt.Errorf("unknown organization capability: %s", capability)
		}
		desired[capability] = true
	}
	relations := append([]string{"member"}, AllCapabilities...)
	updates := make([]*pb.RelationshipUpdate, 0, len(relations))
	for _, relation := range relations {
		operation := pb.RelationshipUpdate_OPERATION_DELETE
		if desired[relation] {
			operation = pb.RelationshipUpdate_OPERATION_TOUCH
		}
		updates = append(updates, &pb.RelationshipUpdate{Operation: operation, Relationship: &pb.Relationship{
			Resource: &pb.ObjectReference{ObjectType: "organization", ObjectId: orgID}, Relation: relation,
			Subject: &pb.SubjectReference{Object: &pb.ObjectReference{ObjectType: "user", ObjectId: userID}},
		}})
	}
	_, err := c.client.WriteRelationships(ctx, &pb.WriteRelationshipsRequest{Updates: updates})
	return err
}

// RemoveOrganizationMember removes a user from an organization (all relations)
func (c *Client) RemoveOrganizationMember(ctx context.Context, orgID, userID string) error {
	// Delete member relation and all capability relations
	relationships := []Relationship{
		{
			Resource:    "organization",
			ResourceID:  orgID,
			Relation:    "member",
			SubjectType: "user",
			SubjectID:   userID,
		},
	}

	// Remove all possible capabilities
	relationships = append(relationships, lo.Map(AllCapabilities, func(cap string, _ int) Relationship {
		return Relationship{
			Resource:    "organization",
			ResourceID:  orgID,
			Relation:    cap,
			SubjectType: "user",
			SubjectID:   userID,
		}
	})...)

	slog.Info("Removing organization member",
		"org_id", orgID,
		"user_id", userID,
	)

	return c.DeleteRelationships(ctx, relationships)
}

// UpdateOrganizationMemberCapabilities updates a user's capabilities in an organization
// This removes old capabilities and sets new ones atomically
func (c *Client) UpdateOrganizationMemberCapabilities(ctx context.Context, orgID, userID string, newCapabilities []string) error {
	return c.SetOrganizationMember(ctx, orgID, userID, newCapabilities)
}

// GetUserOrganizationCapabilities checks which capabilities a user has for a specific organization
func (c *Client) GetUserOrganizationCapabilities(ctx context.Context, orgID, userID string) ([]string, error) {
	var caps []string

	for _, cap := range AllCapabilities {
		permName := cap[4:]
		has, err := c.CheckPermission(ctx, userID, "organization", orgID, permName)
		if err != nil {
			slog.Warn("Failed to check capability",
				"org_id", orgID,
				"user_id", userID,
				"capability", cap,
				"error", err,
			)
			continue
		}
		if has {
			caps = append(caps, cap)
		}
	}

	return caps, nil
}

func (c *Client) AddEventCollaborator(ctx context.Context, eventID, orgID string) error {
	slog.Info("Adding event collaborator", "event_id", eventID, "org_id", orgID)
	return c.WriteRelationships(ctx, []Relationship{
		{
			Resource:    "event",
			ResourceID:  eventID,
			Relation:    "collaborating_organization",
			SubjectType: "organization",
			SubjectID:   orgID,
		},
	})
}

func (c *Client) RemoveEventCollaborator(ctx context.Context, eventID, orgID string) error {
	slog.Info("Removing event collaborator", "event_id", eventID, "org_id", orgID)
	return c.DeleteRelationships(ctx, []Relationship{
		{
			Resource:    "event",
			ResourceID:  eventID,
			Relation:    "collaborating_organization",
			SubjectType: "organization",
			SubjectID:   orgID,
		},
	})
}
