// Package grpc wires the scheduling domain service onto the generated gRPC stubs.
package grpc

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commontracing "slate/libs/common-go/tracing"
	pb "slate/services/scheduling-service/api/proto"
	"slate/services/scheduling-service/internal/repository"
	"slate/services/scheduling-service/internal/service"
)

type SchedulingServer struct {
	pb.UnimplementedSchedulingServiceServer
	sched      *service.Scheduler
	tenantSlug string
}

func NewSchedulingServer(sched *service.Scheduler, tenantSlug string) *SchedulingServer {
	return &SchedulingServer{sched: sched, tenantSlug: tenantSlug}
}

// tagSpan satisfies the CONTRACTS.md "trace.propagation" acceptance gate:
//  1. If the inbound metadata did not carry a tenant.slug, fall back to the
//     service's provisioned TenantConfig.Slug (tenant-per-container invariant).
//  2. Attach request_id + tenant.slug to the active span via common-go helper
//     so Tempo queries by correlation work across service hops.
//  3. Attach domain attributes (instructor_id / student_id) supplied by callers.
func (s *SchedulingServer) tagSpan(ctx context.Context, kvs ...attribute.KeyValue) context.Context {
	// Pin the tenant.slug on the context when it wasn't forwarded by the caller.
	if commontracing.TenantSlugFromContext(ctx) == "" && s.tenantSlug != "" {
		ctx = commontracing.WithTenantSlug(ctx, s.tenantSlug)
	}
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ctx
	}
	commontracing.TagSpanWithCorrelation(ctx, span)
	if len(kvs) > 0 {
		span.SetAttributes(kvs...)
	}
	return ctx
}

func (s *SchedulingServer) CreateSchedule(ctx context.Context, req *pb.CreateScheduleRequest) (*pb.Schedule, error) {
	ctx = s.tagSpan(ctx, attribute.String("instructor_id", req.GetInstructorId()))
	if req.GetInstructorId() == "" {
		return nil, status.Error(codes.InvalidArgument, "instructor_id is required")
	}
	var loc *string
	if req.GetLocation() != "" {
		v := req.GetLocation()
		loc = &v
	}
	sched := repository.Schedule{
		InstructorID:        req.GetInstructorId(),
		DayOfWeek:           int(req.GetDayOfWeek()),
		StartTime:           req.GetStartTime(),
		EndTime:             req.GetEndTime(),
		SlotDurationMinutes: int(req.GetSlotDurationMinutes()),
		Format:              req.GetFormat(),
		Location:            loc,
	}
	out, err := s.sched.CreateSchedule(ctx, sched)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("create schedule: %v", err))
	}
	return toPbSchedule(out), nil
}

func (s *SchedulingServer) ListSchedules(ctx context.Context, req *pb.ListSchedulesRequest) (*pb.ListSchedulesResponse, error) {
	ctx = s.tagSpan(ctx, attribute.String("instructor_id", req.GetInstructorId()))
	schedules, err := s.sched.ListSchedules(ctx, req.GetInstructorId(), req.GetActiveOnly())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list schedules: %v", err))
	}
	out := make([]*pb.Schedule, 0, len(schedules))
	for i := range schedules {
		out = append(out, toPbSchedule(&schedules[i]))
	}
	return &pb.ListSchedulesResponse{Schedules: out}, nil
}

func (s *SchedulingServer) ListAvailableSlots(ctx context.Context, req *pb.ListSlotsRequest) (*pb.ListSlotsResponse, error) {
	ctx = s.tagSpan(ctx, attribute.String("instructor_id", req.GetInstructorId()))
	slots, fromCache, err := s.sched.AvailableSlots(ctx, req.GetInstructorId(), req.GetDateFrom(), req.GetDateTo())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("list slots: %v", err))
	}
	out := make([]*pb.Slot, 0, len(slots))
	for _, sl := range slots {
		out = append(out, &pb.Slot{
			ScheduleId:    sl.ScheduleID,
			InstructorId:  sl.InstructorID,
			SlotDate:      sl.SlotDate,
			SlotStartTime: sl.SlotStartTime,
			SlotEndTime:   sl.SlotEndTime,
			Format:        sl.Format,
			Location:      sl.Location,
		})
	}
	return &pb.ListSlotsResponse{Slots: out, FromCache: fromCache}, nil
}

func (s *SchedulingServer) BookSlot(ctx context.Context, req *pb.BookSlotRequest) (*pb.Booking, error) {
	ctx = s.tagSpan(ctx,
		attribute.String("instructor_id", req.GetInstructorId()),
		attribute.String("student_id", req.GetStudentId()))
	if req.GetInstructorId() == "" || req.GetStudentId() == "" || req.GetSlotDate() == "" || req.GetSlotStartTime() == "" {
		return nil, status.Error(codes.InvalidArgument, "instructor_id, student_id, slot_date, slot_start_time required")
	}
	b, err := s.sched.Book(ctx,
		req.GetScheduleId(), req.GetInstructorId(), req.GetStudentId(),
		req.GetSlotDate(), req.GetSlotStartTime(), req.GetPreContext())
	if err != nil {
		if errors.Is(err, repository.ErrSlotTaken) {
			return nil, status.Error(codes.FailedPrecondition, "slot already booked")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("book slot: %v", err))
	}
	return toPbBooking(b), nil
}

func (s *SchedulingServer) CancelBooking(ctx context.Context, req *pb.CancelBookingRequest) (*emptypb.Empty, error) {
	ctx = s.tagSpan(ctx, attribute.String("student_id", req.GetStudentId()))
	if _, err := s.sched.Cancel(ctx, req.GetBookingId()); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "booking not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("cancel booking: %v", err))
	}
	return &emptypb.Empty{}, nil
}

func (s *SchedulingServer) GetInstructorDay(ctx context.Context, req *pb.GetInstructorDayRequest) (*pb.InstructorDayResponse, error) {
	ctx = s.tagSpan(ctx, attribute.String("instructor_id", req.GetInstructorId()))
	bookings, err := s.sched.InstructorDay(ctx, req.GetInstructorId(), req.GetSlotDate())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("instructor day: %v", err))
	}
	out := make([]*pb.Booking, 0, len(bookings))
	for i := range bookings {
		out = append(out, toPbBooking(&bookings[i]))
	}
	return &pb.InstructorDayResponse{
		InstructorId: req.GetInstructorId(),
		SlotDate:     req.GetSlotDate(),
		Bookings:     out,
	}, nil
}

func toPbSchedule(s *repository.Schedule) *pb.Schedule {
	out := &pb.Schedule{
		Id:                  s.ID,
		InstructorId:        s.InstructorID,
		DayOfWeek:           int32(s.DayOfWeek),
		StartTime:           s.StartTime,
		EndTime:             s.EndTime,
		SlotDurationMinutes: int32(s.SlotDurationMinutes),
		Format:              s.Format,
		IsActive:            s.IsActive,
		CreatedAt:           timestamppb.New(s.CreatedAt),
	}
	if s.Location != nil {
		out.Location = *s.Location
	}
	return out
}

func toPbBooking(b *repository.Booking) *pb.Booking {
	out := &pb.Booking{
		Id:            b.ID,
		InstructorId:  b.InstructorID,
		StudentId:     b.StudentID,
		SlotDate:      b.SlotDate,
		SlotStartTime: b.SlotStartTime,
		Status:        b.Status,
		CreatedAt:     timestamppb.New(b.CreatedAt),
	}
	if b.ScheduleID != nil {
		out.ScheduleId = *b.ScheduleID
	}
	if b.Questions != nil {
		out.Questions = *b.Questions
	}
	return out
}
