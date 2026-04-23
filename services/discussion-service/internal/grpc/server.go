// Package grpc implements the DiscussionService gRPC server.
package grpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "slate/services/discussion-service/api/proto"
	"slate/services/discussion-service/internal/kafka"
	"slate/services/discussion-service/internal/mentions"
	"slate/services/discussion-service/internal/models"
	"slate/services/discussion-service/internal/repository"
	"slate/services/discussion-service/internal/userauth"

	commontracing "slate/libs/common-go/tracing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuthorRoleResolver answers "is this author an instructor for this course?"
// When unavailable (or the course roster is not yet replicated), callers
// default to "student" — ListThreadsNeedingReply will still work because it
// only *requires* instructor posts to silence a thread.
type AuthorRoleResolver interface {
	AuthorRole(ctx context.Context, tenantID, courseID, userID string) (string, error)
}

// Store is the subset of repository methods the server depends on. Declared
// here (not in repository/) to keep the repo package free of consumer-driven
// interfaces and to let tests inject fakes without a real database.
type Store interface {
	CreateThread(ctx context.Context, tenantID, courseID, title, createdBy string) (*models.Thread, error)
	GetThread(ctx context.Context, tenantID, threadID string) (*models.Thread, error)
	ListThreads(ctx context.Context, tenantID, courseID string, limit int, cursor string) ([]models.Thread, string, bool, error)
	CreatePost(ctx context.Context, tenantID, threadID, authorID, authorRole, content string, parentPostID *string, mentionedUserIDs []string) (*repository.CreatePostResult, error)
	ListPosts(ctx context.Context, tenantID, threadID string) ([]models.Post, error)
	GetInbox(ctx context.Context, tenantID, userID string, limit int, cursor string, unseenOnly bool) ([]models.InboxItem, string, bool, error)
	MarkInboxRead(ctx context.Context, tenantID, userID, itemID string) error
	InstructorCoursesFor(ctx context.Context, tenantID, instructorID string) ([]string, error)
	ListThreadsNeedingReply(ctx context.Context, tenantID string, courseIDs []string, limit int, now time.Time) ([]models.Thread, error)
}

// MentionPublisher emits the `discussion.mention` Kafka event. Implemented by
// kafka.Producer; tests pass a fake.
type MentionPublisher interface {
	PublishMention(ctx context.Context, evt kafka.MentionEvent) error
}

// staticInstructorResolver consults the Store's instructor_courses view.
type staticInstructorResolver struct{ store Store }

func (s staticInstructorResolver) AuthorRole(ctx context.Context, tenantID, courseID, userID string) (string, error) {
	courses, err := s.store.InstructorCoursesFor(ctx, tenantID, userID)
	if err != nil {
		return "student", err
	}
	for _, c := range courses {
		if c == courseID {
			return "instructor", nil
		}
	}
	return "student", nil
}

// Server implements DiscussionServiceServer.
type Server struct {
	pb.UnimplementedDiscussionServiceServer
	store    Store
	producer MentionPublisher
	userRes  userauth.Resolver
	roleRes  AuthorRoleResolver
	now      func() time.Time
	slug     string
}

type Options struct {
	Repo         Store
	Producer     MentionPublisher
	UserResolver userauth.Resolver
	RoleResolver AuthorRoleResolver
	TenantSlug   string
	Now          func() time.Time
}

