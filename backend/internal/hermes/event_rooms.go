package hermes

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/studyverse/ems-backend/internal/roomid"
	"github.com/uptrace/bun"
)

// The map resolves campus aliases against approved, dated group schedules. An empty
// list from today's partial feed is not evidence that teaching has cleared this room.
type roomTimetable struct {
	endpoint string
	client   *http.Client
}

func newRoomTimetable(mapURL string) roomTimetable {
	origin, err := url.Parse(mapURL)
	if err != nil || (origin.Scheme != "https" && origin.Scheme != "http") || origin.Host == "" || origin.User != nil {
		return roomTimetable{}
	}
	endpoint := origin.ResolveReference(&url.URL{Path: "/api/schedule/rooms"}).String()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return roomTimetable{endpoint: endpoint, client: &http.Client{Transport: transport, Timeout: 4 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

const (
	timetableUnavailable = "unavailable"
	timetablePartial     = "partial"
	timetableFree        = "available"
	timetableBusy        = "busy"
)

func (s roomTimetable) status(ctx context.Context, room, start, end string) string {
	if s.client == nil {
		return timetableUnavailable
	}
	query := url.Values{"room": {room}, "from": {start}, "to": {end}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return timetableUnavailable
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return timetableUnavailable
	}
	defer resp.Body.Close()
	const max = 256 << 10
	data, err := io.ReadAll(io.LimitReader(resp.Body, max+1))
	if err != nil || len(data) > max {
		return timetableUnavailable
	}
	if resp.StatusCode == http.StatusBadRequest {
		var body struct{ Error string }
		if json.Unmarshal(data, &body) == nil && body.Error == "Choose a room from the campus map." {
			return timetablePartial
		}
	}
	var body struct {
		RoomID   string `json:"roomId"`
		From     string
		To       string
		Coverage string
		Events   []json.RawMessage
	}
	if resp.StatusCode != http.StatusOK || json.Unmarshal(data, &body) != nil || body.RoomID != room ||
		body.From != start || body.To != end || body.Events == nil {
		return timetableUnavailable
	}
	if len(body.Events) != 0 {
		return timetableBusy
	}
	switch body.Coverage {
	case "complete":
		return timetableFree
	case "partial":
		return timetablePartial
	default:
		return timetableUnavailable
	}
}

func (s *EventAuthor) roomOptions(ctx context.Context, db bun.IDB, req EventRequest, forBooking bool) ([]EventRoom, error) {
	selected := roomid.Normalize(req.RoomID)
	if forBooking {
		// Serialize against normal reservations/approvals and FK inserts. Read occupancy
		// in a fresh statement after any lock wait so committed competitors are visible.
		var id string
		if err := db.NewRaw("SELECT id FROM rooms WHERE id=? FOR UPDATE", selected).Scan(ctx, &id); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, eventFailure(connect.CodeNotFound, "Такой аудитории нет в каталоге.")
			}
			return nil, err
		}
	}
	var candidates []struct {
		ID       string
		Name     string
		Capacity *int32
		Busy     bool
	}
	query := db.NewSelect().TableExpr("rooms r").ColumnExpr("r.id, r.name, r.capacity").
		ColumnExpr(`EXISTS(SELECT 1 FROM room_bookings b WHERE b.room_id=r.id AND b.start_time < ? AND b.end_time > ?)
			OR EXISTS(SELECT 1 FROM booking_requests p WHERE p.room_id=r.id AND p.status='pending' AND p.start_time < ? AND p.end_time > ?)
			OR EXISTS(SELECT 1 FROM events e WHERE e.id <> ? AND e.status='published'
				AND e.start_time < ? AND e.end_time > ?
				AND replace(replace(upper(trim(COALESCE(e.structured_location->>'room_id', e.structured_location->>'roomId'))),'С','C'),'К','K')=upper(r.id)) AS busy`,
			req.EndTime, req.StartTime, req.EndTime, req.StartTime, req.EventID, req.EndTime, req.StartTime).
		Where("r.is_active AND r.is_bookable AND (r.capacity IS NULL OR r.capacity > 0)")
	if selected != "" {
		query = query.Where("r.id = ?", selected)
	}
	if req.Capacity != nil {
		query = query.Where("(r.capacity IS NULL OR r.capacity >= ?)", *req.Capacity)
	}
	if err := query.OrderExpr("busy, r.capacity ASC NULLS LAST, r.id").Limit(12).Scan(ctx, &candidates); err != nil {
		return nil, err
	}
	results := make([]EventRoom, len(candidates))
	// Fixed shortlist and bounded parallel reads. No owner, credentials or event text go to the timetable.
	var wg sync.WaitGroup
	slots := make(chan struct{}, 4)
	for i, candidate := range candidates {
		results[i] = EventRoom{ID: candidate.ID, Name: candidate.Name, Capacity: candidate.Capacity, Status: "busy"}
		if candidate.Busy {
			continue
		}
		slots <- struct{}{}
		wg.Go(func() {
			defer func() { <-slots }()
			status := s.timetable.status(ctx, candidate.ID, req.StartTime, req.EndTime)
			switch {
			case status == timetableUnavailable:
				results[i].Status = "unavailable"
			case status == timetableBusy:
				results[i].Status = "busy"
			case status == timetableFree && candidate.Capacity != nil && *candidate.Capacity > 0:
				results[i].Status = "available"
			default:
				results[i].Status = "unknown"
			}
		})
	}
	wg.Wait()
	if forBooking && len(results) == 1 && results[0].Status == "unavailable" {
		return nil, eventFailure(connect.CodeUnavailable, "Не удалось сверить расписание. Повторите проверку перед подачей заявки.")
	}
	return results, nil
}
