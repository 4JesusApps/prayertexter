//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/service"
)

// SentMessage records a single SMS that was dispatched to the
// RecordingMessageSender during a test.
type SentMessage struct {
	To   string
	Body string
}

// RecordingMessageSender is the MessageSender implementation used in integration
// tests. It captures every send for assertion and supports targeted failure
// injection so rollback paths can be exercised.
type RecordingMessageSender struct {
	mu       sync.Mutex
	messages []SentMessage
	// failOnce[phone] = error to return on the next send to that phone; consumed
	// once and then removed.
	failOnce map[string]error
}

func newRecordingSender() *RecordingMessageSender {
	return &RecordingMessageSender{failOnce: map[string]error{}}
}

// SendMessage records the call. If FailNextSendTo was previously invoked for
// the destination phone, the recorded failure is returned and consumed.
func (r *RecordingMessageSender) SendMessage(_ context.Context, to, body string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err, ok := r.failOnce[to]; ok {
		delete(r.failOnce, to)
		r.messages = append(r.messages, SentMessage{To: to, Body: body})
		return err
	}
	r.messages = append(r.messages, SentMessage{To: to, Body: body})
	return nil
}

// FailNextSendTo makes the next SendMessage call to `phone` return err.
// Used to exercise rollback paths in PrayerService.AssignPrayer and similar.
func (r *RecordingMessageSender) FailNextSendTo(phone string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failOnce[phone] = err
}

// Messages returns a copy of the recorded sends.
func (r *RecordingMessageSender) Messages() []SentMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]SentMessage, len(r.messages))
	copy(out, r.messages)
	return out
}

// Reset clears the recorded sends and any pending failure injections.
func (r *RecordingMessageSender) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages = nil
	r.failOnce = map[string]error{}
}

// AssertSentTo fails the test unless a message was sent to `phone` whose body
// contains `bodyContains`.
func (r *RecordingMessageSender) AssertSentTo(t *testing.T, phone, bodyContains string) {
	t.Helper()
	for _, m := range r.Messages() {
		if m.To == phone && strings.Contains(m.Body, bodyContains) {
			return
		}
	}
	t.Fatalf("expected SMS to %s containing %q; got: %v", phone, bodyContains, r.Messages())
}

// AssertNotSentTo fails the test if any message was sent to `phone`.
func (r *RecordingMessageSender) AssertNotSentTo(t *testing.T, phone string) {
	t.Helper()
	for _, m := range r.Messages() {
		if m.To == phone {
			t.Fatalf("expected no SMS to %s; got: %v", phone, m)
		}
	}
}

// CountTo returns the number of recorded sends to the given phone.
func (r *RecordingMessageSender) CountTo(phone string) int {
	n := 0
	for _, m := range r.Messages() {
		if m.To == phone {
			n++
		}
	}
	return n
}

// testEnv bundles the per-test infrastructure: a clean DynamoDB state, real
// repositories wired to the container, a real Router with all real services,
// and the recording sender for outbound SMS assertions.
type testEnv struct {
	t            *testing.T
	ctx          context.Context
	cfg          config.Config
	router       *service.Router
	prayerSvc    *service.PrayerService
	members      repository.MemberRepository
	prayers      repository.PrayerRepository
	blocked      repository.BlockedPhonesRepository
	intercessors repository.IntercessorPhonesRepository
	sender       *RecordingMessageSender
}

// newTestEnv builds a fresh integration environment for a single test. It
// truncates all tables before returning so tests are isolated.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	resetTables(t)

	cfg := config.Config{
		IntercessorsPerPrayer: 2,
		PrayerReminderHours:   3,
	}

	const ddbTimeout = 5
	members := repository.NewMemberRepository(ddbClient, memberTable, ddbTimeout)
	prayers := repository.NewPrayerRepository(ddbClient, activePrayerTable, queuedPrayerTable, ddbTimeout)
	blocked := repository.NewBlockedPhonesRepository(ddbClient, generalTable, ddbTimeout)
	intercessors := repository.NewIntercessorPhonesRepository(ddbClient, generalTable, ddbTimeout)

	sender := newRecordingSender()

	memberSvc := service.NewMemberService(members, intercessors, prayers, sender, cfg)
	prayerSvc := service.NewPrayerService(members, intercessors, prayers, sender, cfg)
	adminSvc := service.NewAdminService(members, blocked, sender, memberSvc)
	router := service.NewRouter(members, blocked, memberSvc, prayerSvc, adminSvc)

	return &testEnv{
		t:            t,
		ctx:          context.Background(),
		cfg:          cfg,
		router:       router,
		prayerSvc:    prayerSvc,
		members:      members,
		prayers:      prayers,
		blocked:      blocked,
		intercessors: intercessors,
		sender:       sender,
	}
}

// msg is a shorthand for building a domain.TextMessage in test bodies.
func msg(phone, body string) domain.TextMessage {
	return domain.TextMessage{Phone: phone, Body: body}
}

