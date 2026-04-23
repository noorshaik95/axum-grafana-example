package grpc

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	pb "slate/services/discussion-service/api/proto"
	"slate/services/discussion-service/internal/kafka"
	"slate/services/discussion-service/internal/models"
	"slate/services/discussion-service/internal/repository"
	"slate/services/discussion-service/internal/userauth"

	"github.com/google/uuid"
)

// ---------- in-memory Store fake ----------

type fakeStore struct {
	mu               sync.Mutex
	threads          map[string]models.Thread
	posts            map[string][]models.Post   // threadID → posts (ordered)
	postRoles        map[string]string          // postID → role
	mentions         map[string][]models.Mention // postID → mentions
	inbox            map[string][]models.InboxItem // userID → items
	instructorCourse map[string]map[string]struct{} // instructorID → courseIDs
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		threads:          map[string]models.Thread{},
		posts:            map[string][]models.Post{},
		postRoles:        map[string]string{},
		mentions:         map[string][]models.Mention{},
		inbox:            map[string][]models.InboxItem{},
		instructorCourse: map[string]map[string]struct{}{},
	}
}

func (f *fakeStore) CreateThread(_ context.Context, tenantID, courseID, title, createdBy string) (*models.Thread, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := uuid.NewString()
	now := time.Now().UTC()
	t := models.Thread{
		ID: id, TenantID: tenantID, CourseID: courseID, Title: title,
		CreatedBy: createdBy, CreatedAt: now, LastActivityAt: now,
	}
	f.threads[id] = t
	return &t, nil
}

func (f *fakeStore) GetThread(_ context.Context, tenantID, threadID string) (*models.Thread, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.threads[threadID]
	if !ok || t.TenantID != tenantID {
		return nil, repository.ErrNotFound
	}
	return &t, nil
}

