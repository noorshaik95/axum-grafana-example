package repository

import (
	"database/sql/driver"

	"github.com/lib/pq"
)

// pqStringArray wraps a string slice for use with PostgreSQL `ANY($1)` on text[] / uuid[].
func pqStringArray(s []string) driver.Valuer {
	return pq.Array(s)
}
