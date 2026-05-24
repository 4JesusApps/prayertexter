//go:build integration

package integration_test

import (
	"strings"
	"testing"
	"time"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
)

// TestScheduler_AssignsQueuedPrayer covers the queued-prayer-becomes-active
// path: a prayer was queued because no intercessors were available; later an
// intercessor signs up; RunScheduledJobs assigns the queued prayer.
func TestScheduler_AssignsQueuedPrayer(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+13000000001"
	const intr = "+13000000011"

	env.seedRequestor(requestor, "Sam")

	// First the requestor sends a prayer with no intercessors available; it queues.
	env.handle(msg(requestor, "please pray for healing for my grandfather"))
	env.requireQueuedCount(1)
	env.sender.AssertSentTo(t, requestor, messaging.MsgPrayerQueued)
	env.sender.Reset() // clear the queued-notification SMS so subsequent asserts are clean

	// Later an intercessor signs up.
	env.seedIntercessor(intr, "Ivy", 5)

	// Scheduler runs.
	if err := env.prayerSvc.RunScheduledJobs(env.ctx); err != nil {
		t.Fatalf("RunScheduledJobs: %v", err)
	}

	// Queued prayer moved to active.
	env.requireQueuedCount(0)
	pryr := env.requireActivePrayer(intr)
	if !strings.Contains(pryr.Request, "grandfather") {
		t.Fatalf("unexpected active prayer body: %q", pryr.Request)
	}
	// Intercessor and requestor were both notified.
	env.sender.AssertSentTo(t, intr, "grandfather")
	env.sender.AssertSentTo(t, requestor, messaging.MsgPrayerAssigned)
}

// TestScheduler_SendsReminderAfterThreshold covers the reminder branch: an
// active prayer with a ReminderDate older than cfg.PrayerReminderHours should
// trigger a reminder SMS and a ReminderCount increment.
func TestScheduler_SendsReminderAfterThreshold(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+13000000002"
	const intr = "+13000000021"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	// Manually seed an active prayer with a stale ReminderDate (4h ago, > 3h
	// threshold). Skipping the AssignPrayer code path so we can control timing.
	old := time.Now().Add(-4 * time.Hour).Format(time.RFC3339)
	stale := &domain.Prayer{
		IntercessorPhone: intr,
		Request:          "please pray for safe travels for my family",
		Requestor:        domain.Member{Phone: requestor, Name: "Sam"},
		Intercessor:      domain.Member{Phone: intr, Name: "Ivy"},
		ReminderDate:     old,
		ReminderCount:    0,
	}
	if err := env.prayers.Save(env.ctx, stale, false); err != nil {
		t.Fatalf("seed active prayer: %v", err)
	}

	if err := env.prayerSvc.RunScheduledJobs(env.ctx); err != nil {
		t.Fatalf("RunScheduledJobs: %v", err)
	}

	// Reminder SMS went out to the intercessor.
	env.sender.AssertSentTo(t, intr, "safe travels")

	// ReminderCount incremented and ReminderDate refreshed.
	pryr := env.requireActivePrayer(intr)
	if pryr.ReminderCount != 1 {
		t.Fatalf("expected ReminderCount=1, got %d", pryr.ReminderCount)
	}
	parsed, err := time.Parse(time.RFC3339, pryr.ReminderDate)
	if err != nil {
		t.Fatalf("parse refreshed ReminderDate: %v", err)
	}
	if time.Since(parsed) > time.Minute {
		t.Fatalf("ReminderDate should have been refreshed; got %s (age %s)", pryr.ReminderDate, time.Since(parsed))
	}
}

// TestScheduler_DoesNotRemindBeforeThreshold verifies a fresh active prayer
// (ReminderDate within threshold) does NOT trigger a reminder.
func TestScheduler_DoesNotRemindBeforeThreshold(t *testing.T) {
	env := newTestEnv(t)
	const requestor = "+13000000003"
	const intr = "+13000000031"

	env.seedRequestor(requestor, "Sam")
	env.seedIntercessor(intr, "Ivy", 5)

	recent := time.Now().Add(-30 * time.Minute).Format(time.RFC3339)
	fresh := &domain.Prayer{
		IntercessorPhone: intr,
		Request:          "please pray for my health this week",
		Requestor:        domain.Member{Phone: requestor, Name: "Sam"},
		Intercessor:      domain.Member{Phone: intr, Name: "Ivy"},
		ReminderDate:     recent,
		ReminderCount:    0,
	}
	if err := env.prayers.Save(env.ctx, fresh, false); err != nil {
		t.Fatalf("seed active prayer: %v", err)
	}

	if err := env.prayerSvc.RunScheduledJobs(env.ctx); err != nil {
		t.Fatalf("RunScheduledJobs: %v", err)
	}

	env.sender.AssertNotSentTo(t, intr)

	pryr := env.requireActivePrayer(intr)
	if pryr.ReminderCount != 0 {
		t.Fatalf("expected ReminderCount=0 (no reminder yet), got %d", pryr.ReminderCount)
	}
}