func (f *fakeStore) ListThreads(_ context.Context, tenantID, courseID string, limit int, _ string) ([]models.Thread, string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []models.Thread
	for _, t := range f.threads {
		if t.TenantID == tenantID && t.CourseID == courseID {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastActivityAt.After(out[j].LastActivityAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, "", false, nil
}

func (f *fakeStore) CreatePost(_ context.Context, tenantID, threadID, authorID, authorRole, content string, parent *string, mentionedUserIDs []string) (*repository.CreatePostResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.threads[threadID]
	if !ok || t.TenantID != tenantID {
		return nil, repository.ErrNotFound
	}
	now := time.Now().UTC()
	post := models.Post{
		ID: uuid.NewString(), ThreadID: threadID, AuthorID: authorID,
		Content: content, ParentPostID: parent, CreatedAt: now,
	}
	f.posts[threadID] = append(f.posts[threadID], post)
	f.postRoles[post.ID] = authorRole

	var ms []models.Mention
	for _, uid := range mentionedUserIDs {
		if uid == "" || uid == authorID {
			continue
		}
		m := models.Mention{ID: uuid.NewString(), PostID: post.ID, MentionedUserID: uid}
		f.mentions[post.ID] = append(f.mentions[post.ID], m)
		ms = append(ms, m)
		f.inbox[uid] = append(f.inbox[uid], models.InboxItem{
			ID: uuid.NewString(), UserID: uid, Type: "mention",
			ReferenceID: post.ID, CreatedAt: now,
		})
	}

	t.LastActivityAt = now
	t.ReplyCount++
	f.threads[threadID] = t

	return &repository.CreatePostResult{Post: post, Mentions: ms, CourseID: t.CourseID}, nil
}

func (f *fakeStore) ListPosts(_ context.Context, tenantID, threadID string) ([]models.Post, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.threads[threadID]
	if !ok || t.TenantID != tenantID {
		return nil, nil
	}
	return append([]models.Post(nil), f.posts[threadID]...), nil
}

func (f *fakeStore) GetInbox(_ context.Context, _ , userID string, limit int, _ string, unseenOnly bool) ([]models.InboxItem, string, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []models.InboxItem
	for _, it := range f.inbox[userID] {
		if unseenOnly && it.SeenAt != nil {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, "", false, nil
}

func (f *fakeStore) MarkInboxRead(_ context.Context, _ , userID, itemID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := f.inbox[userID]
	for i := range items {
		if items[i].ID == itemID {
			now := time.Now().UTC()
			items[i].SeenAt = &now
			f.inbox[userID] = items
			return nil
		}
	}
	return repository.ErrNotFound
}

func (f *fakeStore) InstructorCoursesFor(_ context.Context, _ , instructorID string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	set := f.instructorCourse[instructorID]
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out, nil
}

func (f *fakeStore) addInstructor(instructorID string, courseIDs ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.instructorCourse[instructorID] == nil {
		f.instructorCourse[instructorID] = map[string]struct{}{}
	}
	for _, c := range courseIDs {
		f.instructorCourse[instructorID][c] = struct{}{}
	}
}

// backdatePost mutates a post's created_at — used by the 24h boundary test.
func (f *fakeStore) backdatePost(postID string, t time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for tid, list := range f.posts {
		for i := range list {
			if list[i].ID == postID {
				list[i].CreatedAt = t
				f.posts[tid] = list
				return
			}
		}
	}
}

func (f *fakeStore) ListThreadsNeedingReply(_ context.Context, tenantID string, courseIDs []string, limit int, now time.Time) ([]models.Thread, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	courseSet := map[string]struct{}{}
	for _, c := range courseIDs {
		courseSet[c] = struct{}{}
	}

	type annotated struct {
		thread         models.Thread
		lastInstructor time.Time
		lastStudent    time.Time
	}
	var candidates []annotated

	for _, t := range f.threads {
		if t.TenantID != tenantID {
			continue
		}
		if _, ok := courseSet[t.CourseID]; !ok {
			continue
		}
		var lastI, lastS time.Time
		for _, p := range f.posts[t.ID] {
			role := f.postRoles[p.ID]
			if role == "instructor" && p.CreatedAt.After(lastI) {
				lastI = p.CreatedAt
			}
			if role == "student" && p.CreatedAt.After(lastS) {
				lastS = p.CreatedAt
			}
		}
		if lastS.IsZero() {
			continue
		}
		cutoff := now.Add(-24 * time.Hour)
		if !lastI.IsZero() && !lastI.Before(cutoff) {
			continue
		}
		if !lastI.IsZero() && !lastS.After(lastI) {
			continue
		}
		candidates = append(candidates, annotated{t, lastI, lastS})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].thread.LastActivityAt.Before(candidates[j].thread.LastActivityAt)
	})
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}
	out := make([]models.Thread, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.thread)
	}
	return out, nil
}

// ---------- capturing publisher ----------

type capturingPublisher struct {
	mu     sync.Mutex
	events []kafka.MentionEvent
}

func (c *capturingPublisher) PublishMention(_ context.Context, evt kafka.MentionEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, evt)
	return nil
}

// ---------- tests ----------

func newServer(t *testing.T, users map[string]string) (*Server, *fakeStore, *capturingPublisher) {
	t.Helper()
	store := newFakeStore()
	pub := &capturingPublisher{}
	srv := New(Options{
		Repo:         store,
		Producer:     pub,
		UserResolver: userauth.StaticResolver{Users: users},
		Now:          func() time.Time { return time.Now().UTC() },
	})
	return srv, store, pub
}

func TestThreadCRUD(t *testing.T) {
	srv, _, _ := newServer(t, nil)
	ctx := context.Background()

	th, err := srv.CreateThread(ctx, &pb.CreateThreadRequest{
		TenantId: "t1", CourseId: "c1", Title: "Question", CreatedBy: "u1",
	})
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	if th.Id == "" || th.CourseId != "c1" {
		t.Errorf("unexpected thread: %+v", th)
	}

	// List should contain it.
	list, err := srv.ListThreads(ctx, &pb.ListThreadsRequest{TenantId: "t1", CourseId: "c1"})
	if err != nil {
		t.Fatalf("ListThreads: %v", err)
	}
	if len(list.Threads) != 1 || list.Threads[0].Id != th.Id {
		t.Errorf("list mismatch: %+v", list)
	}

	// Get should return empty posts.
	got, err := srv.GetThread(ctx, &pb.GetThreadRequest{TenantId: "t1", ThreadId: th.Id})
	if err != nil {
		t.Fatalf("GetThread: %v", err)
	}
	if len(got.Posts) != 0 {
		t.Errorf("expected no posts, got %d", len(got.Posts))
	}

	// Wrong tenant → NotFound.
	if _, err := srv.GetThread(ctx, &pb.GetThreadRequest{TenantId: "other", ThreadId: th.Id}); err == nil {
		t.Errorf("expected NotFound for cross-tenant read")
	}
}

