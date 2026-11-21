package orgmembers

import (
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/perms"
	"github.com/uptrace/bun"
)

var _ eventsv1connect.OrganizationMembersServiceHandler = (*Service)(nil)

type Service struct {
	eventsv1connect.UnimplementedOrganizationMembersServiceHandler
	db          *bun.DB
	perms       *perms.Client
	authz       *authz.Enforcer
	publisher   *events.Publisher
}

func NewService(db *bun.DB, permsClient *perms.Client, authzEnforcer *authz.Enforcer, publisher *events.Publisher) *Service {
	return &Service{db: db, perms: permsClient, authz: authzEnforcer, publisher: publisher}
}
