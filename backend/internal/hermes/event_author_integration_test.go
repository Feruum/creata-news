//go:build integration

package hermes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	roomsv1 "github.com/studyverse/ems-backend/gen/rooms/v1"
	"github.com/studyverse/ems-backend/internal/auth"
	"github.com/studyverse/ems-backend/internal/authz"
	domainevents "github.com/studyverse/ems-backend/internal/events"
	"github.com/studyverse/ems-backend/internal/registrationfiles"
	"github.com/studyverse/ems-backend/internal/registrationforms"
	"github.com/studyverse/ems-backend/internal/services/events"
	"github.com/studyverse/ems-backend/internal/services/rooms"
	"github.com/studyverse/ems-backend/internal/storage"
	"github.com/studyverse/ems-backend/internal/testsupport"
)

type eventCoverStore struct {
	storage.Provider
	keys []string
}

func (s *eventCoverStore) UploadObject(_ context.Context, key, mime string, data []byte) error {
	if mime != "image/png" {
		return errors.New("expected normalized cover")
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		return err
	}
	s.keys = append(s.keys, key)
	return nil
}
func (s *eventCoverStore) GetPublicURL(key string) string { return "https://test.invalid/" + key }

func TestOwnerEventAuthoringTransactions(t *testing.T) {
	db := testsupport.ClonedDB(t)
	db.SetMaxOpenConns(3)
	owner := uuid.NewString()
	ctx := context.WithValue(t.Context(), auth.UserIDKey, owner)
	insert := func(query string, args ...any) int32 {
		t.Helper()
		var id int32
		if err := db.NewRaw(query, args...).Scan(ctx, &id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	actor := insert("INSERT INTO users(kratos_id,username,email,password) VALUES(?,'hermes-owner','owner@test.invalid','test') RETURNING id", owner)
	kind := insert("INSERT INTO organization_types(title) VALUES('Hermes test') RETURNING id")
	org := insert("INSERT INTO organizations(title,organization_type_id) VALUES('Renamed Sunday host',?) RETURNING id", kind)
	// Isolate the historical, unique slug in this disposable clone.
	exec("DELETE FROM organization_slugs WHERE slug='sunday-open'")
	exec("INSERT INTO organization_slugs(organization_id,slug,is_current) VALUES(?,'sunday-open',false)", org)
	capacity := int32(42)
	exec("UPDATE rooms SET is_active=true,is_bookable=true,capacity=50 WHERE id='K.201'")
	var teaching atomic.Bool
	var timetableDown atomic.Bool
	timetable := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if timetableDown.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		lessons := []int{}
		if teaching.Load() {
			lessons = []int{1}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"roomId": r.URL.Query().Get("room"), "from": r.URL.Query().Get("from"), "to": r.URL.Query().Get("to"), "events": lessons, "coverage": "partial"})
	}))
	defer timetable.Close()
	permissions := authz.NewEnforcer(db, testsupport.AllowedPermissions(t))
	store := &eventCoverStore{}
	publisher := domainevents.NewDurablePublisher(db, nil)
	eventService := events.NewService(db, nil, permissions, store, publisher, "", "")
	roomService := rooms.NewService(db, nil, permissions, publisher)
	newAuthor := func() *EventAuthor { return NewEventAuthor(db, permissions, eventService, roomService, timetable.URL) }
	author := newAuthor()
	start := time.Now().UTC().AddDate(0, 0, 3).Truncate(time.Hour)
	draft := EventRequest{OwnerID: owner, RequestID: uuid.NewString(), Title: "Hermes synthetic draft", Description: "Workshop", StartTime: start.Format(time.RFC3339), EndTime: start.Add(time.Hour).Format(time.RFC3339), Capacity: &capacity}
	// Racing deliveries on separate connections and a new service instance replay the same result.
	var wg sync.WaitGroup
	results := make([]EventResult, 5)
	errs := make([]error, len(results))
	for i := range results {
		wg.Go(func() { results[i], errs[i] = newAuthor().Run(ctx, "draft", draft) })
	}
	wg.Wait()
	eventID := results[0].EventID
	for i, r := range results {
		if errs[i] != nil || r.EventID != eventID || eventID < 1 || r.Status != "draft" {
			t.Fatalf("draft %d: %+v %v", i, r, errs[i])
		}
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := db.NewRaw(query, args...).Scan(ctx, &n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if count("SELECT count(*) FROM events WHERE title=? AND user_id=? AND organization_id=? AND status='draft' AND publish_on_sunday_open AND structured_location->>'room_id' IS NULL AND image_url LIKE '%/og/events-fallback-v1.png'", draft.Title, actor, org) != 1 || count("SELECT count(*) FROM booking_requests WHERE event_id=?", eventID) != 0 {
		t.Fatal("draft did not pin host/actor/cover or created a booking")
	}
	changed := draft
	changed.Title = "Different"
	if _, err := author.Run(ctx, "draft", changed); connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("mutation key reused: %v", err)
	}
	other, err := eventService.CreateEvent(ctx, connect.NewRequest(&eventsv1.CreateEventRequest{Title: "Outside Hermes", Description: "Example", OrganizationId: org, StartTime: draft.StartTime, EndTime: draft.EndTime, Status: eventsv1.EventStatus_EVENT_STATUS_DRAFT.Enum()}))
	if err != nil {
		t.Fatal(err)
	}
	publish := EventRequest{OwnerID: owner, RequestID: uuid.NewString(), EventID: other.Msg.Event.Id}
	if _, err := author.Run(ctx, "publish", publish); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("foreign event reachable: %v", err)
	}
	if _, err := author.Run(ctx, "rooms", publish); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("room lookup exposed a foreign draft: %v", err)
	}
	publish.EventID = eventID
	if _, err := author.Run(ctx, "publish", publish); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("draft published without a room: %v", err)
	}
	search := EventRequest{OwnerID: owner, RequestID: uuid.NewString(), StartTime: draft.StartTime, EndTime: draft.EndTime, Capacity: &capacity, RoomID: "K.201"}
	status := func(want string) {
		t.Helper()
		r, err := author.Run(ctx, "rooms", search)
		if err != nil || r.Rooms == nil || len(*r.Rooms) != 1 || (*r.Rooms)[0].Status != want {
			t.Fatalf("room search want %q: %+v %v", want, r, err)
		}
	}
	status("unknown")
	book := search
	book.RequestID = uuid.NewString()
	book.EventID = eventID
	if _, err := author.Run(ctx, "book", book); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unchecked partial schedule: %v", err)
	}
	book.OwnerVerifiedAvailability = true
	timetableDown.Store(true)
	status("unavailable")
	if _, err := author.Run(ctx, "book", book); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("offline timetable bypassed: %v", err)
	}
	timetableDown.Store(false)
	teaching.Store(true)
	status("busy")
	if _, err := author.Run(ctx, "book", book); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("teaching conflict bypassed: %v", err)
	}
	teaching.Store(false)
	exec("INSERT INTO booking_requests(room_id,requester_id,title,start_time,end_time,status) VALUES('K.201',?,'Other request',?,?,'pending')", actor, draft.StartTime, draft.EndTime)
	status("busy")
	if _, err := author.Run(ctx, "book", book); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("pending request bypassed: %v", err)
	}
	exec("DELETE FROM booking_requests WHERE title='Other request' AND requester_id=?", actor)
	exec("UPDATE events SET structured_location='{\"type\":\"campus_room\",\"room_id\":\"K.201\"}'::jsonb,status='published' WHERE id=?", other.Msg.Event.Id)
	status("busy")
	// A rejected/cancelled request leaves the old room on an editable draft.
	// Only a live request, reservation or published room assignment holds the slot.
	exec("UPDATE events SET status='draft' WHERE id=?", other.Msg.Event.Id)
	for _, inactiveStatus := range []string{"rejected", "cancelled"} {
		exec("INSERT INTO booking_requests(room_id,requester_id,event_id,title,start_time,end_time,status) VALUES('K.201',?,?,'Old request',?,?,?)", actor, other.Msg.Event.Id, draft.StartTime, draft.EndTime, inactiveStatus)
		status("unknown")
	}
	exec("UPDATE events SET status='cancelled' WHERE id=?", other.Msg.Event.Id)
	old := book
	old.EndTime = start.Add(2 * time.Hour).Format(time.RFC3339)
	if _, err := author.Run(ctx, "book", old); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("stale confirmation accepted: %v", err)
	}
	exec("UPDATE events SET start_time=start_time+interval '1 day', end_time=end_time+interval '1 day', capacity=NULL WHERE id=?", eventID)
	latest := search
	latest.EventID = eventID
	fresh, err := author.Run(ctx, "rooms", latest)
	if err != nil || fresh.Capacity != nil || fresh.StartTime != start.AddDate(0, 0, 1).Format(time.RFC3339) || fresh.EndTime == search.EndTime {
		t.Fatalf("room suggestions used obsolete draft facts: %+v %v", fresh, err)
	}
	if _, err := author.Run(ctx, "book", book); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("confirmation for an old draft accepted: %v", err)
	}
	exec("UPDATE events SET start_time=?,end_time=?,capacity=? WHERE id=?", draft.StartTime, draft.EndTime, capacity, eventID)
	for i := range results {
		wg.Go(func() { results[i], errs[i] = newAuthor().Run(ctx, "book", book) })
	}
	wg.Wait()
	for i, r := range results {
		if errs[i] != nil || r.BookingStatus != "pending" || r.EventID != eventID {
			t.Fatalf("booking %d: %+v %v", i, r, errs[i])
		}
	}
	if count("SELECT count(*) FROM booking_requests WHERE event_id=? AND status='pending'", eventID) != 1 || count("SELECT count(*) FROM room_bookings WHERE event_id=?", eventID) != 0 {
		t.Fatal("staff booking was duplicated or auto-approved")
	}
	if _, err := author.Run(ctx, "publish", publish); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("room approval bypassed: %v", err)
	}
	var bookingID int32
	if err := db.NewRaw("SELECT id FROM booking_requests WHERE event_id=?", eventID).Scan(ctx, &bookingID); err != nil {
		t.Fatal(err)
	}
	if _, err := roomService.ApproveBookingRequest(ctx, connect.NewRequest(&roomsv1.ApproveBookingRequestRequest{Id: bookingID})); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		r, err := newAuthor().Run(ctx, "publish", publish)
		if err != nil || r.Status != "published" {
			t.Fatalf("publish/replay: %+v %v", r, err)
		}
	}
	if count("SELECT count(*) FROM hermes_event_mutations WHERE event_id=?", eventID) != 3 {
		t.Fatal("unexpected durable receipts")
	}

	// Simulate failure after storage accepted an upload. Retrying must reuse its validated object.
	paid := draft
	paid.Paid = true
	paid.Title = "Hermes synthetic paid"
	paid.RequestID = uuid.NewString()
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	paid.Cover = imageBytes.Bytes()
	if _, err := author.Run(ctx, "draft", paid); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("paid draft missing its form returned %v", err)
	}
	if count("SELECT count(*) FROM events WHERE title=?", paid.Title) != 0 {
		t.Fatal("partial paid draft committed")
	}
	files := registrationfiles.New(db, struct{ storage.PrivateProvider }{})
	forms := registrationforms.NewService(db, permissions, files)
	r, err := newAuthor().WithPaidForm(forms.PrepareHermesPaidForm).Run(ctx, "draft", paid)
	if err != nil || r.Status != "draft" || len(store.keys) != 2 || store.keys[0] != store.keys[1] {
		t.Fatalf("non-atomic cover/draft retry: %+v %v keys=%v", r, err, store.keys)
	}
	if count(`SELECT count(*) FROM event_registration_forms f JOIN event_registration_form_versions v ON f.published_version_id=v.id
		WHERE f.event_id=? AND v.definition->>'approvalMode'='APPROVAL_MODE_MANUAL'
		AND v.definition->'fields'->0->>'id'='payment-receipt' AND v.definition->'fields'->0->>'required'='true'`, r.EventID) != 1 ||
		count("SELECT count(*) FROM event_registration_forms WHERE event_id=?", eventID) != 0 {
		t.Fatal("paid and free registration requirements differ from the draft instructions")
	}
}