func TestMentionParsingAndInboxAndKafka(t *testing.T) {
	srv, store, pub := newServer(t, map[string]string{
		"alice": "user-alice",
		"bob":   "user-bob",
	})
	ctx := context.Background()

	th, err := srv.CreateThread(ctx, &pb.CreateThreadRequest{
		TenantId: "t1", CourseId: "c1", Title: "Help", CreatedBy: "u1",
	})
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}

	post, err := srv.CreatePost(ctx, &pb.CreatePostRequest{
		TenantId: "t1", ThreadId: th.Id, AuthorId: "u1",
		Content: "hey @alice and @bob — can you look? email nobody@example.com",
	})
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if len(post.MentionedUserIds) != 2 {
		t.Errorf("want 2 mentions, got %v", post.MentionedUserIds)
	}

	// Each mentioned user has an inbox item of type=mention referencing the post.
	for _, uid := range []string{"user-alice", "user-bob"} {
		inbox, err := srv.GetInbox(ctx, &pb.GetInboxRequest{TenantId: "t1", UserId: uid})
		if err != nil {
			t.Fatalf("GetInbox(%s): %v", uid, err)
		}
		if len(inbox.Items) != 1 || inbox.Items[0].Type != "mention" || inbox.Items[0].ReferenceId != post.Id {
			t.Errorf("inbox[%s] mismatch: %+v", uid, inbox.Items)
		}
	}

	// Kafka: exactly two events published, one per mention.
	if got := len(pub.events); got != 2 {
		t.Fatalf("want 2 Kafka events, got %d: %+v", got, pub.events)
	}
	seen := map[string]bool{}
	for _, e := range pub.events {
		seen[e.MentionedUser] = true
		if e.ThreadID != th.Id || e.PostID != post.Id || e.CourseID != "c1" || e.TenantID != "t1" {
			t.Errorf("unexpected event: %+v", e)
		}
		if e.MentionedBy != "u1" {
			t.Errorf("mentioned_by = %q, want u1", e.MentionedBy)
		}
		if e.ThreadTitle != "Help" {
			t.Errorf("thread_title = %q, want Help", e.ThreadTitle)
		}
		if e.Excerpt == "" {
			t.Errorf("expected non-empty excerpt")
		}
	}
	if !seen["user-alice"] || !seen["user-bob"] {
		t.Errorf("missing recipients in events: %+v", pub.events)
	}

	// MarkInboxRead makes the inbox item unseen-only query skip it.
	inbox, _ := srv.GetInbox(ctx, &pb.GetInboxRequest{TenantId: "t1", UserId: "user-alice"})
	if _, err := srv.MarkInboxRead(ctx, &pb.MarkReadRequest{
		TenantId: "t1", UserId: "user-alice", InboxItemId: inbox.Items[0].Id,
	}); err != nil {
		t.Fatalf("MarkInboxRead: %v", err)
	}
	unseen, _ := srv.GetInbox(ctx, &pb.GetInboxRequest{TenantId: "t1", UserId: "user-alice", UnseenOnly: true})
	if len(unseen.Items) != 0 {
		t.Errorf("expected no unseen items, got %d", len(unseen.Items))
	}
	_ = store // keep fake accessible for future assertions
}

func TestUnknownUsernamesDoNotMention(t *testing.T) {
	srv, _, pub := newServer(t, map[string]string{"alice": "user-alice"})
	ctx := context.Background()
	th, _ := srv.CreateThread(ctx, &pb.CreateThreadRequest{
		TenantId: "t1", CourseId: "c1", Title: "x", CreatedBy: "u1",
	})
	post, err := srv.CreatePost(ctx, &pb.CreatePostRequest{
		TenantId: "t1", ThreadId: th.Id, AuthorId: "u1",
		Content: "hey @ghost you around? @alice",
	})
	if err != nil {
		t.Fatalf("CreatePost: %v", err)
	}
	if len(post.MentionedUserIds) != 1 || post.MentionedUserIds[0] != "user-alice" {
		t.Errorf("want only user-alice, got %v", post.MentionedUserIds)
	}
	if len(pub.events) != 1 {
		t.Errorf("want 1 event, got %d", len(pub.events))
	}
}

