package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"slate/services/incident-service/internal/models"
)

func newMockRepo(t *testing.T) (*Repository, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	return New(db), mock, db
}

func TestCreateIncident_HappyPath(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	created := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO incidents`)).
		WithArgs("DB slow", "latency p99 > 2s", "P1", nil, "course-service", nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("abc-123", created, created))

	svc := "course-service"
	inc, err := repo.CreateIncident(context.Background(), CreateIncidentParams{
		Service:  &svc,
		Title:    "DB slow",
		Priority: "P1",
		Impact:   "latency p99 > 2s",
	})
	if err != nil {
		t.Fatalf("CreateIncident: %v", err)
	}
	if inc.ID != "abc-123" || inc.Status != "open" || inc.Priority != "P1" {
		t.Errorf("unexpected incident: %+v", inc)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreateIncident_InvalidPriority(t *testing.T) {
	repo, _, db := newMockRepo(t)
	defer db.Close()

	_, err := repo.CreateIncident(context.Background(), CreateIncidentParams{
		Title:    "x",
		Priority: "SEV1",
	})
	if err != ErrInvalidPriority {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestCreateIncident_EmptyTitle(t *testing.T) {
	repo, _, db := newMockRepo(t)
	defer db.Close()

	_, err := repo.CreateIncident(context.Background(), CreateIncidentParams{
		Title:    "   ",
		Priority: "P2",
	})
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestUpdateIncident_OpenToResolved(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title`)).
		WithArgs("inc-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "priority", "status", "tenant_id", "service",
			"created_by", "resolved_at", "created_at", "updated_at",
		}).AddRow("inc-1", "t", "imp", "P2", "open", nil, nil, nil, nil, now, now))
	// The UPDATE has a dynamic resolved_at expression; use a regex that tolerates it.
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE incidents`)).
		WithArgs("resolved", "P2", "t", "imp", "inc-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "resolved_at"}).
			AddRow("inc-1", now, now, now))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO incident_events`)).
		WithArgs("inc-1", nil, "status open -> resolved").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	resolved := "resolved"
	inc, err := repo.UpdateIncident(context.Background(), UpdateIncidentParams{
		ID:     "inc-1",
		Status: &resolved,
	})
	if err != nil {
		t.Fatalf("UpdateIncident: %v", err)
	}
	if inc.Status != "resolved" {
		t.Errorf("expected status resolved, got %q", inc.Status)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestUpdateIncident_ResolvedTerminal(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title`)).
		WithArgs("inc-2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "priority", "status", "tenant_id", "service",
			"created_by", "resolved_at", "created_at", "updated_at",
		}).AddRow("inc-2", "t", "", "P1", "resolved", nil, nil, nil, now, now, now))
	mock.ExpectRollback()

	open := "open"
	_, err := repo.UpdateIncident(context.Background(), UpdateIncidentParams{
		ID:     "inc-2",
		Status: &open,
	})
	if err != ErrInvalidTransition {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestUpdateIncident_NotFound(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title`)).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	_, err := repo.UpdateIncident(context.Background(), UpdateIncidentParams{ID: "missing"})
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateIncident_OpenToWatching_NoResolvedAt(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, title`)).
		WithArgs("inc-3").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "priority", "status", "tenant_id", "service",
			"created_by", "resolved_at", "created_at", "updated_at",
		}).AddRow("inc-3", "t", "imp", "P2", "open", nil, nil, nil, nil, now, now))
	mock.ExpectQuery(regexp.QuoteMeta(`UPDATE incidents`)).
		WithArgs("watching", "P2", "t", "imp", "inc-3").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "resolved_at"}).
			AddRow("inc-3", now, now, nil))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO incident_events`)).
		WithArgs("inc-3", nil, "status open -> watching").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	watching := "watching"
	inc, err := repo.UpdateIncident(context.Background(), UpdateIncidentParams{
		ID: "inc-3", Status: &watching,
	})
	if err != nil {
		t.Fatalf("UpdateIncident: %v", err)
	}
	if inc.ResolvedAt != nil {
		t.Errorf("expected resolved_at nil, got %v", inc.ResolvedAt)
	}
}

func TestHasOpenIncidentFor(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT EXISTS`)).
		WithArgs("course-service", nil).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	got, err := repo.HasOpenIncidentFor(context.Background(), "course-service", nil)
	if err != nil {
		t.Fatalf("HasOpenIncidentFor: %v", err)
	}
	if !got {
		t.Error("expected true")
	}
}

func TestListIncidents_WithFilters(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(`SELECT id, title, description, priority, status, tenant_id, service, created_by,\s+resolved_at, created_at, updated_at\s+FROM incidents\s+WHERE status = \$1`).
		WithArgs("open").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "title", "description", "priority", "status", "tenant_id", "service",
			"created_by", "resolved_at", "created_at", "updated_at",
		}).
			AddRow("a", "title1", "d", "P1", "open", nil, "course-service", nil, nil, now, now).
			AddRow("b", "title2", "d", "P2", "open", nil, "email-service", nil, nil, now, now))

	openStr := "open"
	rows, err := repo.ListIncidents(context.Background(), ListFilter{Status: &openStr, Limit: 10})
	if err != nil {
		t.Fatalf("ListIncidents: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestAddEvent(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO incident_events`)).
		WithArgs("inc-9", nil, "comment", "hello").
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow("ev-1", now))

	ev, err := repo.AddEvent(context.Background(), "inc-9", nil, "comment", "hello")
	if err != nil {
		t.Fatalf("AddEvent: %v", err)
	}
	if ev.ID != "ev-1" {
		t.Errorf("expected ev-1, got %q", ev.ID)
	}
	if ev.EventType != models.EventComment {
		t.Errorf("expected comment, got %q", ev.EventType)
	}
}

func TestCountIncidentsSince(t *testing.T) {
	repo, mock, db := newMockRepo(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM incidents WHERE created_at >= $1`)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

	n, err := repo.CountIncidentsSince(context.Background(), time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("CountIncidentsSince: %v", err)
	}
	if n != 7 {
		t.Errorf("expected 7, got %d", n)
	}
}
