package store

import (
	"database/sql"
	"time"
)

func int64Ptr(v int64) *int64 { return new(v) }

func PtrFromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func NullInt64FromPtr(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func ParseNullableTime(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, value.String)
	if err != nil {
		return nil
	}
	return &t
}