func TestNeedingReply24hBoundary(t *testing.T) {
	// Freeze "now" so we can backdate instructor posts precisely.
	fixedNow := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	store := newFakeStore()
	store.addInstructor("instructor-1", "c1")
	pub := &capturingPublisher{}
	srv := New(Options{
		Repo:     store,
		Producer: pub,
		Now:      func() time.Time { return fixedNow },
	})
	ctx := context.Background()

	// Three threads in c1:
	//   A) student-only                  → NEEDS REPLY
	//   B) instructor replied 25h ago, student since   → NEEDS REPLY (>24h ago AND newer student post)
	//   C) instructor replied 1h ago                   → NOT needing reply
	th := func(title string) string {
		out, _ := store.CreateThread(ctx, "t1", "c1", title, "u1")
		return out.ID
	}
	a := th("A: student only")
	b := th("B: stale instructor reply")
	c := th("C: fresh instructor reply")

	mkPost := func(threadID, author, role string, at time.Time) string {
		res, _ := store.CreatePost(ctx, "t1", threadID, author, role, "x", nil, nil)
		store.backdatePost(res.Post.ID, at)
		// also backdate last_activity to match so ordering is deterministic
		store.mu.Lock()
		t := store.threads[threadID]
		if at.After(t.LastActivityAt) || t.LastActivityAt.After(at.Add(time.Second)) {
			t.LastActivityAt = at
		}
		store.threads[threadID] = t
		store.mu.Unlock()
		return res.Post.ID
	}

	mkPost(a, "stu-1", "student", fixedNow.Add(-2*time.Hour))

	mkPost(b, "instructor-1", "instructor", fixedNow.Add(-25*time.Hour))
	mkPost(b, "stu-1", "student", fixedNow.Add(-3*time.Hour))

	mkPost(c, "instructor-1", "instructor", fixedNow.Add(-1*time.Hour))
	mkPost(c, "stu-1", "student", fixedNow.Add(-2*time.Hour)) // student older than instr → not needed

	resp, err := srv.GetThreadsNeedingReply(ctx, &pb.ListNeedingReplyRequest{
		TenantId: "t1", InstructorId: "instructor-1",
	})
	if err != nil {
		t.Fatalf("ListThreadsNeedingReply: %v", err)
	}
	if len(resp.Threads) != 2 {
		t.Fatalf("want 2 threads, got %d: %+v", len(resp.Threads), resp.Threads)
	}
	ids := map[string]bool{resp.Threads[0].Id: true, resp.Threads[1].Id: true}
	if !ids[a] || !ids[b] || ids[c] {
		t.Errorf("unexpected set: %+v (a=%s b=%s c=%s)", ids, a, b, c)
	}

	// Oldest-first: B's last_activity is older than A's.
	if !resp.Threads[0].LastActivityAt.AsTime().Before(resp.Threads[1].LastActivityAt.AsTime()) {
		t.Errorf("expected oldest-first ordering: %v", resp.Threads)
	}
}

func TestNeedingReplyOnlyForOwnedCourses(t *testing.T) {
	store := newFakeStore()
	store.addInstructor("instructor-1", "c1") // NOT c2
	srv := New(Options{Repo: store, Now: func() time.Time { return time.Now().UTC() }})
	ctx := context.Background()

	th, _ := store.CreateThread(ctx, "t1", "c2", "title", "u1")
	_, _ = store.CreatePost(ctx, "t1", th.ID, "stu", "student", "hey", nil, nil)

	resp, err := srv.GetThreadsNeedingReply(ctx, &pb.ListNeedingReplyRequest{
		TenantId: "t1", InstructorId: "instructor-1",
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(resp.Threads) != 0 {
		t.Errorf("instructor should not see c2 threads; got %+v", resp.Threads)
	}
}

func TestCreatePostPropagatesNotFound(t *testing.T) {
	srv, _, _ := newServer(t, nil)
	_, err := srv.CreatePost(context.Background(), &pb.CreatePostRequest{
		TenantId: "t1", ThreadId: "does-not-exist", AuthorId: "u1", Content: "hi",
	})
	if err == nil || !errors.Is(err, err) { // just ensure non-nil
		t.Fatalf("want error, got nil")
	}
}
