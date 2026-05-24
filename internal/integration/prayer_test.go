//go:build integration

package integration_test

import (
	"strings"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
)

// TestPrayer_AssignedToAvailableIntercessors covers the happy path: requestor
// sends a prayer, the router finds two available intercessors, both get the
// prayer, and the requestor gets the assigned confirmation.
func TestPrayer_AssignedToAvailableIntercessors(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000001"
	const intr1 = "+12000000011"
	const intr2 = "+12000000012"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr1, "Ivy", 5)
	env.seedIntercessor(intr2, "Ian", 5)

	env.handle(msg(requestor, "please pray for my mother's surgery"))

	env.sender.AssertSentTo(t, intr1, "please pray for my mother's surgery")
	env.sender.AssertSentTo(t, intr2, "please pray for my mother's surgery")
	env.sender.AssertSentTo(t, requestor, messaging.MsgPrayerAssigned)

	// Both intercessors should have an active prayer keyed by their phone.
	p1 := env.requireActivePrayer(intr1)
	p2 := env.requireActivePrayer(intr2)
	if !strings.Contains(p1.Request, "mother's surgery") {
		t.Fatalf("intr1 prayer request unexpected: %q", p1.Request)
	}
	if !strings.Contains(p2.Request, "mother's surgery") {
		t.Fatalf("intr2 prayer request unexpected: %q", p2.Request)
	}

	// Reservation increments PrayerCount on each intercessor.
	mem1 := env.requireMember(intr1)
	mem2 := env.requireMember(intr2)
	if mem1.PrayerCount != 1 {
		t.Fatalf("expected intr1 PrayerCount=1, got %d", mem1.PrayerCount)
	}
	if mem2.PrayerCount != 1 {
		t.Fatalf("expected intr2 PrayerCount=1, got %d", mem2.PrayerCount)
	}
}

// TestPrayer_NoIntercessorsQueued verifies the queueing branch: with no
// intercessors available the prayer is saved to QueuedPrayer and the requestor
// gets MsgPrayerQueued.
func TestPrayer_NoIntercessorsQueued(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000002"

	env.seedRequestor(requestor, "Sam")

	env.handle(msg(requestor, "please pray for my upcoming job interview"))
	env.sender.AssertSentTo(t, requestor, messaging.MsgPrayerQueued)

	queued := env.requireQueuedCount(1)
	if !strings.Contains(queued[0].Request, "job interview") {
		t.Fatalf("queued prayer text unexpected: %q", queued[0].Request)
	}
	if queued[0].Requestor.Phone != requestor {
		t.Fatalf("queued prayer requestor unexpected: %q", queued[0].Requestor.Phone)
	}
}

// TestPrayer_ProfanityRejected verifies that a prayer containing profanity
// triggers ProfanityDetectedTmpl and creates no DB state.
func TestPrayer_ProfanityRejected(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000003"
	const intr = "+12000000031"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(requestor, "please pray for this fuck of a situation"))

	// Body contains the flagged word per ProfanityDetectedTmpl.
	env.sender.AssertSentTo(t, requestor, "fuck")
	env.sender.AssertNotSentTo(t, intr)

	env.requireNoActivePrayer(intr)
	env.requireQueuedCount(0)
}

// TestPrayer_InvalidRequestTooFewWords verifies that prayers with fewer than
// 5 words trigger MsgInvalidRequest and create no DB state.
func TestPrayer_InvalidRequestTooFewWords(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000004"
	const intr = "+12000000041"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(requestor, "pray help"))

	env.sender.AssertSentTo(t, requestor, messaging.MsgInvalidRequest)
	env.sender.AssertNotSentTo(t, intr)
	env.requireNoActivePrayer(intr)
}

// TestPrayer_AnonTriggerMasksRequestor verifies that "#anon" in the request body
// masks the requestor's name to "Anonymous" on the persisted ActivePrayer and
// the SMS sent to the intercessor.
func TestPrayer_AnonTriggerMasksRequestor(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000005"
	const intr = "+12000000051"

	env.seedRequestor(requestor, "RealName")
	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(requestor, "#anon please pray for my health and family"))

	// Intercessor SMS uses the PrayerIntroTmpl which inlines the requestor name.
	env.sender.AssertSentTo(t, intr, "Anonymous")
	env.sender.AssertSentTo(t, requestor, messaging.MsgPrayerAssigned)

	pryr := env.requireActivePrayer(intr)
	if pryr.Requestor.Name != "Anonymous" {
		t.Fatalf("expected persisted Requestor.Name=Anonymous, got %q", pryr.Requestor.Name)
	}
	// The literal "#anon" must be stripped from the persisted request body.
	if strings.Contains(strings.ToLower(pryr.Request), "#anon") {
		t.Fatalf("expected '#anon' to be stripped; got %q", pryr.Request)
	}
}

