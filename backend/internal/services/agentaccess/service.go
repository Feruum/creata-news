package agentaccess

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
	agentaccessv1 "github.com/studyverse/ems-backend/gen/agentaccess/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userResolver interface {
	ResolveByKratosID(context.Context, string) (models.User, error)
}

type permissionStore interface {
	CheckPermission(context.Context, string, string, string, string) (bool, error)
	CreateOwnerRelationship(context.Context, string, string, string) error
	WriteRelationships(context.Context, []perms.Relationship) error
	ReadRelationships(context.Context, string, string) ([]perms.Relationship, error)
}

type Service struct {
	users         userResolver
	permissions   permissionStore
	organizations organizationPermissions
}

func NewService(database *bun.DB, permissions *perms.Client) *Service {
	var store permissionStore
	if permissions != nil {
		store = permissions
	}
	return &Service{
		users:         &bunUserResolver{database: database},
		permissions:   store,
		organizations: authz.NewEnforcer(database, permissions),
	}
}

type bunUserResolver struct {
	database *bun.DB
}

func (resolver *bunUserResolver) ResolveByKratosID(ctx context.Context, kratosID string) (models.User, error) {
	var user models.User
	if err := resolver.database.NewSelect().Model(&user).Where("kratos_id = ?", kratosID).Scan(ctx); err != nil {
		return models.User{}, err
	}
	return user, nil
}

func (service *Service) Authenticate(
	ctx context.Context,
	_ *connect.Request[agentaccessv1.AuthenticateRequest],
) (*connect.Response[agentaccessv1.AuthenticateResponse], error) {
	principal, _, err := service.resolvePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&agentaccessv1.AuthenticateResponse{Principal: principal}), nil
}

func (service *Service) CheckAccess(
	ctx context.Context,
	request *connect.Request[agentaccessv1.CheckAccessRequest],
) (*connect.Response[agentaccessv1.CheckAccessResponse], error) {
	principal, userID, err := service.resolvePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	resourceType, permission, err := mapAccess(request.Msg.ResourceType, request.Msg.Permission)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	allowed, err := service.checkPermission(
		ctx,
		userID,
		resourceType,
		request.Msg.ResourceId,
		permission,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service unavailable"))
	}
	return connect.NewResponse(&agentaccessv1.CheckAccessResponse{
		Allowed:   allowed,
		Principal: principal,
	}), nil
}

func (service *Service) RegisterResource(
	ctx context.Context,
	request *connect.Request[agentaccessv1.RegisterResourceRequest],
) (*connect.Response[agentaccessv1.RegisterResourceResponse], error) {
	principal, userID, err := service.resolvePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	resourceType, err := mapResource(request.Msg.ResourceType)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	alreadyOwned, err := service.checkPermission(
		ctx,
		userID,
		resourceType,
		request.Msg.ResourceId,
		"view",
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service unavailable"))
	}
	if alreadyOwned {
		return connect.NewResponse(&agentaccessv1.RegisterResourceResponse{
			Principal: principal,
			Created:   false,
		}), nil
	}

	if err := service.permissions.CreateOwnerRelationship(ctx, resourceType, request.Msg.ResourceId, userID); err != nil {
		if status.Code(err) == codes.FailedPrecondition {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("resource is already owned"))
		}
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service unavailable"))
	}
	return connect.NewResponse(&agentaccessv1.RegisterResourceResponse{
		Principal: principal,
		Created:   true,
	}), nil
}

func (service *Service) LinkSessionDocument(
	ctx context.Context,
	request *connect.Request[agentaccessv1.LinkSessionDocumentRequest],
) (*connect.Response[agentaccessv1.LinkSessionDocumentResponse], error) {
	_, userID, err := service.resolvePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	canManage, err := service.checkPermission(
		ctx,
		userID,
		"agent_session",
		request.Msg.SessionId,
		"manage_documents",
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service unavailable"))
	}
	if !canManage {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("session owner access required"))
	}
	canViewDocument, err := service.checkPermission(
		ctx,
		userID,
		"document",
		request.Msg.DocumentId,
		"view",
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service unavailable"))
	}
	if !canViewDocument {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("document view access required"))
	}
	if err := service.permissions.WriteRelationships(ctx, []perms.Relationship{
		{
			Resource:    "agent_session",
			ResourceID:  request.Msg.SessionId,
			Relation:    "source_document",
			SubjectType: "document",
			SubjectID:   request.Msg.DocumentId,
		},
	}); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("authorization service unavailable"))
	}
	return connect.NewResponse(&agentaccessv1.LinkSessionDocumentResponse{Linked: true}), nil
}

func (service *Service) resolvePrincipal(
	ctx context.Context,
) (*agentaccessv1.Principal, string, error) {
	kratosID := auth.GetUserID(ctx)
	if kratosID == "" {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
	}
	user, err := service.users.ResolveByKratosID(ctx, kratosID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", connect.NewError(connect.CodeUnauthenticated, errors.New("EMS user not found"))
	}
	if err != nil {
		return nil, "", connect.NewError(connect.CodeInternal, errors.New("failed to resolve EMS user"))
	}
	principal := &agentaccessv1.Principal{
		UserId:           user.ID,
		KratosIdentityId: kratosID,
		Email:            user.Email,
		FirstName:        user.FirstName,
		LastName:         user.LastName,
	}
	return principal, strconv.FormatInt(int64(user.ID), 10), nil
}

func mapResource(resource agentaccessv1.AgentResourceType) (string, error) {
	switch resource {
	case agentaccessv1.AgentResourceType_AGENT_RESOURCE_TYPE_DOCUMENT:
		return "document", nil
	case agentaccessv1.AgentResourceType_AGENT_RESOURCE_TYPE_SESSION:
		return "agent_session", nil
	default:
		return "", fmt.Errorf("unsupported agent resource type %s", resource.String())
	}
}

func mapAccess(
	resource agentaccessv1.AgentResourceType,
	permission agentaccessv1.AgentPermission,
) (string, string, error) {
	resourceType, err := mapResource(resource)
	if err != nil {
		return "", "", err
	}
	allowed := map[agentaccessv1.AgentResourceType]map[agentaccessv1.AgentPermission]string{
		agentaccessv1.AgentResourceType_AGENT_RESOURCE_TYPE_DOCUMENT: {
			agentaccessv1.AgentPermission_AGENT_PERMISSION_VIEW:   "view",
			agentaccessv1.AgentPermission_AGENT_PERMISSION_EDIT:   "edit",
			agentaccessv1.AgentPermission_AGENT_PERMISSION_DELETE: "delete",
			agentaccessv1.AgentPermission_AGENT_PERMISSION_SHARE:  "share",
		},
		agentaccessv1.AgentResourceType_AGENT_RESOURCE_TYPE_SESSION: {
			agentaccessv1.AgentPermission_AGENT_PERMISSION_VIEW:         "view",
			agentaccessv1.AgentPermission_AGENT_PERMISSION_DELETE:       "delete",
			agentaccessv1.AgentPermission_AGENT_PERMISSION_SEND_MESSAGE: "send_message",
			agentaccessv1.AgentPermission_AGENT_PERMISSION_USE_DOCUMENT: "use_document",
		},
	}
	permissionName, ok := allowed[resource][permission]
	if !ok {
		return "", "", fmt.Errorf("permission %s is invalid for %s", permission.String(), resource.String())
	}
	return resourceType, permissionName, nil
}
