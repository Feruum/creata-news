package hermes

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/authz"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/studyverse/ems-backend/internal/roomid"
	"github.com/studyverse/ems-backend/internal/services/events"
	"github.com/studyverse/ems-backend/internal/services/rooms"
	"github.com/uptrace/bun"
)

type EventAuthor struct {
	db        *bun.DB
	authz     *authz.Enforcer
	events    *events.Service
	rooms     *rooms.Service
	timetable roomTimetable
	paidForm  func(context.Context, bun.Tx, int32) error
}

func NewEventAuthor(db *bun.DB, permissions *authz.Enforcer, eventService *events.Service, roomService *rooms.Service, mapURL string) *EventAuthor {
	return &EventAuthor{db: db, authz: permissions, events: eventService, rooms: roomService, timetable: newRoomTimetable(mapURL)}
}

func (s *EventAuthor) WithPaidForm(create func(context.Context, bun.Tx, int32) error) *EventAuthor {
	s.paidForm = create
	return s
}

func (s *EventAuthor) Run(ctx context.Context, action string, req EventRequest) (result EventResult, err error) {
	if err := req.validate(action); err != nil {
		return result, err
	}
	var orgID int32
	err = s.db.NewSelect().Table("organization_slugs").Column("organization_id").Where("slug = ?", "sunday-open").Scan(ctx, &orgID)
	if errors.Is(err, sql.ErrNoRows) {
		return result, eventFailure(connect.CodeFailedPrecondition, "Организация Sunday Open не настроена в Studyverse.")
	}
	if err != nil {
		return result, err
	}
	// Require the actor's live application permissions on the server-selected host, also for a replay.
	actorID, err := s.authz.RequireOrgPermission(ctx, orgID, "create_event")
	if err != nil {
		return result, err
	}
	if action == "rooms" {
		if req.EventID > 0 {
			event, err := ownedDraft(ctx, s.db, req.EventID, actorID, orgID, false)
			if err != nil {
				return result, err
			}
			req.StartTime, req.EndTime, req.Capacity = event.StartTime.UTC().Format(time.RFC3339Nano), event.EndTime.UTC().Format(time.RFC3339Nano), event.Capacity
		}
		slots, err := s.roomOptions(ctx, s.db, req, false)
		return EventResult{Rooms: &slots, StartTime: req.StartTime, EndTime: req.EndTime, Capacity: req.Capacity}, err
	}
	mutationID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("studyverse/hermes/events/v1/"+action+"/"+req.RequestID))
	payload, err := json.Marshal(req)
	if err != nil {
		return result, err
	}
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	err = s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Publish no partial receipt. A concurrent retry waits for the first transaction's commit or rollback.
		if _, err := tx.NewRaw("SELECT pg_advisory_xact_lock(?::bigint)", int64(binary.BigEndian.Uint64(mutationID[:8]))).Exec(ctx); err != nil {
			return err
		}
		var previous struct {
			EventID     *int32
			PayloadHash string
			Response    json.RawMessage
		}
		err := tx.NewRaw("SELECT event_id, payload_hash, response FROM hermes_event_mutations WHERE mutation_id=? AND owner_id=?", mutationID, actorID).Scan(ctx, &previous)
		if err == nil {
			if previous.PayloadHash != hash {
				return eventFailure(connect.CodeAlreadyExists, "Запрос с таким идентификатором уже выполнен с другими данными.")
			}
			if previous.EventID == nil {
				return eventFailure(connect.CodeNotFound, "Эта карточка события удалена.")
			}
			return json.Unmarshal(previous.Response, &result)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		scoped := s.events.InTransaction(tx)
		switch action {
		case "draft":
			result, err = s.draft(ctx, tx, scoped, req, orgID, mutationID, digest)
		case "book", "publish":
			if action == "book" {
				if err := rooms.LockEventBooking(ctx, tx, req.EventID); err != nil {
					return err
				}
			}
			event, loadErr := ownedDraft(ctx, tx, req.EventID, actorID, orgID, true)
			if loadErr != nil {
				return loadErr
			}
			if action == "book" {
				result, err = s.book(ctx, tx, scoped, req, event)
			} else {
				result, err = publishDraft(ctx, scoped, event.ID)
			}
		}
		if err != nil {
			return err
		}
		response, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = tx.NewRaw(`INSERT INTO hermes_event_mutations (mutation_id, action, owner_id, event_id, payload_hash, response)
			VALUES (?, ?, ?, ?, ?, ?::jsonb)`, mutationID, action, actorID, result.EventID, hash, string(response)).Exec(ctx)
		return err
	})
	return result, err
}