// handle is a shorthand for router.Handle that fails the test on error.
func (e *testEnv) handle(text domain.TextMessage) {
	e.t.Helper()
	if err := e.router.Handle(e.ctx, text); err != nil {
		e.t.Fatalf("router.Handle(%+v): %v", text, err)
	}
}

// handleExpectErr runs router.Handle and returns the error without failing the
// test. Used for rollback scenarios that intentionally trigger a failure.
func (e *testEnv) handleExpectErr(text domain.TextMessage) error {
	return e.router.Handle(e.ctx, text)
}

// seedIntercessor creates a fully signed-up intercessor member and adds the
// phone to the IntercessorPhones list.
func (e *testEnv) seedIntercessor(phone, name string, weeklyLimit int) {
	e.t.Helper()
	mem := &domain.Member{
		Phone:             phone,
		Name:              name,
		Intercessor:       true,
		SetupStatus:       domain.MemberSetupComplete,
		SetupStage:        domain.MemberSignUpStepFinal,
		WeeklyPrayerLimit: weeklyLimit,
		WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
	}
	if err := e.members.Save(e.ctx, mem); err != nil {
		e.t.Fatalf("seed intercessor %s: %v", phone, err)
	}
	phones, err := e.intercessors.Get(e.ctx)
	if err != nil {
		e.t.Fatalf("seed intercessor list: %v", err)
	}
	phones.AddPhone(phone)
	if err := e.intercessors.Save(e.ctx, phones); err != nil {
		e.t.Fatalf("seed intercessor list save: %v", err)
	}
}

// seedRequestor creates a signed-up prayer-only member (not an intercessor).
func (e *testEnv) seedRequestor(phone, name string) {
	e.t.Helper()
	mem := &domain.Member{
		Phone:       phone,
		Name:        name,
		Intercessor: false,
		SetupStatus: domain.MemberSetupComplete,
		SetupStage:  domain.MemberSignUpStepFinal,
	}
	if err := e.members.Save(e.ctx, mem); err != nil {
		e.t.Fatalf("seed requestor %s: %v", phone, err)
	}
}

// seedBlocked appends phones to the BlockedPhones list.
func (e *testEnv) seedBlocked(phones ...string) {
	e.t.Helper()
	bp, err := e.blocked.Get(e.ctx)
	if err != nil {
		e.t.Fatalf("seed blocked get: %v", err)
	}
	for _, p := range phones {
		bp.AddPhone(p)
	}
	if err := e.blocked.Save(e.ctx, bp); err != nil {
		e.t.Fatalf("seed blocked save: %v", err)
	}
}

// requireMember loads a member or fails the test.
func (e *testEnv) requireMember(phone string) *domain.Member {
	e.t.Helper()
	mem, err := e.members.Get(e.ctx, phone)
	if err != nil {
		e.t.Fatalf("get member %s: %v", phone, err)
	}
	return mem
}

// requireNoMember fails unless the member is absent.
func (e *testEnv) requireNoMember(phone string) {
	e.t.Helper()
	_, err := e.members.Get(e.ctx, phone)
	if !errors.Is(err, repository.ErrNotFound) {
		e.t.Fatalf("expected member %s to be absent; got err=%v", phone, err)
	}
}

// requireNoActivePrayer fails unless the active prayer for the given
// intercessor phone is absent.
func (e *testEnv) requireNoActivePrayer(intercessorPhone string) {
	e.t.Helper()
	_, err := e.prayers.Get(e.ctx, intercessorPhone, false)
	if !errors.Is(err, repository.ErrNotFound) {
		e.t.Fatalf("expected no active prayer for %s; got err=%v", intercessorPhone, err)
	}
}

// requireActivePrayer loads an active prayer or fails the test.
func (e *testEnv) requireActivePrayer(intercessorPhone string) *domain.Prayer {
	e.t.Helper()
	p, err := e.prayers.Get(e.ctx, intercessorPhone, false)
	if err != nil {
		e.t.Fatalf("get active prayer %s: %v", intercessorPhone, err)
	}
	return p
}

// requireQueuedCount fails unless the queued prayer table contains exactly n entries.
func (e *testEnv) requireQueuedCount(n int) []domain.Prayer {
	e.t.Helper()
	queued, err := e.prayers.GetAll(e.ctx, true)
	if err != nil {
		e.t.Fatalf("list queued: %v", err)
	}
	if len(queued) != n {
		e.t.Fatalf("expected %d queued prayers; got %d: %+v", n, len(queued), queued)
	}
	return queued
}

// errSendFailed is a generic error injected into RecordingMessageSender for
// rollback path tests.
var errSendFailed = errors.New("injected send failure")

// dump is a convenience for diagnostics when a test fails mid-flow.
func (e *testEnv) dump() string {
	var b strings.Builder
	fmt.Fprintln(&b, "messages:")
	for _, m := range e.sender.Messages() {
		fmt.Fprintf(&b, "  -> %s: %s\n", m.To, m.Body)
	}
	return b.String()
}