func New(opts Options) *Server {
	if opts.UserResolver == nil {
		opts.UserResolver = userauth.NoopResolver{}
	}
	if opts.RoleResolver == nil {
		opts.RoleResolver = staticInstructorResolver{store: opts.Repo}
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	if opts.Producer == nil {
		opts.Producer = noopPublisher{}
	}
	return &Server{
		store:    opts.Repo,
		producer: opts.Producer,
		userRes:  opts.UserResolver,
		roleRes:  opts.RoleResolver,
		slug:     opts.TenantSlug,
		now:      opts.Now,
	}
}

type noopPublisher struct{}

func (noopPublisher) PublishMention(context.Context, kafka.MentionEvent) error { return nil }

// prepareCtx runs the CONTRACTS `trace.propagation` acceptance gate for every
// handler:
//  1. extract traceparent + x-request-id + x-tenant-slug from inbound gRPC md
//  2. seed context with tenant.slug from TENANT_SLUG env if not supplied by caller
//  3. synthesize a request_id when the caller did not send one (gateway always
//     sends one, but direct gRPC callers may not)
//
// The returned context carries request_id + tenant.slug for downstream helpers
// (Kafka header injection) to use.
func (s *Server) prepareCtx(ctx context.Context) context.Context {
	ctx = commontracing.ExtractTraceparent(ctx)
	if commontracing.TenantSlugFromContext(ctx) == "" && s.slug != "" {
		ctx = commontracing.WithTenantSlug(ctx, s.slug)
	}
	if commontracing.RequestIDFromContext(ctx) == "" {
		ctx = commontracing.WithRequestID(ctx, newRequestID())
	}
	return ctx
}

func (s *Server) tagSpan(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	commontracing.TagSpanWithCorrelation(ctx, span)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
}

// newRequestID is a tiny non-crypto id just for correlation when the caller
// didn't forward one. Keeping it here avoids a new dependency.
func newRequestID() string {
	var b [8]byte
	for i := range b {
		b[i] = byte(time.Now().UnixNano() >> (i * 4))
	}
	const hex = "0123456789abcdef"
	out := make([]byte, 16)
	for i, v := range b {
		out[i*2] = hex[v>>4]
		out[i*2+1] = hex[v&0x0f]
	}
	return "req-" + string(out)
}

// ---------- threads ----------

func (s *Server) CreateThread(ctx context.Context, req *pb.CreateThreadRequest) (*pb.Thread, error) {
	ctx = s.prepareCtx(ctx)
	if req.TenantId == "" || req.CourseId == "" || req.Title == "" || req.CreatedBy == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id, course_id, title, created_by required")
	}
	s.tagSpan(ctx,
		attribute.String("tenant.id", req.TenantId),
		attribute.String("course.id", req.CourseId),
	)

	t, err := s.store.CreateThread(ctx, req.TenantId, req.CourseId, req.Title, req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("create thread: %v", err))
	}
	s.tagSpan(ctx, attribute.String("thread.id", t.ID))

	// Initial post is optional — if provided, reuse CreatePost logic so
	// mentions fire consistently.
	if req.InitialPost != "" {
		if _, err := s.createPost(ctx, req.TenantId, t.ID, req.CreatedBy, req.InitialPost, nil); err != nil {
			return nil, err
		}
		// Reload to reflect reply_count bump.
		if refreshed, err := s.store.GetThread(ctx, req.TenantId, t.ID); err == nil {
			t = refreshed
		}
	}
	return threadToProto(t), nil
}

func (s *Server) ListThreads(ctx context.Context, req *pb.ListThreadsRequest) (*pb.ListThreadsResponse, error) {
	ctx = s.prepareCtx(ctx)
	if req.TenantId == "" || req.CourseId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and course_id required")
	}
	s.tagSpan(ctx,
		attribute.String("tenant.id", req.TenantId),
		attribute.String("course.id", req.CourseId),
	)
	items, next, hasMore, err := s.store.ListThreads(ctx, req.TenantId, req.CourseId, int(req.Limit), req.Cursor)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list threads: %v", err))
	}
	out := make([]*pb.Thread, 0, len(items))
	for i := range items {
		out = append(out, threadToProto(&items[i]))
	}
	return &pb.ListThreadsResponse{Threads: out, NextCursor: next, HasMore: hasMore}, nil
}

func (s *Server) GetThread(ctx context.Context, req *pb.GetThreadRequest) (*pb.ThreadWithPosts, error) {
	ctx = s.prepareCtx(ctx)
	if req.TenantId == "" || req.ThreadId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and thread_id required")
	}
	s.tagSpan(ctx,
		attribute.String("tenant.id", req.TenantId),
		attribute.String("thread.id", req.ThreadId),
	)
	t, err := s.store.GetThread(ctx, req.TenantId, req.ThreadId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("get thread: %v", err))
	}
	posts, err := s.store.ListPosts(ctx, req.TenantId, req.ThreadId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("list posts: %v", err))
	}
	pbPosts := make([]*pb.Post, 0, len(posts))
	for i := range posts {
		pbPosts = append(pbPosts, postToProto(&posts[i], nil))
	}
	return &pb.ThreadWithPosts{Thread: threadToProto(t), Posts: pbPosts}, nil
}