func publishDraft(ctx context.Context, scoped *events.Service, eventID int32) (EventResult, error) {
	_, err := scoped.PublishEvent(ctx, connect.NewRequest(&eventsv1.PublishEventRequest{EventId: eventID}))
	if err == nil {
		return EventResult{EventID: eventID, Status: "published"}, nil
	}
	var rpcErr *connect.Error
	if !errors.As(err, &rpcErr) || rpcErr.Code() != connect.CodeFailedPrecondition {
		return EventResult{}, err
	}
	reason := "Для публикации нужна одобренная бронь аудитории на всё время события."
	switch rpcErr.Message() {
	case "cannot publish: room booking request is still pending approval":
		reason = "Заявка на аудиторию ожидает одобрения. Событие пока остаётся черновиком."
	case "cannot publish: room booking request was rejected":
		reason = "Заявка на аудиторию отклонена. Выберите другую аудиторию."
	case "cannot publish: room booking request was cancelled":
		reason = "Заявка на аудиторию отменена. Выберите аудиторию заново."
	}
	return EventResult{}, eventFailure(connect.CodeFailedPrecondition, reason)
}

func ownedDraft(ctx context.Context, db bun.IDB, id, actorID, orgID int32, lock bool) (models.Event, error) {
	var event models.Event
	q := db.NewSelect().Model(&event).
		Where("e.id = ? AND e.user_id = ? AND e.organization_id = ?", id, actorID, orgID).
		Where("EXISTS (SELECT 1 FROM hermes_event_mutations m WHERE m.event_id=e.id AND m.owner_id=e.user_id AND m.action='draft')")
	if lock {
		q = q.For("UPDATE OF e")
	}
	err := q.Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return event, eventFailure(connect.CodeNotFound, "Черновик Sunday Open, созданный вами через Гермес, не найден.")
	}
	if err == nil && (event.Status != models.EventStatusDraft || !event.EndTime.After(time.Now())) {
		return event, eventFailure(connect.CodeFailedPrecondition, "Событие уже закончилось или больше не является черновиком.")
	}
	return event, err
}

func (s *EventAuthor) draft(ctx context.Context, tx bun.Tx, scoped *events.Service, req EventRequest, orgID int32, mutationID uuid.UUID, digest [32]byte) (EventResult, error) {
	cover := "https://sundayopen.social/og/events-fallback-v1.png"
	if len(req.Cover) > 0 {
		var err error
		cover, err = scoped.UploadDraftCover(ctx, req.Cover, http.DetectContentType(req.Cover), orgID, uuid.NewSHA1(mutationID, digest[:]))
		if err != nil {
			return EventResult{}, err
		}
	}
	location := &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM}
	switch req.Format {
	case "online":
		location.Type = eventsv1.LocationType_LOCATION_TYPE_ONLINE
	case "hybrid":
		location.Type = eventsv1.LocationType_LOCATION_TYPE_HYBRID
	}
	if req.OnlineURL != "" {
		location.OnlineUrl = &req.OnlineURL
	}
	// JSON strings are a supported description format. User text is never interpreted as editor markup.
	description, _ := json.Marshal(req.Description)
	status := eventsv1.EventStatus_EVENT_STATUS_DRAFT
	created, err := scoped.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{
		Title: req.Title, Description: string(description), ImageUrl: &cover, OrganizationId: orgID, Status: &status,
		StartTime: req.StartTime, EndTime: req.EndTime, Capacity: req.Capacity, StructuredLocation: location,
	}))
	if err != nil {
		return EventResult{}, err
	}
	if req.Paid {
		if s.paidForm == nil {
			return EventResult{}, eventFailure(connect.CodeUnavailable, "Форма для квитанции об оплате сейчас недоступна.")
		}
		if err := s.paidForm(ctx, tx, created.Msg.Event.Id); err != nil {
			return EventResult{}, err
		}
	}
	return EventResult{EventID: created.Msg.Event.Id, Status: "draft"}, nil
}