// TestPrayer_CompleteFullLifecycle covers the request -> complete cycle:
// requestor sends prayer, intercessor sends "prayed", thank-you + confirmation
// SMS go out, active prayer is removed.
func TestPrayer_CompleteFullLifecycle(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000006"
	const intr = "+12000000061"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(requestor, "please pray for safe travels this weekend"))
	env.requireActivePrayer(intr)

	env.handle(msg(intr, "prayed"))

	env.sender.AssertSentTo(t, intr, messaging.MsgPrayerThankYou)
	// PrayerConfirmationTmpl uses {{.Name}} which is the intercessor's name "Ivy".
	env.sender.AssertSentTo(t, requestor, "Ivy")

	env.requireNoActivePrayer(intr)
}

// TestPrayer_CompleteWhenNoActivePrayer verifies the empty-prayer case: an
// intercessor who sends "prayed" with no active prayer gets MsgNoActivePrayer.
func TestPrayer_CompleteWhenNoActivePrayer(t *testing.T) {
	env := newTestEnv(t)
	const intr = "+12000000007"

	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(intr, "prayed"))
	env.sender.AssertSentTo(t, intr, messaging.MsgNoActivePrayer)
}

// seedMaxedIntercessor seeds an intercessor whose PrayerCount is already at
// the WeeklyPrayerLimit and whose WeeklyPrayerDate is recent (within the 7-day
// reset window) -- so they are NOT eligible for new prayers.
func (e *testEnv) seedMaxedIntercessor(phone, name string, limit int) {
	e.t.Helper()
	mem := &domain.Member{
		Phone:             phone,
		Name:              name,
		Intercessor:       true,
		SetupStatus:       domain.MemberSetupComplete,
		SetupStage:        domain.MemberSignUpStepFinal,
		WeeklyPrayerLimit: limit,
		PrayerCount:       limit,
		WeeklyPrayerDate:  time.Now().Format(time.RFC3339),
	}
	if err := e.members.Save(e.ctx, mem); err != nil {
		e.t.Fatalf("seed maxed intercessor %s: %v", phone, err)
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

// TestPrayer_AllIntercessorsMaxedOutQueued covers the case where intercessors
// exist on the list but every one of them is at their WeeklyPrayerLimit and
// not eligible for a reset. FindIntercessors must return
// ErrNoAvailableIntercessors and the prayer must end up queued.
func TestPrayer_AllIntercessorsMaxedOutQueued(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000009"
	const intr1 = "+12000000091"
	const intr2 = "+12000000092"

	env.seedRequestor(requestor, "Sam")
	env.seedMaxedIntercessor(intr1, "Ivy", 1)
	env.seedMaxedIntercessor(intr2, "Ian", 1)

	env.handle(msg(requestor, "please pray for my grandmother this evening"))
	env.sender.AssertSentTo(t, requestor, messaging.MsgPrayerQueued)
	env.sender.AssertNotSentTo(t, intr1)
	env.sender.AssertNotSentTo(t, intr2)

	env.requireNoActivePrayer(intr1)
	env.requireNoActivePrayer(intr2)
	env.requireQueuedCount(1)
}

// TestPrayer_MixedPoolUsesOnlyAvailable verifies partial-availability:
// one intercessor is available, one is maxed. FindIntercessors must return
// just the available one and the prayer is assigned (only) to them.
func TestPrayer_MixedPoolUsesOnlyAvailable(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000010"
	const available = "+12000000101"
	const maxed = "+12000000102"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(available, "Ivy", 5)
	env.seedMaxedIntercessor(maxed, "Max", 1)

	env.handle(msg(requestor, "please pray for my health this Saturday"))

	env.sender.AssertSentTo(t, available, "health")
	env.sender.AssertNotSentTo(t, maxed)
	env.sender.AssertSentTo(t, requestor, messaging.MsgPrayerAssigned)

	env.requireActivePrayer(available)
	env.requireNoActivePrayer(maxed)
}

// TestPrayer_AnonUppercaseMidBody verifies the trigger is case-insensitive and
// works mid-body (not just at the start). handleTriggerWords uses a
// case-insensitive regex.
func TestPrayer_AnonUppercaseMidBody(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000011"
	const intr = "+12000000111"

	env.seedRequestor(requestor, "RealName")
	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(requestor, "please pray for me #ANON today and tomorrow"))

	env.sender.AssertSentTo(t, intr, "Anonymous")
	pryr := env.requireActivePrayer(intr)
	if pryr.Requestor.Name != "Anonymous" {
		t.Fatalf("expected persisted Requestor.Name=Anonymous, got %q", pryr.Requestor.Name)
	}
	if strings.Contains(strings.ToLower(pryr.Request), "#anon") {
		t.Fatalf("expected '#anon' stripped (case-insensitive); got %q", pryr.Request)
	}
}

// TestPrayer_CompleteWithWhitespaceAndCaps verifies cleanStr normalization at
// the COMPLETE PRAYER stage: a "prayed" reply with capitals, leading/trailing
// spaces, and newlines should still trigger the Complete flow.
func TestPrayer_CompleteWithWhitespaceAndCaps(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000012"
	const intr = "+12000000121"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	env.handle(msg(requestor, "please pray for safe travels tomorrow morning"))
	env.requireActivePrayer(intr)
	env.sender.Reset()

	env.handle(msg(intr, "  PRAYED\n"))

	env.sender.AssertSentTo(t, intr, messaging.MsgPrayerThankYou)
	env.sender.AssertSentTo(t, requestor, "Ivy") // PrayerConfirmationTmpl uses intercessor name
	env.requireNoActivePrayer(intr)
}

// TestPrayer_CompleteHandlesAbsentRequestor verifies the Complete flow when
// the original requestor is no longer a member (e.g., they cancelled before
// the intercessor finished praying). The intercessor still gets the thank-you,
// the missing requestor is skipped, and the active prayer is deleted.
func TestPrayer_CompleteHandlesAbsentRequestor(t *testing.T) {
	env := newTestEnv(t)
	const ghostRequestor = "+12000000013"
	const intr = "+12000000131"

	env.seedIntercessor(intr, "Ivy", 5)

	// Seed an active prayer directly with a requestor who does NOT exist in
	// the Member table.
	if err := env.prayers.Save(env.ctx, &domain.Prayer{
		IntercessorPhone: intr,
		Request:          "please pray for safe recovery from surgery",
		Requestor:        domain.Member{Phone: ghostRequestor, Name: "Ghost"},
		Intercessor:      domain.Member{Phone: intr, Name: "Ivy"},
		ReminderDate:     time.Now().Format(time.RFC3339),
	}, false); err != nil {
		t.Fatalf("seed active prayer: %v", err)
	}

	env.handle(msg(intr, "prayed"))

	env.sender.AssertSentTo(t, intr, messaging.MsgPrayerThankYou)
	env.sender.AssertNotSentTo(t, ghostRequestor)
	env.requireNoActivePrayer(intr)
}

// TestPrayer_AssignRollbackOnSendFailure is the headline rollback test: when
// SendMessage to the intercessor fails mid-AssignPrayer, the reservation and
// active prayer must both be reverted. Mocks cannot validate this post-condition
// because they ARE the database.
func TestPrayer_AssignRollbackOnSendFailure(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+12000000008"
	const intr = "+12000000081"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	// Confirm the seeded baseline: PrayerCount starts at 0.
	if env.requireMember(intr).PrayerCount != 0 {
		t.Fatal("seeded intercessor must start with PrayerCount=0")
	}

	// Force the SendMessage call that would notify the intercessor to fail.
	env.sender.FailNextSendTo(intr, errSendFailed)

	err := env.handleExpectErr(msg(requestor, "please pray for my family this week"))
	if err == nil {
		t.Fatalf("expected router.Handle to surface the send failure; got nil. dump: %s", env.dump())
	}

	// Active prayer must NOT exist -- rollbackAssignedPrayer deleted it.
	env.requireNoActivePrayer(intr)

	// Reservation must be rolled back -- intercessor's PrayerCount is 0 again.
	mem := env.requireMember(intr)
	if mem.PrayerCount != 0 {
		t.Fatalf("expected intercessor PrayerCount rolled back to 0; got %d. dump: %s",
			mem.PrayerCount, env.dump())
	}

	// The requestor must not have received MsgPrayerAssigned since assignment failed.
	for _, m := range env.sender.Messages() {
		if m.To == requestor && strings.Contains(m.Body, messaging.MsgPrayerAssigned) {
			t.Fatalf("requestor should not have been told prayer was assigned; got %v", m)
		}
	}
}