// ---------- posts ----------

func (s *Server) CreatePost(ctx context.Context, req *pb.CreatePostRequest) (*pb.Post, error) {
	ctx = s.prepareCtx(ctx)
	if req.TenantId == "" || req.ThreadId == "" || req.AuthorId == "" || req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id, thread_id, author_id, content required")
	}
	var parent *string
	if req.ParentPostId != "" {
		p := req.ParentPostId
		parent = &p
	}
	return s.createPost(ctx, req.TenantId, req.ThreadId, req.AuthorId, req.Content, parent)
}

func (s *Server) createPost(ctx context.Context, tenantID, threadID, authorID, content string, parent *string) (*pb.Post, error) {
	s.tagSpan(ctx,
		attribute.String("tenant.id", tenantID),
		attribute.String("thread.id", threadID),
	)

	// 1) parse @usernames, 2) resolve, 3) determine author role, 4) persist.
	usernames := mentions.Parse(content)
	userMap, err := s.userRes.ResolveUsernames(ctx, tenantID, usernames)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("resolve mentions: %v", err))
	}
	mentionedIDs := make([]string, 0, len(userMap))
	for _, id := range userMap {
		mentionedIDs = append(mentionedIDs, id)
	}

	// Fetch course for role lookup — done via thread for tenant safety.
	t, err := s.store.GetThread(ctx, tenantID, threadID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("load thread: %v", err))
	}
	role, err := s.roleRes.AuthorRole(ctx, tenantID, t.CourseID, authorID)
	if err != nil {
		role = "student"
	}

	res, err := s.store.CreatePost(ctx, tenantID, threadID, authorID, role, content, parent, mentionedIDs)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "thread not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("create post: %v", err))
	}

	// Emit one Kafka event per mention. We use a best-effort loop — a broker
	// outage does not fail the RPC; the mention row is durable.
	excerpt := truncateExcerpt(content)
	for _, m := range res.Mentions {
		_ = s.producer.PublishMention(ctx, kafka.MentionEvent{
			TenantID:      tenantID,
			ThreadID:      threadID,
			ThreadTitle:   t.Title,
			MentionedBy:   authorID,
			MentionedUser: m.MentionedUserID,
			Excerpt:       excerpt,
			PostID:        res.Post.ID,
			CourseID:      res.CourseID,
			// MentionedEmail is intentionally left empty — email-service
			// resolves via user-auth GetUser. Populating it here would cost a
			// user-auth roundtrip per mention inside the write path.
		})
	}

	return postToProto(&res.Post, mentionedIDs), nil
}

// ---------- inbox ----------

func (s *Server) GetInbox(ctx context.Context, req *pb.GetInboxRequest) (*pb.InboxResponse, error) {
	ctx = s.prepareCtx(ctx)
	if req.TenantId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and user_id required")
	}
	s.tagSpan(ctx, attribute.String("tenant.id", req.TenantId))
	items, next, hasMore, err := s.store.GetInbox(ctx, req.TenantId, req.UserId, int(req.Limit), req.Cursor, req.UnseenOnly)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get inbox: %v", err))
	}
	out := make([]*pb.InboxItem, 0, len(items))
	for i := range items {
		out = append(out, inboxToProto(&items[i]))
	}
	return &pb.InboxResponse{Items: out, NextCursor: next, HasMore: hasMore}, nil
}

func (s *Server) MarkInboxRead(ctx context.Context, req *pb.MarkReadRequest) (*emptypb.Empty, error) {
	ctx = s.prepareCtx(ctx)
	if req.TenantId == "" || req.UserId == "" || req.InboxItemId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id, user_id, inbox_item_id required")
	}
	s.tagSpan(ctx, attribute.String("tenant.id", req.TenantId))
	if err := s.store.MarkInboxRead(ctx, req.TenantId, req.UserId, req.InboxItemId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "inbox item not found")
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("mark read: %v", err))
	}
	return &emptypb.Empty{}, nil
}

