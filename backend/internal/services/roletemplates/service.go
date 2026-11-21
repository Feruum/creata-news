package roletemplates

import (
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/uptrace/bun"
)

var _ eventsv1connect.RoleTemplatesServiceHandler = (*Service)(nil)

type Service struct {
	eventsv1connect.UnimplementedRoleTemplatesServiceHandler
	db    *bun.DB
	authz *authz.Enforcer
}

func NewService(db *bun.DB, authzEnforcer *authz.Enforcer) *Service {
	return &Service{db: db, authz: authzEnforcer}
}
