package models

import "time"

// Priority constants map one-to-one onto the P0-P4 strings stored in the
// incidents.priority column (CHECK constraint enforces these).
const (
	PriorityP0 = "P0"
	PriorityP1 = "P1"
	PriorityP2 = "P2"
	PriorityP3 = "P3"
	PriorityP4 = "P4"
)

// Status constants map onto the incidents.status column.
const (
	StatusOpen     = "open"
	StatusWatching = "watching"
	StatusResolved = "resolved"
)

// Event type constants for incident_events.event_type.
const (
	EventComment      = "comment"
	EventStatusChange = "status_change"
	EventEscalation   = "escalation"
	EventAutoOpen     = "auto_open"
)

type Incident struct {
	ID          string
	TenantID    *string
	Service     *string
	Title       string
	Description string
	Priority    string
	Status      string
	CreatedBy   *string
	ResolvedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Events      []IncidentEvent
}

type IncidentEvent struct {
	ID         string
	IncidentID string
	ActorID    *string
	EventType  string
	Content    string
	CreatedAt  time.Time
}

// IsValidPriority reports whether p matches the CHECK constraint.
func IsValidPriority(p string) bool {
	switch p {
	case PriorityP0, PriorityP1, PriorityP2, PriorityP3, PriorityP4:
		return true
	}
	return false
}

// IsValidStatus reports whether s matches the CHECK constraint.
func IsValidStatus(s string) bool {
	switch s {
	case StatusOpen, StatusWatching, StatusResolved:
		return true
	}
	return false
}

// IsValidTransition validates incident status transitions.
// Legal transitions: open → watching, open → resolved, watching → open,
// watching → resolved. resolved is terminal.
func IsValidTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusOpen:
		return to == StatusWatching || to == StatusResolved
	case StatusWatching:
		return to == StatusOpen || to == StatusResolved
	case StatusResolved:
		return false
	}
	return false
}
