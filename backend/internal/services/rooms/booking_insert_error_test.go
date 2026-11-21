package rooms

import (
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRoomBookingInsertErrorPreservesProductionDriverConflictCode(t *testing.T) {
	conflict := &pgconn.PgError{Code: "23P01", Message: "Booking overlaps with existing booking for this room"}
	err := roomBookingInsertError(fmt.Errorf("insert reservation: %w", conflict))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("wrapped production driver occupancy conflict = %v, want failed_precondition", err)
	}
}

func TestRoomBookingInsertErrorPreservesUnexpectedFailure(t *testing.T) {
	failed := errors.New("database connection lost")
	err := roomBookingInsertError(failed)
	if connect.CodeOf(err) != connect.CodeInternal || !errors.Is(err, failed) {
		t.Fatalf("unexpected booking failure must remain internal with its cause: %v", err)
	}
}
