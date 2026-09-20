package postgres

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestNullableTime(t *testing.T) {
	if actual := nullableTime(pgtype.Timestamptz{}); actual != nil {
		t.Fatalf("expected NULL next_retry to map to nil, got %v", actual)
	}

	want := time.Date(2026, 9, 20, 3, 32, 12, 0, time.UTC)
	actual := nullableTime(pgtype.Timestamptz{Time: want, Valid: true})
	if actual == nil || !actual.Equal(want) {
		t.Fatalf("expected %v, got %v", want, actual)
	}
}