// ---------- instructor queue ----------

func (s *Server) GetThreadsNeedingReply(ctx context.Context, req *pb.ListNeedingReplyRequest) (*pb.ListThreadsResponse, error) {
	ctx = s.prepareCtx(ctx)
	if req.TenantId == "" || req.InstructorId == "" {
		return nil, status.Error(codes.InvalidArgument, "tenant_id and instructor_id required")
	}
	s.tagSpan(ctx,
		attribute.String("tenant.id", req.TenantId),
		attribute.String("instructor.id", req.InstructorId),
	)

	// Canonical source of truth for "which courses does this instructor teach":
	// the local instructor_courses table. We intersect the caller-provided list
	// (if any) to allow filtering to a subset in the UI.
	known, err := s.store.InstructorCoursesFor(ctx, req.TenantId, req.InstructorId)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("courses: %v", err))
	}
	courseIDs := intersect(known, req.CourseIds)
	if len(courseIDs) == 0 {
		return &pb.ListThreadsResponse{}, nil
	}

	items, err := s.store.ListThreadsNeedingReply(ctx, req.TenantId, courseIDs, int(req.Limit), s.now())
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("needing reply: %v", err))
	}
	out := make([]*pb.Thread, 0, len(items))
	for i := range items {
		out = append(out, threadToProto(&items[i]))
	}
	return &pb.ListThreadsResponse{Threads: out}, nil
}

// intersect returns the intersection of a (the authoritative list) with b.
// If b is empty, it returns all of a.
func intersect(a, b []string) []string {
	if len(b) == 0 {
		return a
	}
	set := make(map[string]struct{}, len(a))
	for _, x := range a {
		set[x] = struct{}{}
	}
	out := make([]string, 0, len(b))
	for _, x := range b {
		if _, ok := set[x]; ok {
			out = append(out, x)
		}
	}
	return out
}

// truncateExcerpt returns a short snippet of the post body suitable for email
// previews. Max 280 chars, trimmed on a word boundary when possible.
func truncateExcerpt(s string) string {
	const maxLen = 280
	if len(s) <= maxLen {
		return s
	}
	cut := s[:maxLen]
	if idx := lastSpace(cut); idx > maxLen/2 {
		cut = cut[:idx]
	}
	return cut + "…"
}

func lastSpace(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ' ' || s[i] == '\n' || s[i] == '\t' {
			return i
		}
	}
	return -1
}

// ---------- health ----------

func (s *Server) HealthCheck(_ context.Context, _ *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{Status: "ok"}, nil
}

// ---------- proto helpers ----------

func threadToProto(t *models.Thread) *pb.Thread {
	return &pb.Thread{
		Id:             t.ID,
		CourseId:       t.CourseID,
		Title:          t.Title,
		CreatedBy:      t.CreatedBy,
		CreatedAt:      timestamppb.New(t.CreatedAt),
		LastActivityAt: timestamppb.New(t.LastActivityAt),
		ReplyCount:     int32(t.ReplyCount),
	}
}

func postToProto(p *models.Post, mentionedIDs []string) *pb.Post {
	out := &pb.Post{
		Id:               p.ID,
		ThreadId:         p.ThreadID,
		AuthorId:         p.AuthorID,
		Content:          p.Content,
		CreatedAt:        timestamppb.New(p.CreatedAt),
		MentionedUserIds: mentionedIDs,
	}
	if p.ParentPostID != nil {
		out.ParentPostId = *p.ParentPostID
	}
	if p.EditedAt != nil {
		out.EditedAt = timestamppb.New(*p.EditedAt)
	}
	return out
}

func inboxToProto(i *models.InboxItem) *pb.InboxItem {
	out := &pb.InboxItem{
		Id:          i.ID,
		UserId:      i.UserID,
		Type:        i.Type,
		ReferenceId: i.ReferenceID,
		CreatedAt:   timestamppb.New(i.CreatedAt),
	}
	if i.SeenAt != nil {
		out.SeenAt = timestamppb.New(*i.SeenAt)
	}
	return out
}
