package models

import (
	"time"

	"github.com/uptrace/bun"
)

type OrganizationSubscription struct {
	bun.BaseModel `bun:"table:organization_subscriptions,alias:os"`

	ID             int32     `bun:"id,pk,autoincrement" json:"id"`
	OrganizationID int32     `bun:"organization_id,notnull" json:"organization_id"`
	UserID         int32     `bun:"user_id,notnull" json:"user_id"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:current_timestamp" json:"created_at"`
}
