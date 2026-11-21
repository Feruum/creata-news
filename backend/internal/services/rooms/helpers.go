package rooms

import (
	"context"
	"encoding/json"
	"time"

	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/models"
)

type bookingRequestRow struct {
	models.BookingRequest
	RoomName         *string `bun:"room_name"`
	RoomBlock        *string `bun:"room_block"`
	RoomRoomNumber   *string `bun:"room_room_number"`
	RoomFloor        *int32  `bun:"room_floor"`
	RequesterName    string  `bun:"requester_name"`
	OrganizationName *string `bun:"organization_name"`
}

type conflictingRequestRow struct {
	models.BookingRequest
	RoomName *string `bun:"room_name"`
}

func protoToDBRoomType(t roomsv1.RoomType) models.RoomType {
	switch t {
	case roomsv1.RoomType_ROOM_TYPE_CLASSROOM:
		return models.RoomTypeClassroom
	case roomsv1.RoomType_ROOM_TYPE_LABORATORY:
		return models.RoomTypeLaboratory
	case roomsv1.RoomType_ROOM_TYPE_LIBRARY:
		return models.RoomTypeLibrary
	case roomsv1.RoomType_ROOM_TYPE_OFFICE:
		return models.RoomTypeOffice
	case roomsv1.RoomType_ROOM_TYPE_CAFETERIA:
		return models.RoomTypeCafeteria
	case roomsv1.RoomType_ROOM_TYPE_GYM:
		return models.RoomTypeGym
	case roomsv1.RoomType_ROOM_TYPE_AUDITORIUM:
		return models.RoomTypeAuditorium
	case roomsv1.RoomType_ROOM_TYPE_STUDY_ROOM:
		return models.RoomTypeStudyRoom
	case roomsv1.RoomType_ROOM_TYPE_COMMON_AREA:
		return models.RoomTypeCommonArea
	case roomsv1.RoomType_ROOM_TYPE_MEDICAL:
		return models.RoomTypeMedical
	case roomsv1.RoomType_ROOM_TYPE_LECTURE_HALL:
		return models.RoomTypeLectureHall
	case roomsv1.RoomType_ROOM_TYPE_COMPUTER_LAB:
		return models.RoomTypeComputerLab
	default:
		return models.RoomTypeOther
	}
}

func dbToProtoRoomType(t models.RoomType) roomsv1.RoomType {
	switch t {
	case models.RoomTypeClassroom:
		return roomsv1.RoomType_ROOM_TYPE_CLASSROOM
	case models.RoomTypeLaboratory:
		return roomsv1.RoomType_ROOM_TYPE_LABORATORY
	case models.RoomTypeLibrary:
		return roomsv1.RoomType_ROOM_TYPE_LIBRARY
	case models.RoomTypeOffice:
		return roomsv1.RoomType_ROOM_TYPE_OFFICE
	case models.RoomTypeCafeteria:
		return roomsv1.RoomType_ROOM_TYPE_CAFETERIA
	case models.RoomTypeGym:
		return roomsv1.RoomType_ROOM_TYPE_GYM
	case models.RoomTypeAuditorium:
		return roomsv1.RoomType_ROOM_TYPE_AUDITORIUM
	case models.RoomTypeStudyRoom:
		return roomsv1.RoomType_ROOM_TYPE_STUDY_ROOM
	case models.RoomTypeCommonArea:
		return roomsv1.RoomType_ROOM_TYPE_COMMON_AREA
	case models.RoomTypeMedical:
		return roomsv1.RoomType_ROOM_TYPE_MEDICAL
	case models.RoomTypeLectureHall:
		return roomsv1.RoomType_ROOM_TYPE_LECTURE_HALL
	case models.RoomTypeComputerLab:
		return roomsv1.RoomType_ROOM_TYPE_COMPUTER_LAB
	default:
		return roomsv1.RoomType_ROOM_TYPE_OTHER
	}
}

func protoToDBBookingStatus(s roomsv1.BookingRequestStatus) models.BookingRequestStatus {
	switch s {
	case roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_PENDING:
		return models.BookingRequestStatusPending
	case roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_APPROVED:
		return models.BookingRequestStatusApproved
	case roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_REJECTED:
		return models.BookingRequestStatusRejected
	case roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_CANCELLED:
		return models.BookingRequestStatusCancelled
	default:
		return models.BookingRequestStatusPending
	}
}

func dbToProtoBookingStatus(s models.BookingRequestStatus) roomsv1.BookingRequestStatus {
	switch s {
	case models.BookingRequestStatusPending:
		return roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_PENDING
	case models.BookingRequestStatusApproved:
		return roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_APPROVED
	case models.BookingRequestStatusRejected:
		return roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_REJECTED
	case models.BookingRequestStatusCancelled:
		return roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_CANCELLED
	default:
		return roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_UNSPECIFIED
	}
}

