package hermes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestEventActionInputIsScoped(t *testing.T) {
	req := EventRequest{RequestID: uuid.NewString(), StartTime: "2028-01-01T08:00:00+05:00", EndTime: "2028-01-01T09:00:00+05:00", Title: "A", Description: "B"}
	for _, tc := range []struct {
		name   string
		mutate func(*EventRequest)
	}{
		{"preselected room", func(r *EventRequest) { r.RoomID = "K.201" }},
		{"reserved event ID", func(r *EventRequest) { r.EventID = 1 }},
		{"negative interval", func(r *EventRequest) { r.EndTime = r.StartTime }},
		{"invalid cover", func(r *EventRequest) { r.Cover = []byte("<svg/>") }},
		{"online credentials", func(r *EventRequest) { r.Format = "online"; r.OnlineURL = "https://secret@meet.example" }},
		{"arbitrary URL scheme", func(r *EventRequest) { r.Format = "online"; r.OnlineURL = "file:///private" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := req
			tc.mutate(&r)
			if err := r.validate("draft"); err == nil {
				t.Fatal("invalid draft accepted")
			}
		})
	}
	req.Format = "hybrid"
	req.OnlineURL = "https://meet.example/class"
	if err := req.validate("draft"); err != nil {
		t.Fatal(err)
	}
}

func TestRoomTimetableEvidence(t *testing.T) {
	const start, end = "2028-01-01T08:00:00Z", "2028-01-01T09:00:00Z"
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{"partial", `{"roomId":"K.201","from":"` + start + `","to":"` + end + `","events":[],"coverage":"partial"}`, timetablePartial, 200},
		{"class", `{"roomId":"K.201","from":"` + start + `","to":"` + end + `","events":[{}],"coverage":"partial"}`, timetableBusy, 200},
		{"complete", `{"roomId":"K.201","from":"` + start + `","to":"` + end + `","events":[],"coverage":"complete"}`, timetableFree, 200},
		{"missing rows", `{"roomId":"K.201","from":"` + start + `","to":"` + end + `","coverage":"complete"}`, timetableUnavailable, 200},
		{"wrong interval", `{"roomId":"K.201","from":"yesterday","events":[],"coverage":"complete"}`, timetableUnavailable, 200},
		{"not in timetable", `{"error":"Choose a room from the campus map."}`, timetablePartial, 400},
		{"failed", `{}`, timetableUnavailable, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Query().Get("room") != "K.201" || r.URL.Query().Get("from") != start || r.URL.Query().Get("to") != end || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
					t.Fatal("unexpected timetable request")
				}
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			if got := newRoomTimetable(server.URL).status(t.Context(), "K.201", start, end); got != tc.want {
				t.Fatalf("status=%q want %q", got, tc.want)
			}
		})
	}
}
