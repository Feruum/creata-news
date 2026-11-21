package main

import (
	"connectrpc.com/connect"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
	"net/http"
	"strconv"
)

type platformRoleAssigner interface {
	AssignPlatformRole(context.Context, int32, string) error
}

func adminPromotionHandler(db *bun.DB, assigner platformRoleAssigner, secret string) http.HandlerFunc {
	expected := sha256.Sum256([]byte(secret))
	return func(w http.ResponseWriter, request *http.Request) {
		candidate := request.Header.Get("X-Admin-Secret")
		received := sha256.Sum256([]byte(candidate))
		if secret == "" || candidate == "" || subtle.ConstantTimeCompare(expected[:], received[:]) != 1 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		userID := request.URL.Query().Get("user_id")
		email := request.URL.Query().Get("email")
		role := request.URL.Query().Get("role")
		if userID == "" && email == "" {
			http.Error(w, "user_id or email is required", http.StatusBadRequest)
			return
		}
		if role == "" {
			role = "administrator"
		}
		if role != "administrator" && role != "staff" && role != "none" {
			http.Error(w, "role must be administrator, staff or none", http.StatusBadRequest)
			return
		}
		if userID == "" {
			var user models.User
			if err := db.NewSelect().Model(&user).Where("email=?", email).Scan(request.Context()); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.Error(w, "user not found", http.StatusNotFound)
				} else {
					http.Error(w, "user lookup failed", http.StatusInternalServerError)
				}
				return
			}
			userID = fmt.Sprint(user.ID)
		}
		id, err := strconv.ParseInt(userID, 10, 32)
		if err != nil || id <= 0 {
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}
		if err := assigner.AssignPlatformRole(request.Context(), int32(id), role); err != nil {
			code := http.StatusInternalServerError
			switch connect.CodeOf(err) {
			case connect.CodeNotFound:
				code = http.StatusNotFound
			case connect.CodeInvalidArgument:
				code = http.StatusBadRequest
			case connect.CodeUnavailable:
				code = http.StatusServiceUnavailable
			}
			http.Error(w, "platform role assignment failed", code)
			return
		}
		_, _ = fmt.Fprintf(w, "User %d assigned platform role %s", id, role)
	}
}
