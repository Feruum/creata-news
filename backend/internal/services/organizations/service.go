package organizations

import (
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/studyverse/ems-backend/internal/services/orgmembers"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/uptrace/bun"
)

var _ eventsv1connect.OrganizationsServiceHandler = (*Service)(nil)

type Service struct {
	eventsv1connect.UnimplementedOrganizationsServiceHandler
	db         *bun.DB
	perms      *perms.Client
	authz      *authz.Enforcer
	storage    storage.Provider
	publisher  *events.Publisher
	orgMembers *orgmembers.Service
}

func NewService(db *bun.DB, permsClient *perms.Client, authzEnforcer *authz.Enforcer, storageProvider storage.Provider, publisher *events.Publisher, orgMembersService *orgmembers.Service) *Service {
	return &Service{
		db:         db,
		perms:      permsClient,
		authz:      authzEnforcer,
		storage:    storageProvider,
		publisher:  publisher,
		orgMembers: orgMembersService,
	}
}
