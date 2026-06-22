package store

import "database/sql"

func int64Ptr(v int64) *int64 { return &v }

func ptrFromNullInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func nullInt64FromPtr(v *int64) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func ptrFromValidInt64(value int64, valid int64) *int64 {
	if valid == 0 {
		return nil
	}
	return &value
}

func validInt64FromPtr(v *int64) (value int64, valid int64) {
	if v == nil {
		return 0, 0
	}
	return *v, 1
}
