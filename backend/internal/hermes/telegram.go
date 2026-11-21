package hermes

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

type TelegramStore struct{ db *bun.DB }

func NewTelegramStore(db *bun.DB) *TelegramStore { return &TelegramStore{db: db} }

// UserCount uses the same Studyverse user table as the administration dashboard.
func (s *TelegramStore) UserCount(ctx context.Context) (int, error) {
	if s == nil || s.db == nil {
		return 0, ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return s.db.NewSelect().Model((*models.User)(nil)).Count(ctx)
}

func (s *TelegramStore) VerifiedLink(ctx context.Context, kratosID string) (TelegramLink, error) {
	if s == nil || s.db == nil {
		return TelegramLink{}, ErrUnavailable
	}
	if kratosID == "" {
		return TelegramLink{}, ErrDenied
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	var user models.User
	err := s.db.NewSelect().Model(&user).Column("telegram_id", "telegram_linked_at").Where("kratos_id = ?", kratosID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return TelegramLink{}, ErrDenied
	}
	if err != nil {
		return TelegramLink{}, ErrUnavailable
	}
	if user.TelegramID == nil || *user.TelegramID <= 0 || user.TelegramLinkedAt == nil || user.TelegramLinkedAt.IsZero() || user.TelegramLinkedAt.After(time.Now()) {
		return TelegramLink{}, ErrDenied
	}
	return TelegramLink{ID: strconv.FormatInt(*user.TelegramID, 10), LinkedAt: user.TelegramLinkedAt.UTC()}, nil
}
