package eventattendance

import (
	"github.com/studyverse/ems-backend/gen/events/v1/eventsv1connect"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/uptrace/bun"
)

var _ eventsv1connect.EventAttendanceServiceHandler = (*Service)(nil)

type Service struct {
	eventsv1connect.UnimplementedEventAttendanceServiceHandler
	db        *bun.DB
	authz     *authz.Enforcer
	mapAppURL string
	forms     *registrationforms.Service
}

func (s *Service) WithForms(forms *registrationforms.Service) *Service { s.forms = forms; return s }

func NewService(db *bun.DB, authzEnforcer *authz.Enforcer, mapAppURL string) *Service {
	return &Service{db: db, authz: authzEnforcer, mapAppURL: mapAppURL}
}