func (s *EventAuthor) book(ctx context.Context, tx bun.Tx, scoped *events.Service, req EventRequest, event models.Event) (EventResult, error) {
	start, _ := time.Parse(time.RFC3339, req.StartTime)
	end, _ := time.Parse(time.RFC3339, req.EndTime)
	if !event.StartTime.Equal(start) || !event.EndTime.Equal(end) || (req.Capacity == nil) != (event.Capacity == nil) ||
		(req.Capacity != nil && *req.Capacity != *event.Capacity) {
		return EventResult{}, eventFailure(connect.CodeFailedPrecondition, "Время или число мест изменилось. Запросите аудитории заново и подтвердите выбор.")
	}
	var location struct {
		Type      string  `json:"type"`
		RoomID    string  `json:"room_id"`
		OnlineURL *string `json:"online_url"`
	}
	if err := json.Unmarshal(event.StructuredLocation, &location); err != nil {
		return EventResult{}, err
	}
	if location.Type != "campus_room" && location.Type != "hybrid" {
		return EventResult{}, eventFailure(connect.CodeFailedPrecondition, "Для этого формата события аудитория не предусмотрена.")
	}
	req.RoomID = roomid.Normalize(req.RoomID)
	// A fresh chat message can repeat a selection with a new delivery ID. Do not stack requests.
	var active []models.BookingRequest
	if err := tx.NewSelect().Model(&active).Column("room_id", "start_time", "end_time", "status").
		Where("event_id = ? AND status IN ('pending', 'approved')", event.ID).For("SHARE").Scan(ctx); err != nil {
		return EventResult{}, err
	}
	if len(active) > 0 {
		if len(active) == 1 && active[0].RoomID == req.RoomID && location.RoomID == req.RoomID &&
			active[0].StartTime.Equal(start) && active[0].EndTime.Equal(end) {
			return EventResult{EventID: event.ID, RoomID: req.RoomID, BookingStatus: string(active[0].Status)}, nil
		}
		return EventResult{}, eventFailure(connect.CodeFailedPrecondition, "У этого черновика уже есть действующая заявка на аудиторию.")
	}
	options, err := s.roomOptions(ctx, tx, req, true)
	if err != nil {
		return EventResult{}, err
	}
	if len(options) != 1 || options[0].Status == "busy" {
		return EventResult{}, eventFailure(connect.CodeFailedPrecondition, "Аудитория занята или не подходит по вместимости. Выберите другую.")
	}
	if options[0].Status != "available" && !req.OwnerVerifiedAvailability {
		return EventResult{}, eventFailure(connect.CodeFailedPrecondition, "Перед подачей заявки подтвердите, что проверили аудиторию, вместимость и точное время.")
	}
	room := options[0]
	selected := &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_CAMPUS_ROOM, RoomId: &room.ID, RoomName: &room.Name}
	if location.Type == "hybrid" {
		selected.Type = eventsv1.LocationType_LOCATION_TYPE_HYBRID
		selected.OnlineUrl = location.OnlineURL
	}
	_, err = scoped.UpdateEvent(ctx, connect.NewRequest(&eventsv1.UpdateEventRequest{Id: event.ID, StructuredLocation: selected}))
	if err != nil {
		return EventResult{}, err
	}
	booking, err := s.rooms.RequestReviewedBooking(ctx, tx, connect.NewRequest(&roomsv1.CreateBookingRequestRequest{
		RoomId: room.ID, EventId: &event.ID, OrganizationId: &event.OrganizationID, Title: event.Title,
		StartTime: event.StartTime.Format(time.RFC3339Nano), EndTime: event.EndTime.Format(time.RFC3339Nano),
	}))
	if err != nil {
		return EventResult{}, err
	}
	if booking.Msg.AutoApproved || booking.Msg.BookingRequest.Status != roomsv1.BookingRequestStatus_BOOKING_REQUEST_STATUS_PENDING {
		return EventResult{}, errors.New("reviewed booking returned an unexpected state")
	}
	return EventResult{EventID: event.ID, RoomID: room.ID, BookingStatus: "pending"}, nil
}
