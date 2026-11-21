package eventattendance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/uptrace/bun"
)

// Attendance writes use the same event lock as submissions, review, and
// cancellation. The callback checks current registration state after this lock,
// so a cancelled booking cannot race a privileged check-in into attendance.
func attendanceMutation[T any](ctx context.Context, s *Service, registrationID int32, staffOnly bool, mutate func(context.Context, bun.IDB, int32) (*connect.Response[T], error)) (*connect.Response[T], error) {
	if _, err := s.authz.GetLocalUserID(ctx); err != nil {
		return nil, err
	}
	var eventID int32
	if err := s.db.NewRaw(`SELECT event_id FROM event_registrations WHERE id=?`, registrationID).Scan(ctx, &eventID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("registration not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var actor int32
	var err error
	if staffOnly {
		actor, err = s.authz.RequirePlatformStaff(ctx)
	} else {
		actor, err = s.authz.RequireResourcePermission(ctx, "event", fmt.Sprint(eventID), "check_in")
	}
	if err != nil {
		return nil, err
	}
	var response *connect.Response[T]
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewRaw(`SELECT id FROM events WHERE id=? FOR UPDATE`, eventID).Scan(ctx, &eventID); err != nil {
			return err
		}
		var err error
		response, err = mutate(ctx, tx, actor)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("registration not found"))
	}
	if err != nil {
		var ce *connect.Error
		if errors.As(err, &ce) {
			return nil, err
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return response, nil
}
