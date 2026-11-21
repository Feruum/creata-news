package search

import "testing"

func TestCampusLocationUsesStringAndLegacyRoomIDs(t *testing.T) {
	for _, roomID := range []string{`"A-101"`, `101`} {
		location, kind := ExtractLocationFromStructured([]byte(`{"type":"campus_room","room_id":` + roomID + `,"room_name":"Workshop room","floor":"1"}`))
		if location != "Workshop room, Floor 1" || kind != "campus_room" {
			t.Errorf("campus location with room_id=%s was lost: location=%q kind=%q", roomID, location, kind)
		}
	}
}

func TestCampusLocationFloorValues(t *testing.T) {
	for _, tc := range []struct{ name, floor, wantLocation string }{
		{"numeric", `1`, "Workshop room, Floor 1"},
		{"zero", `0`, "Workshop room, Floor 0"},
		{"basement", `-1`, "Workshop room, Floor -1"},
		{"legacy numeric string", `"1"`, "Workshop room, Floor 1"},
		{"legacy named string", `"Ground"`, "Workshop room, Floor Ground"},
		{"missing", ``, "Workshop room"},
		{"null", `null`, "Workshop room"},
		{"empty string", `""`, "Workshop room"},
		{"boolean", `true`, "Workshop room"},
		{"object", `{}`, "Workshop room"},
		{"array", `[]`, "Workshop room"},
		{"fractional", `1.5`, "Workshop room"},
		{"out of range", `2147483648`, "Workshop room"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := `{"type":"campus_room","room_name":"Workshop room"`
			if tc.floor != "" {
				payload += `,"floor":` + tc.floor
			}
			payload += `}`
			location, kind := ExtractLocationFromStructured([]byte(payload))
			if location != tc.wantLocation || kind != "campus_room" {
				t.Fatalf("location=%q kind=%q; want %q, campus_room", location, kind, tc.wantLocation)
			}
		})
	}
}

func TestStructuredLocationKindsAndInvalidPayloads(t *testing.T) {
	for _, tc := range []struct{ name, payload, wantLocation, wantKind string }{
		{"campus without room", `{"type":"campus_room","room_id":"A-101","floor":1}`, "", "campus_room"},
		{"campus with empty room", `{"type":"campus_room","room_name":"","floor":1}`, "", "campus_room"},
		{"external", `{"type":"external","external_address":"Astana, 10 Main Street","floor":1}`, "Astana, 10 Main Street", "external"},
		{"online platform", `{"type":"online","online_platform":"Zoom","floor":1}`, "Zoom", "online"},
		{"online URL", `{"type":"online","online_url":"https://example.test"}`, "Online", "online"},
		{"online without details", `{"type":"online"}`, "", "online"},
		{"hybrid room", `{"type":"hybrid","room_name":"Workshop room","floor":1,"online_platform":"Zoom"}`, "Workshop room + Zoom", "hybrid"},
		{"hybrid external", `{"type":"hybrid","external_address":"Main hall","online_platform":"Zoom"}`, "Main hall + Zoom", "hybrid"},
		{"hybrid fallback", `{"type":"hybrid","room_name":"Workshop room"}`, "Workshop room + Online", "hybrid"},
		{"unknown", `{"type":"unknown","room_name":"Workshop room","floor":1}`, "", "unknown"},
		{"empty", ``, "", ""},
		{"null", `null`, "", ""},
		{"malformed", `{"type":"campus_room","room_name":"Workshop room","floor":`, "", ""},
		{"invalid room", `{"type":"campus_room","room_name":true,"floor":1}`, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			location, kind := ExtractLocationFromStructured([]byte(tc.payload))
			if location != tc.wantLocation || kind != tc.wantKind {
				t.Fatalf("location=%q kind=%q; want %q, %q", location, kind, tc.wantLocation, tc.wantKind)
			}
		})
	}
}
