package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type recordingRoleAssigner struct {
	calls int
	id    int32
	role  string
	err   error
}

func (a *recordingRoleAssigner) AssignPlatformRole(_ context.Context, id int32, role string) error {
	a.calls++
	a.id = id
	a.role = role
	return a.err
}
func TestBootstrapPromotionAuthorizesBeforeAssigning(t *testing.T) {
	for _, secret := range []string{"", "incorrect"} {
		assigner := &recordingRoleAssigner{}
		request := httptest.NewRequest(http.MethodPost, "/admin/promote?user_id=42", nil)
		request.Header.Set("X-Admin-Secret", secret)
		response := httptest.NewRecorder()
		adminPromotionHandler(nil, assigner, "configured-secret")(response, request)
		if response.Code != http.StatusUnauthorized || assigner.calls != 0 {
			t.Fatalf("unauthorized assignment: status=%d calls=%d", response.Code, assigner.calls)
		}
	}
}
func TestBootstrapPromotionUsesValidatedDurableRoleAssigner(t *testing.T) {
	for _, test := range []struct {
		query, role string
		status      int
	}{
		{"?user_id=42", "administrator", http.StatusOK}, {"?user_id=42&role=staff", "staff", http.StatusOK}, {"?user_id=42&role=none", "none", http.StatusOK},
		{"?user_id=42&role=owner", "", http.StatusBadRequest}, {"?user_id=-1", "", http.StatusBadRequest}, {"?user_id=2147483648", "", http.StatusBadRequest}, {"", "", http.StatusBadRequest},
	} {
		t.Run(test.query, func(t *testing.T) {
			assigner := &recordingRoleAssigner{}
			request := httptest.NewRequest(http.MethodPost, "/admin/promote"+test.query, nil)
			request.Header.Set("X-Admin-Secret", "configured-secret")
			response := httptest.NewRecorder()
			adminPromotionHandler(nil, assigner, "configured-secret")(response, request)
			if response.Code != test.status {
				t.Fatalf("status %d, expected %d", response.Code, test.status)
			}
			if test.status == http.StatusOK {
				if assigner.calls != 1 || assigner.id != 42 || assigner.role != test.role {
					t.Fatalf("incorrect assignment: %+v", assigner)
				}
			} else if assigner.calls != 0 {
				t.Fatal("invalid request assigned role")
			}
		})
	}
	assigner := &recordingRoleAssigner{err: errors.New("durable assignment failed")}
	request := httptest.NewRequest(http.MethodPost, "/admin/promote?user_id=42", nil)
	request.Header.Set("X-Admin-Secret", "configured-secret")
	response := httptest.NewRecorder()
	adminPromotionHandler(nil, assigner, "configured-secret")(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatal("failed assignment reported success")
	}
}
