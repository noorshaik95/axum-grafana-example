package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/protobuf/types/known/emptypb"

	pb "slate/services/incident-service/api/proto"
	grpcserver "slate/services/incident-service/internal/grpc"
)

type HTTPHandler struct {
	grpc *grpcserver.Server
}

func NewHTTPHandler(grpc *grpcserver.Server) *HTTPHandler {
	return &HTTPHandler{grpc: grpc}
}

// PublicStatus handles GET /api/status. Public route: no auth required.
// The gateway already declares this path as public per plan W17 routing.
func (h *HTTPHandler) PublicStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	resp, err := h.grpc.GetPublicStatus(ctx, &emptypb.Empty{})
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusInternalServerError)
		return
	}

	out := toPublicStatusJSON(resp)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=30")
	_ = json.NewEncoder(w).Encode(out)
}

type publicStatusJSON struct {
	Overall           string                 `json:"overall"`
	Components        []publicComponentJSON  `json:"components"`
	IncidentsLast7D   int32                  `json:"incidents_last_7d"`
	GeneratedAtUnixMs int64                  `json:"generated_at_unix_ms"`
}

type publicComponentJSON struct {
	Service         string  `json:"service"`
	Health          string  `json:"health"`
	OpenIncidents   int32   `json:"open_incidents"`
	HighestPriority *string `json:"highest_priority,omitempty"`
}

func toPublicStatusJSON(resp *pb.PublicStatusResponse) publicStatusJSON {
	out := publicStatusJSON{
		Overall:           overallToString(resp.GetOverall()),
		IncidentsLast7D:   resp.GetIncidentsLast_7D(),
		GeneratedAtUnixMs: resp.GetGeneratedAtUnixMs(),
	}
	for _, c := range resp.GetComponents() {
		comp := publicComponentJSON{
			Service:       c.GetService(),
			Health:        healthToString(c.GetHealth()),
			OpenIncidents: c.GetOpenIncidents(),
		}
		if c.HighestPriority != nil {
			s := priorityToString(*c.HighestPriority)
			comp.HighestPriority = &s
		}
		out.Components = append(out.Components, comp)
	}
	return out
}

func overallToString(o pb.OverallStatus) string {
	switch o {
	case pb.OverallStatus_OPERATIONAL:
		return "operational"
	case pb.OverallStatus_DEGRADED:
		return "degraded"
	case pb.OverallStatus_OUTAGE:
		return "outage"
	}
	return "unknown"
}

func healthToString(h pb.ComponentHealth) string {
	switch h {
	case pb.ComponentHealth_GREEN:
		return "green"
	case pb.ComponentHealth_AMBER:
		return "amber"
	case pb.ComponentHealth_RED:
		return "red"
	}
	return "unknown"
}

func priorityToString(p pb.Priority) string {
	switch p {
	case pb.Priority_P0:
		return "P0"
	case pb.Priority_P1:
		return "P1"
	case pb.Priority_P2:
		return "P2"
	case pb.Priority_P3:
		return "P3"
	case pb.Priority_P4:
		return "P4"
	}
	return ""
}
