//go:build integration

package events

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	eventsv1 "github.com/studyverse/ems-backend/gen/events/v1"
	internaldb "github.com/studyverse/ems-backend/internal/db"
	"github.com/studyverse/ems-backend/internal/models"
	"github.com/uptrace/bun"
)

const testDatabaseURL = "postgres://postgres:postgres@localhost:5432/eventdb?sslmode=disable"

func TestCreateEvent_UnauthenticatedIsDenied(t *testing.T) {
	svc := &Service{}
	req := connect.NewRequest(&eventsv1.CreateEventRequest{
		Title:              "unauthenticated-create",
		Description:        `[{"type":"p","children":[{"text":"runtime test"}]}]`,
		OrganizationId:     1,
		StartTime:          time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
		EndTime:            time.Now().UTC().Add(3 * time.Hour).Format(time.RFC3339),
		StructuredLocation: &eventsv1.EventLocation{Type: eventsv1.LocationType_LOCATION_TYPE_ONLINE},
	})

	_, err := svc.CreateEvent(context.Background(), req)
	if err == nil {
		t.Fatal("expected unauthenticated error")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated, got %v", connect.CodeOf(err))
	}
}

func openCreateEventTestDB(t *testing.T) *bun.DB {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = testDatabaseURL
	}

	pool, err := internaldb.NewPool(context.Background(), databaseURL, false)
	if err != nil {
		t.Fatalf("open test pool: %v", err)
	}

	db := internaldb.NewBunDB(pool, false)
	t.Cleanup(func() {
		_ = internaldb.CloseBunDB(db)
		pool.Close()
	})

	return db
}

func ensureSystemImportUser(t *testing.T, db *bun.DB) int32 {
	t.Helper()

	ctx := context.Background()
	var user models.User
	err := db.NewSelect().Model(&user).Where("kratos_id = ?", "system-import").Scan(ctx)
	if err == nil {
		return user.ID
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("select system-import user: %v", err)
	}

	user = models.User{
		KratosID: ptrString("system-import"),
		Username: fmt.Sprintf("system-import-%d", time.Now().UnixNano()),
		Email:    "system@import.local",
		Password: "kratos-managed",
	}
	if _, err := db.NewInsert().Model(&user).Returning("*").Exec(ctx); err != nil {
		t.Fatalf("insert system-import user: %v", err)
	}

	return user.ID
}

func ensureCategory(t *testing.T, db *bun.DB) int32 {
	t.Helper()

	ctx := context.Background()
	var category models.Category
	err := db.NewSelect().Model(&category).OrderExpr("id ASC").Limit(1).Scan(ctx)
	if err == nil {
		return category.ID
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("select category: %v", err)
	}

	category = models.Category{
		Name:  fmt.Sprintf("Test Category %d", time.Now().UnixNano()),
		Color: "#3366FF",
	}
	if _, err := db.NewInsert().Model(&category).Returning("*").Exec(ctx); err != nil {
		t.Fatalf("insert category: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.NewDelete().Model((*models.Category)(nil)).Where("id = ?", category.ID).Exec(context.Background())
	})

	return category.ID
}

func ensureOrganization(t *testing.T, db *bun.DB) int32 {
	t.Helper()

	ctx := context.Background()
	var organization models.Organization
	err := db.NewSelect().Model(&organization).OrderExpr("id ASC").Limit(1).Scan(ctx)
	if err == nil {
		return organization.ID
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("select organization: %v", err)
	}

	organizationTypeID := ensureOrganizationType(t, db)
	organization = models.Organization{
		Title:              fmt.Sprintf("Test Organization %d", time.Now().UnixNano()),
		OrganizationTypeID: organizationTypeID,
	}
	if _, err := db.NewInsert().Model(&organization).Returning("*").Exec(ctx); err != nil {
		t.Fatalf("insert organization: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.NewDelete().Model((*models.Organization)(nil)).Where("id = ?", organization.ID).Exec(context.Background())
	})

	return organization.ID
}

func ensureOrganizationType(t *testing.T, db *bun.DB) int32 {
	t.Helper()

	ctx := context.Background()
	var organizationType models.OrganizationType
	err := db.NewSelect().Model(&organizationType).OrderExpr("id ASC").Limit(1).Scan(ctx)
	if err == nil {
		return organizationType.ID
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("select organization type: %v", err)
	}

	organizationType = models.OrganizationType{Title: fmt.Sprintf("Test Org Type %d", time.Now().UnixNano())}
	if _, err := db.NewInsert().Model(&organizationType).Returning("*").Exec(ctx); err != nil {
		t.Fatalf("insert organization type: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.NewDelete().Model((*models.OrganizationType)(nil)).Where("id = ?", organizationType.ID).Exec(context.Background())
	})

	return organizationType.ID
}
