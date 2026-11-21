package eventattendance

import (
	"time"

	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

func dbEventAttendanceToProto(a models.EventAttendance) *eventsv1.EventAttendance {
	att := &eventsv1.EventAttendance{
		Id:             a.ID,
		RegistrationId: a.RegistrationID,
		Status:         dbAttendanceStatusToProto(a.Status),
		CreatedAt:      a.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      a.UpdatedAt.Format(time.RFC3339),
	}
	if a.CheckedInAt != nil {
		checkedInAt := a.CheckedInAt.Format(time.RFC3339)
		att.CheckedInAt = &checkedInAt
	}
	if a.CheckedInBy != nil {
		att.CheckedInBy = a.CheckedInBy
	}
	if a.Notes != nil {
		att.Notes = a.Notes
	}
	if a.Method != nil {
		att.Method = checkInMethodToProto(*a.Method)
	}
	if a.Reason != nil {
		att.Reason = a.Reason
	}
	return att
}

func dbAttendanceStatusToProto(status models.AttendanceStatus) eventsv1.AttendanceStatus {
	switch status {
	case models.AttendanceStatusAttended:
		return eventsv1.AttendanceStatus_ATTENDANCE_STATUS_ATTENDED
	case models.AttendanceStatusNoShow:
		return eventsv1.AttendanceStatus_ATTENDANCE_STATUS_NO_SHOW
	case models.AttendanceStatusCheckedIn:
		return eventsv1.AttendanceStatus_ATTENDANCE_STATUS_CHECKED_IN
	default:
		return eventsv1.AttendanceStatus_ATTENDANCE_STATUS_UNSPECIFIED
	}
}

func protoAttendanceStatusToDB(status eventsv1.AttendanceStatus) models.AttendanceStatus {
	switch status {
	case eventsv1.AttendanceStatus_ATTENDANCE_STATUS_ATTENDED:
		return models.AttendanceStatusAttended
	case eventsv1.AttendanceStatus_ATTENDANCE_STATUS_NO_SHOW:
		return models.AttendanceStatusNoShow
	case eventsv1.AttendanceStatus_ATTENDANCE_STATUS_CHECKED_IN:
		return models.AttendanceStatusCheckedIn
	default:
		return models.AttendanceStatusAttended
	}
}

func checkInMethodToProto(m models.CheckInMethod) eventsv1.CheckInMethod {
	switch m {
	case models.CheckInMethodSelf:
		return eventsv1.CheckInMethod_CHECK_IN_METHOD_SELF
	case models.CheckInMethodManual:
		return eventsv1.CheckInMethod_CHECK_IN_METHOD_MANUAL
	case models.CheckInMethodAuto:
		return eventsv1.CheckInMethod_CHECK_IN_METHOD_AUTO
	case models.CheckInMethodLate:
		return eventsv1.CheckInMethod_CHECK_IN_METHOD_LATE
	case models.CheckInMethodStaffOverride:
		return eventsv1.CheckInMethod_CHECK_IN_METHOD_STAFF_OVERRIDE
	default:
		return eventsv1.CheckInMethod_CHECK_IN_METHOD_UNSPECIFIED
	}
}

func checkInPhaseToProto(phase models.CheckInPhase) eventsv1.CheckInPhase {
	switch phase {
	case models.CheckInPhaseNone:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_NONE
	case models.CheckInPhaseLive:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_LIVE
	case models.CheckInPhaseGrace:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_GRACE
	case models.CheckInPhaseLocked:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_LOCKED
	default:
		return eventsv1.CheckInPhase_CHECK_IN_PHASE_UNSPECIFIED
	}
}