func dbToProtoPriority(p models.BookingPriority) roomsv1.BookingPriority {
	switch p {
	case models.BookingPriorityAdministrative:
		return roomsv1.BookingPriority_BOOKING_PRIORITY_ADMINISTRATIVE
	case models.BookingPriorityAcademic:
		return roomsv1.BookingPriority_BOOKING_PRIORITY_ACADEMIC
	case models.BookingPriorityStudentClub:
		return roomsv1.BookingPriority_BOOKING_PRIORITY_STUDENT_CLUB
	case models.BookingPriorityIndividual:
		return roomsv1.BookingPriority_BOOKING_PRIORITY_INDIVIDUAL
	default:
		return roomsv1.BookingPriority_BOOKING_PRIORITY_UNSPECIFIED
	}
}

func dbRoomToProto(r models.Room) *roomsv1.Room {
	room := &roomsv1.Room{
		Id:           r.ID,
		Name:         r.Name,
		RoomNumber:   r.RoomNumber,
		Block:        r.Block,
		Floor:        r.Floor,
		Type:         dbToProtoRoomType(r.Type),
		IsAccessible: r.IsAccessible,
		IsActive:     r.IsActive,
		IsBookable:   r.IsBookable,
		CreatedAt:    r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    r.UpdatedAt.Format(time.RFC3339),
	}

	if r.Capacity != nil {
		room.Capacity = r.Capacity
	}
	if r.Description != nil {
		room.Description = r.Description
	}
	if r.Features != nil {
		var features []string
		if json.Unmarshal(*r.Features, &features) == nil {
			room.Features = features
		}
	}
	if r.PrimaryImageUrl != nil {
		room.PrimaryImageUrl = r.PrimaryImageUrl
	}

	return room
}

func dbBookingRequestToProto(r models.BookingRequest) *roomsv1.BookingRequest {
	req := &roomsv1.BookingRequest{
		Id:          r.ID,
		RoomId:      r.RoomID,
		RequesterId: r.RequesterID,
		Priority:    dbToProtoPriority(r.Priority),
		Title:       r.Title,
		Status:      dbToProtoBookingStatus(r.Status),
		StartTime:   r.StartTime.Format(time.RFC3339),
		EndTime:     r.EndTime.Format(time.RFC3339),
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}

	if r.OrganizationID != nil {
		req.OrganizationId = r.OrganizationID
	}
	if r.Description != nil {
		req.Description = r.Description
	}
	if r.ReviewedBy != nil {
		req.ReviewedBy = r.ReviewedBy
	}
	if r.ReviewedAt != nil {
		reviewedAt := r.ReviewedAt.Format(time.RFC3339)
		req.ReviewedAt = &reviewedAt
	}
	if r.RejectionReason != nil {
		req.RejectionReason = r.RejectionReason
	}
	if r.EventID != nil {
		req.EventId = r.EventID
	}

	return req
}

func bookingRequestRowToProto(r bookingRequestRow) *roomsv1.BookingRequest {
	req := dbBookingRequestToProto(r.BookingRequest)

	if r.RoomName != nil {
		req.Room = &roomsv1.Room{
			Id:   r.RoomID,
			Name: *r.RoomName,
		}
		if r.RoomBlock != nil {
			req.Room.Block = *r.RoomBlock
		}
		if r.RoomRoomNumber != nil {
			req.Room.RoomNumber = *r.RoomRoomNumber
		}
		if r.RoomFloor != nil {
			req.Room.Floor = *r.RoomFloor
		}
	}
	if r.RequesterName != "" {
		req.RequesterName = &r.RequesterName
	}
	if r.OrganizationName != nil {
		req.OrganizationName = r.OrganizationName
	}

	return req
}

func dbBookingToProto(b models.RoomBooking) *roomsv1.Booking {
	booking := &roomsv1.Booking{
		Id:        b.ID,
		RoomId:    b.RoomID,
		CreatedBy: b.CreatedBy,
		Title:     b.Title,
		StartTime: b.StartTime.Format(time.RFC3339),
		EndTime:   b.EndTime.Format(time.RFC3339),
		CreatedAt: b.CreatedAt.Format(time.RFC3339),
		UpdatedAt: b.UpdatedAt.Format(time.RFC3339),
	}

	if b.BookingRequestID != nil {
		booking.BookingRequestId = b.BookingRequestID
	}
	if b.EventID != nil {
		booking.EventId = b.EventID
	}

	return booking
}

func overlaps(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && end1.After(start2)
}

func (s *Service) determineOrgPriority(ctx context.Context, orgID int32) models.BookingPriority {
	var org models.Organization
	err := s.db.NewSelect().Model(&org).Where("id = ?", orgID).Scan(ctx)
	if err != nil {
		return models.BookingPriorityIndividual
	}

	var orgType models.OrganizationType
	err = s.db.NewSelect().Model(&orgType).Where("id = ?", org.OrganizationTypeID).Scan(ctx)
	if err != nil {
		return models.BookingPriorityStudentClub
	}

	switch orgType.Title {
	case "University Administration", "Administration":
		return models.BookingPriorityAdministrative
	case "Academic Department", "Department", "Faculty":
		return models.BookingPriorityAcademic
	default:
		return models.BookingPriorityStudentClub
	}
}
