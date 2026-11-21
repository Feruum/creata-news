package orgmembers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
)

func (s *Service) GetOrganizationMember(ctx context.Context, req *connect.Request[eventsv1.GetOrganizationMemberRequest]) (*connect.Response[eventsv1.GetOrganizationMemberResponse], error) {
	slog.Debug("GetOrganizationMember", "id", req.Msg.Id)

	var member organizationMemberRow
	err := s.db.NewSelect().
		Model(&member).
		ColumnExpr("om.*").
		ColumnExpr("u.username AS user_username, u.email AS user_email, u.first_name AS user_first_name, u.last_name AS user_last_name").
		ColumnExpr("rt.display_name AS role_template_name, rt.key AS role_template_key").
		Join("JOIN users AS u ON u.id = om.user_id").
		Join("JOIN role_templates AS rt ON rt.id = om.role_template_id").
		Where("om.id = ?", req.Msg.Id).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&eventsv1.GetOrganizationMemberResponse{
		Member: dbOrganizationMemberToProto(member),
	}), nil
}
