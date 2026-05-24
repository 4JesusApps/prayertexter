# Refactor Analysis: `e329447a04e02537c1c1207740070840d752ef78`

## Scope and Validation
- Compared the live repository against:
  - `docs/superpowers/specs/2026-04-19-architecture-refactor-design.md`
  - `docs/superpowers/specs/2026-04-20-dissolve-utility-package-design.md`
- Verified that the current branch tree is identical to the named refactor commit:
  - `HEAD^{tree}` = `d7080e0d66619093c9f4475c9f81f55b646bb5be`
  - `e329447a04e02537c1c1207740070840d752ef78^{tree}` = `d7080e0d66619093c9f4475c9f81f55b646bb5be`
- Validation run:
  - `go test ./...` -> passes
  - `go vet ./...` -> passes
- Coverage snapshot highlights:
  - `internal/service`: `74.5%`
  - `internal/repository`: `49.2%`
  - `internal/messaging`: `9.4%`
  - `internal/awscfg`: `0.0%`

## Executive Summary
The refactor was directionally successful. The codebase now has a much clearer `config` / `domain` / `repository` / `messaging` / `service` split, `internal/utility` was dissolved in a sensible way, and service-level tests are much easier to read and maintain than the pre-refactor style.

However, the refactor also introduced or at least codified one severe boundary bug: repository misses now deserialize into zero-value structs, while routing and service logic assume a real `Member` with a populated `Phone`. That breaks first-contact signup and makes the block list ineffective after a blocked user has been deleted from the member table.

The other major gap is operational reliability. The production Lambda handlers swallow errors instead of returning them, and several destructive flows are not safe under partial failure. The architecture is better than before, but it is not yet as robust or as internally consistent as the design docs describe.

## 1. Confirmed Issues Caused or Exposed by the Refactor
Status update: all issues in section 1 below have now been fixed in the current working tree.

### 1.1 Critical: missing-member lookups break new-user routing and block enforcement
Status: Fixed.

Files:
- `internal/repository/dynamodb.go`
- `internal/repository/dynamodb_test.go`
- `internal/service/router.go`
- `internal/service/member.go`
- `internal/service/admin.go`

What is happening:
- `DynamoDBRepository.Get()` returns a zero-value struct when DynamoDB does not find an item.
- `internal/repository/dynamodb_test.go` explicitly treats that as the expected behavior.
- `Router.Handle()` then uses `mem.Phone` instead of `msg.Phone` for block checks, help/delete replies, signup stage one, logging, and error wrapping.

Practical effects:
- A brand-new user texting `pray` enters `signUpStageOne()` with `mem.Phone == ""`, so the code saves a blank-phone member record and tries to reply to a blank destination.
- Unknown users texting `help`, `stop`, or `cancel` operate on a blank phone.
- Blocked users are added to `BlockedPhones`, then deleted from `Member`, and future block checks fail because the router compares the block list against `mem.Phone`, not the inbound `msg.Phone`.

Why this matters:
- This is a real correctness regression, not just an architectural smell.
- It undermines first-time signup, opt-out/help behavior, and the block feature.

How it was fixed:
- `internal/repository/dynamodb.go` now returns `repository.ErrNotFound` instead of silently unmarshaling a zero-value struct on a missing item.
- `internal/repository/member.go` and `internal/repository/prayer.go` now treat `ErrNotFound` as "does not exist" instead of inferring existence from empty fields.
- `internal/repository/phones.go` now treats missing fixed-key phone-list rows as empty collections, so the new not-found contract does not break first-time setup.
- `internal/service/router.go` now falls back to a synthetic member with `Phone = msg.Phone` when a member lookup misses, and block checks/logging now use the inbound phone (`msg.Phone`) rather than `mem.Phone`.
- Added regression coverage in `internal/repository/dynamodb_test.go` and `internal/service/router_test.go` for not-found lookups, first-contact signup, and blocked numbers with no member row.

### 1.2 High: production Lambda handlers swallow errors and silently drop work
Status: Fixed.

Files:
- `cmd/prayertexter/main.go`
- `cmd/statecontroller/main.go`
- `dev/prayertexter/main.go`

What is happening:
- The production handlers do not return errors.
- `cmd/prayertexter/main.go` logs when `len(snsEvent.Records) > 1` but still processes only `Records[0]`.
- Both production handlers log bootstrap/runtime failures and `return` successfully.

Practical effects:
- Lambda/SNS retries and DLQ behavior are bypassed.
- Multi-record SNS invocations drop extra records.
- `statecontroller` job failures can look like successful invocations.

Why this matters:
- Real production failures become silent successes.
- The local dev handler is stricter than production, so the safest behavior is not what gets deployed.

How it was fixed:
- `cmd/prayertexter/main.go` now returns an error from the Lambda handler, rejects empty SNS batches, processes every SNS record in the batch, and returns a joined error if any record fails.
- `cmd/statecontroller/main.go` now returns an error from the Lambda handler instead of logging and returning success on failures.
- `internal/service/prayer.go` now returns an aggregated error from `RunScheduledJobs()` after attempting both scheduled jobs, so Lambda retry/DLQ behavior can work correctly without losing visibility into whichever job failed.

### 1.3 High: delete and block flows are not safe under partial failure
Status: Fixed.

Files:
- `internal/service/member.go`
- `internal/service/admin.go`

What is happening:
- `MemberService.Delete()` deletes the member first, then removes the intercessor phone, then requeues an active prayer, then sends the confirmation text.
- `AdminService.BlockUser()` saves the blocked list before invoking the member deletion path.

Practical effects if any later step fails:
- Member row deleted but intercessor list not updated
- Member row deleted but active prayer not requeued
- Blocked list updated but delete flow only partially completed
- User-facing SMS sequence no longer matches durable state

Why this matters:
- The system can end up half-deleted and hard to reconcile.

How it was fixed:
- `internal/service/member.go` no longer deletes the member record first. Intercessor cleanup now happens before the destructive member delete.
- `internal/service/member.go` now restores the intercessor phone list if active-prayer cleanup/requeue fails after the phone list update.
- `internal/service/member.go` now requeues active prayers with rollback: if saving the queued prayer fails after deleting the active prayer, the original active prayer is restored.
- `internal/service/admin.go` now cleans up the target member before persisting the block list, and it uses `DeleteWithoutNotification()` so the blocked-user flow only sends the final blocked notification rather than a normal removal SMS plus a blocked SMS.
- Added regression coverage in `internal/service/member_test.go` and `internal/service/admin_test.go` to verify cleanup failures do not proceed to the destructive next step.

### 1.4 High: prayer assignment has failure windows that can create ghost active prayers or duplicates
Status: Fixed.

Files:
- `internal/service/prayer.go`

What is happening:
- `processIntercessor()` increments intercessor quota before assignment is fully complete.
- `AssignPrayer()` saves the active prayer before the SMS send succeeds.
- `AssignQueuedPrayers()` deletes the queued row only after all active saves/sends complete.

Practical effects:
- If the SMS send fails after `AssignPrayer()` saves, the intercessor is treated as busy and has consumed quota even though they never received the prayer.
- If queued assignment fails after some active saves/sends but before queued delete, the same queued prayer can be assigned again on the next run.
- If requestor notification fails after queued delete, the work has happened but the requestor never gets the confirmation.

Why this matters:
- These are real failure windows in core business flows.

How it was fixed:
- `internal/service/prayer.go` no longer increments intercessor quota inside `FindIntercessors()`. Candidate discovery is now non-mutating.
- `internal/service/prayer.go` now reserves intercessor quota inside `AssignPrayer()` and rolls that reservation back if the active-prayer save fails.
- `internal/service/prayer.go` now deletes the newly-saved active prayer and restores the original intercessor state if the SMS send fails, eliminating ghost active prayers and burned capacity from single-assignment failures.
- Queued prayers now carry a stable `QueueID`, and active prayers copied from the queue retain that `QueueID`. `AssignQueuedPrayers()` uses that ID to detect previously-created assignments before retrying, which prevents duplicate re-assignment after a partial failure.
- `AssignQueuedPrayers()` now marks `RequestorNotified = true` on the queued row before deleting it. If deletion fails after the notification send, a retry will see the persisted notification state and will only retry cleanup instead of re-notifying/re-assigning.
- Added regression coverage in `internal/service/prayer_test.go` for rollback on SMS-send failure and for the new queued-processing persistence path.

### 1.5 Medium: requeued prayers keep stale reminder history
Status: Fixed.

Files:
- `internal/service/member.go`
- `internal/service/prayer.go`
- `internal/domain/prayer.go`

What is happening:
- `moveActivePrayer()` clears `Intercessor` and replaces `IntercessorPhone` with a queue ID.
- It does not clear `ReminderDate` or `ReminderCount`.

Practical effects:
- A reassigned prayer can inherit reminder timing/history from the previous intercessor.

Why this matters:
- Reminder behavior becomes incorrect after requeueing.

How it was fixed:
- `internal/service/member.go` now clears `ReminderDate` and `ReminderCount` whenever an active prayer is moved back to the queue.
- `internal/service/member.go` also generates a fresh queue ID when requeueing an active prayer, so the queued row represents a new assignment lifecycle instead of inheriting stale active-state metadata.
- `internal/service/prayer.go` now stamps `ReminderDate` when an assignment is first created, which removes the delayed-start reminder behavior for newly assigned prayers.
- Added regression coverage in `internal/service/member_test.go` to verify reminder metadata is cleared during requeue.

## 2. Likely Regressions or Reliability Risks

### 2.1 Queued assignment can starve later prayers
Status: Fixed during section 1 remediation.

Files:
- `internal/service/prayer.go`

`AssignQueuedPrayers()` breaks the entire loop on the first `ErrNoAvailableIntercessors`. Because availability depends partly on the queued prayer's requestor phone, later prayers may still be assignable but will never be attempted in that run.

### 2.2 First reminder starts late
Status: Fixed during section 1 remediation.

Files:
- `internal/service/prayer.go`

`ReminderDate` is initialized lazily in `RemindActiveIntercessors()` instead of when the prayer is first assigned. That delays the first reminder by at least one scheduler cycle, and the `>` comparison pushes it even later.

### 2.3 Scheduled jobs will miss data at scale
Status: Fixed.

Files:
- `internal/repository/dynamodb.go`
- `internal/repository/dynamodb_test.go`

`GetAll()` used a single `Scan` with no pagination. Once the tables outgrew one scan page, queued assignment and reminder jobs would silently ignore rows.

How it was fixed:
- `DynamoDBRepository.GetAll()` now loops `Scan` calls, threading `LastEvaluatedKey` from each response into the next request's `ExclusiveStartKey` until DynamoDB reports no more pages. The shared per-call timeout still bounds total work, so a runaway scan surfaces as a returned error (which `RunScheduledJobs()` already aggregates and returns to the Lambda runtime) instead of silent data loss.
- Added regression coverage in `internal/repository/dynamodb_test.go` (`TestGetAll_Pagination`) that exercises a two-page response and asserts the second `Scan` is issued with the prior page's `LastEvaluatedKey`.

### 2.4 AWS config is split across two sources of truth
Status: Fixed.

Files:
- `internal/awscfg/awscfg.go`
- `internal/awscfg/awscfg_test.go`
- `cmd/prayertexter/main.go`
- `cmd/statecontroller/main.go`
- `dev/prayertexter/main.go`

The refactor introduced a `Config` struct with `AWS.Region`, `AWS.Backoff`, and `AWS.Retry`, but `GetAwsConfig()` ignored those values and hardcoded retry/backoff defaults while reading region directly from env. The defaults happened to match, so overrides via `PRAY_CONF_AWS_BACKOFF` / `PRAY_CONF_AWS_RETRY` were silently dropped.

How it was fixed:
- `awscfg.GetAwsConfig` now takes `(ctx, region, maxRetry, maxBackoffSeconds)` as primitives. The `os.Getenv` read and the `defaultRegion`/`defaultMaxRetry`/`defaultMaxBackoff` constants are gone — `internal/config` is the single source for these values.
- All three lambda entrypoints (`cmd/prayertexter/main.go`, `cmd/statecontroller/main.go`, `dev/prayertexter/main.go`) now pass `cfg.AWS.Region`, `cfg.AWS.Retry`, `cfg.AWS.Backoff` from the loaded config. This matches the existing convention where repository/messaging constructors take primitives unpacked from `config.Config` at the cmd layer.
- Added `internal/awscfg/awscfg_test.go` (the package was at 0% coverage per §5.2). The test verifies that the passed region and retry count are plumbed through to the returned `aws.Config`.

### 2.5 Intercessor weekly limit accepts `0`
Status: Fixed.

Files:
- `internal/messaging/messages.go`
- `internal/service/member.go`
- `internal/service/member_test.go`

`signUpFinalIntercessor()` accepted `0` as a valid weekly prayer limit. That left users on the intercessor list who were effectively unavailable (the existing `intr.WeeklyPrayerLimit <= 0` guard in `internal/service/prayer.go` already correctly excluded them from selection, but they shouldn't have been signed up that way to begin with).

How it was fixed:
- `signUpFinalIntercessor()` now rejects `num <= 0` immediately after parsing, before mutating the intercessor phone list or saving the member. The user gets a new `MsgInvalidPrayerLimit` reply explaining the constraint, and stays in step three so their next message reply can complete signup correctly. Non-numeric input continues to route through the existing `signUpWrongInput` path.
- Added `MsgInvalidPrayerLimit` constant in `internal/messaging/messages.go`.
- Added `TestSignUpFinalIntercessor_ZeroLimit` regression coverage. (`cleanStr` strips the leading `-`, so negative input is unreachable via the public API today; the `<= 0` check is kept as a defensive guard against future changes to the input pipeline.)

## 3. Improvements That Should Be Considered

### 3.1 Introduce an explicit repository not-found contract
Status: Fixed.

Files:
- `internal/repository/member.go`
- `internal/repository/prayer.go`
- `internal/service/prayer.go`

Recommended direction:
- Return `ErrNotFound`, or return `nil, nil` on misses and make callers handle it explicitly.
- Stop using field heuristics like `SetupStatus != ""` and `Request != ""` to infer existence.

Why:
- This removes the zero-value/member-phone ambiguity that is driving the worst current bug.

How it was fixed:
- §1.1 already moved `DynamoDBRepository.Get()` onto `ErrNotFound`; this change completes the migration by removing the leftover field heuristics in the wrappers.
- `memberRepository.Exists()` now returns true on any successful `Get` and false on `ErrNotFound`. The `mem.SetupStatus != ""` heuristic is gone.
- `prayerRepository.Exists()` now returns true on any successful `Get` against the active-prayer table and false on `ErrNotFound`. The `pryr.Request != ""` heuristic is gone.
- `PrayerService.Complete()` no longer second-guesses with `pryr.Request == ""` after a successful `Get`. That branch is dead now that the repository contract is explicit.
- No service tests required changes — they mock `Exists()` directly. Existing service-level behavior is preserved because, with §1.1's signup-bootstrap fix in place, every persisted member/prayer row is a real one (no more partial blank-phone rows).

### 3.2 Route pre-membership behavior off the inbound phone, not the loaded member
Status: Skipped (functionally satisfied by §1.1; reshape not worth the churn).

Files:
- `internal/service/router.go`

Recommended direction:
- Use `msg.Phone` for block checks, help/delete replies, signup bootstrap, logging, and error context until membership is confirmed.

Why:
- The inbound phone is always authoritative. The loaded member is optional.

Why skipped:
- The functional content of this recommendation was absorbed by §1.1's synthetic-member fallback in `Router.Handle()` and the matching fallback in `AdminService.BlockUser()`. Block checks, all logging, and all error wrapping already key off `msg.Phone`. Downstream services (`Help`, `Delete`, `SignUp`) receive a synthetic `domain.Member{Phone: msg.Phone}` when no row exists, which routes correctly because zero-value fields (`SetupStage`, `Intercessor`, etc.) signal "not yet a member."
- Taking the recommendation literally would mean reshaping `Help(ctx, phone)` / `Delete(ctx, phone)` / `SignUp(ctx, msg, ???)` to take primitives instead of a member. That's larger touch surface across services + tests with zero behavioral improvement, and `SignUp` would arguably get worse because it needs `mem.SetupStage` for its state machine — leading to mixed signatures or two entry points.
- Instead, a documenting comment was added at `internal/service/router.go:43` describing the synthetic-member contract so the convention is explicit to readers.

### 3.3 Unify AWS bootstrap around `config.Config`
Status: Fixed (satisfied by §2.4 remediation).

Files:
- `internal/awscfg/awscfg.go`
- `internal/awscfg/awscfg_test.go`
- `cmd/prayertexter/main.go`
- `cmd/statecontroller/main.go`
- `dev/prayertexter/main.go`

Recommended direction:
- Pass `config.AWSConfig` into `awscfg.GetAwsConfig()` instead of having it read env/hardcoded defaults directly.

Why:
- The current refactor claims explicit dependency injection, but AWS setup still bypasses the injected config model.

How it was fixed:
- See §2.4. `awscfg.GetAwsConfig` now takes `(ctx, region, maxRetry, maxBackoffSeconds)` as primitives from `config.Config`. The `os.Getenv` read and `defaultRegion`/`defaultMaxRetry`/`defaultMaxBackoff` constants are gone. All three Lambda entrypoints pass `cfg.AWS.Region`, `cfg.AWS.Retry`, `cfg.AWS.Backoff` from `config.Load()`.
- Primitives were chosen over the full `config.AWSConfig` struct to match the codebase convention (repository/messaging constructors take primitives unpacked from `config.Config` at the cmd layer). Net result satisfies the recommendation: `internal/config` is the single source of truth for AWS region, retry, and backoff.

### 3.4 Make destructive flows idempotent or compensating
Status: Fixed (satisfied by §1.3 and §1.4 remediation).

Files:
- `internal/service/member.go`
- `internal/service/admin.go`
- `internal/service/prayer.go`

Recommended direction:
- Reorder operations, add compensating cleanup, or document and implement idempotent recovery behavior for partial failures.

Why:
- Current happy-path ordering is understandable, but brittle.

How it was fixed:
- See §1.3 and §1.4. Delete/block flows now reorder so intercessor cleanup happens before the destructive member delete; intercessor-phone list rollback restores prior state if active-prayer cleanup fails; active-prayer requeue rolls back on save failure; admin block flow uses `DeleteWithoutNotification` to avoid duplicate SMS.
- Prayer assignment now reserves intercessor quota inside `AssignPrayer` and rolls the reservation back on save or SMS failure; queued assignment uses a stable `QueueID` to detect previously-created assignments before retrying and persists `RequestorNotified` before deletion so partial-failure retries are idempotent.

### 3.5 Timestamp assignments immediately and clear reminder state on requeue
Status: Fixed (satisfied by §1.5 remediation).

Files:
- `internal/service/prayer.go`
- `internal/service/member.go`

Recommended direction:
- Set reminder state when a prayer first becomes active.
- Clear reminder state whenever an active prayer is moved back to the queue.

Why:
- That makes reminder timing correct and predictable.

How it was fixed:
- See §1.5. `AssignPrayer` now stamps `ReminderDate = time.Now()` and `ReminderCount = 0` at assignment time (`internal/service/prayer.go:105-106`), removing the lazy-init behavior in `RemindActiveIntercessors`. `moveActivePrayer` clears `ReminderDate`, `ReminderCount`, `Intercessor`, and `RequestorNotified` and generates a fresh `QueueID` when an active prayer is requeued (`internal/service/member.go:108-114`).

### 3.6 Decouple the router from concrete services
Status: Skipped (cosmetic; concrete services have no alternative implementations).

Files:
- `internal/service/router.go`
- `internal/service/router_test.go`

Recommended direction:
- Depend on small service interfaces rather than `*MemberService`, `*PrayerService`, and `*AdminService`.

Why:
- This better matches the design doc, keeps the router thinner, and allows dispatch-only tests.

Why skipped:
- Concrete services have no alternative implementations in production, so the architectural benefit is theoretical. The real win would be tighter `router_test.go` mocks (only the 5-6 methods the router dispatches to instead of the full service chain).
- Cost: three new interface declarations, three mockery configs, and corresponding router/test churn. No behavioral improvement.
- Revisit if router tests start causing pain or if a second router implementation appears.

### 3.7 Remove remaining storage/logging leakage from `domain`
Status: Partially fixed (3.7a and 3.7b done; 3.7c skipped — requires schema migration).

Files:
- `internal/domain/phones.go`
- `internal/repository/phones.go`
- `internal/service/member.go`
- `internal/service/member_test.go`

Recommended direction:
- Move `Key` handling fully into repositories.
- Remove logging from `domain`.
- Stop overloading `Prayer.IntercessorPhone` as both active intercessor phone and queued prayer ID.

Why:
- The current domain layer is better than before, but not fully pure.

How it was fixed:
- 3.7a (Key handling): `Key` removed from `domain.BlockedPhones` and `domain.IntercessorPhones`. Added internal storage-row types (`blockedPhonesRow`, `intercessorPhonesRow`) inside `internal/repository/phones.go` that carry `Key` for DynamoDB marshaling. `Get`/`Save` convert between the row and the domain type, so callers and tests no longer need to know the partition-key constant. The `removeIntercessor` rollback in `internal/service/member.go` simplifies to `&domain.IntercessorPhones{Phones: originalPhones}`. Three test fixtures in `member_test.go` had their `Key: "IntercessorPhones"` fields removed.
- 3.7b (domain logging): `slog.Warn("unable to generate phones, phone list is empty")` removed from `GenRandPhones` in `internal/domain/phones.go`; `log/slog` import dropped. The caller in `internal/service/prayer.go` already logs the same condition.

Why 3.7c skipped:
- The queued-prayer table uses `IntercessorPhone` as its partition key (`internal/repository/prayer.go:25-26`), with a generated ID stuffed into that field for queued rows. Decoupling this properly requires either a per-table partition-key field or switching the queued table to key on `QueueID`. Both require a DynamoDB schema migration and operational planning.
- Current overloading is documented via the dedicated `QueueID` field added in §1.4 and works correctly. Revisit when a schema migration is planned.

## 4. Regressions Against the Design Docs

### 4.1 What landed well
- `internal/config` is the only package importing Viper.
- The `internal/utility` dissolution into `internal/apperr`, `internal/awscfg`, and service-local helpers is mostly clean.
- Messaging is properly abstracted behind `messaging.MessageSender`.
- Services are materially smaller and clearer than the pre-refactor monolith.
- Generated mocks and testify suites are a major testing improvement.

### 4.2 What only partially landed
Status: Resolved or deliberately skipped.

- ~~The repository layer did not land the fully explicit generic `Repository[T]` interface described in the architecture spec.~~ — Skipped along with §3.6 (no alternative implementations exist; the interface would be cosmetic).
- ~~`Exists()` is still based on field heuristics rather than explicit not-found handling.~~ — Fixed in §3.1.
- ~~The router depends on concrete services instead of service interfaces.~~ — Skipped per §3.6.
- ~~The domain layer still contains storage/logging concerns (`Key`, `slog`).~~ — Fixed in §3.7a/b.
- ~~AWS bootstrap still bypasses parts of the loaded config.~~ — Fixed in §2.4/§3.3.

### 4.3 Documentation/process regression
Status: README updated; `.gitignore` decision deferred to the user.

Files:
- `README.md`
- `.gitignore`

Observations:
- ~~`README.md` still documents the old architecture (`internal/db`, `internal/object`, `internal/prayertexter`, `internal/utility`).~~ — Rewritten to reflect the current `config` / `awscfg` / `apperr` / `domain` / `messaging` / `repository` / `service` layout, the SNS-triggered Lambda handler, the scheduled-jobs flow, the `#block` admin trigger, and the synthetic-member fallback.
- `.gitignore` still excludes `docs/superpowers` and `docs/cursor`. This is a deliberate choice (those are local-only design/analysis artifacts). No change made; revisit if you want either tree tracked in git.

## 5. Test and Coverage Assessment

### 5.1 Strengths
- `internal/service/prayer_test.go` is the strongest suite and covers several important business rules.
- `internal/service/member_test.go` gives decent confidence in signup and delete/requeue happy paths.
- `internal/domain/phones_test.go` is focused and useful.
- `internal/apperr/apperr_test.go` and `internal/config/config_test.go` are proportionate to package complexity.

### 5.2 Gaps that matter
- ~~`internal/awscfg`: no tests~~ — Partial coverage added in §2.4 (region + retry plumb-through). Backoff inspection still missing (no accessor on wrapped retryer).
- `internal/messaging/pinpoint.go`: no tests — Skipped; needs Pinpoint client mocks, low value relative to effort.
- ~~`internal/messaging/profanity.go`: no tests~~ — Added alongside §6.2 fix (clean text, allowlisted words, global-dictionary-mutation regression).
- ~~`internal/repository/member.go`: no direct tests~~ — Added (`member_test.go`).
- ~~`internal/repository/prayer.go`: no direct tests~~ — Added (`prayer_test.go`) covering active/queued table selection and `Exists` semantics.
- ~~`internal/repository/phones.go`: no direct tests~~ — Added (`phones_test.go`) covering row↔domain conversion and partition-key attachment for both blocked and intercessor lists.
- `internal/service/router.go`: missing important edge/error-path coverage — §1.1 added the most critical cases (missing-member routing, blocked-phone-with-no-member-row). Further coverage deferred.
- `RunScheduledJobs()`: no direct coverage — Deferred (small follow-up if you want it).

### 5.3 Most important missing regression tests
- Missing member record -> incoming `pray`, `help`, `stop`
- Blocked number after its member row has been deleted
- Multi-record SNS event handling
- Queued assignment where the first queued prayer is unassignable but a later one is assignable
- Partial failure during `AssignPrayer()`, `Complete()`, and `Delete()`
- Repository wrapper semantics:
  - fixed-key phone repos
  - active-vs-queued table selection
  - `Exists()` behavior
- `awscfg` retry/backoff behavior
- `pinpoint` local-mode and throttling behavior

### 5.4 Why the current tests missed the biggest bug
Files:
- `internal/repository/dynamodb_test.go`
- `internal/service/router_test.go`

The tests validate each side independently but miss the broken contract between them:
- repository tests explicitly approve zero-value-on-miss behavior
- router tests always mock a `Member` with a populated `Phone`

That is why the most serious current bug can exist while the suite still passes.

## 6. Anything Else

### 6.1 Logging/privacy review
Status: Accepted as-is (explicit product decision, 2026-05-24).

Files:
- `internal/service/router.go`
- `internal/service/prayer.go`
- `internal/service/member.go`
- `internal/messaging/pinpoint.go`

The refactor standardized error/log handling in useful ways, but the code now logs raw message bodies and phone numbers in several places.

Decision: raw phone numbers and message bodies remain in logs. This was reviewed explicitly and accepted on 2026-05-24; it is not an accidental inheritance. Revisit if the deployment context, retention policy, or regulatory posture changes.

### 6.2 Global-state footgun in profanity handling
Status: Fixed.

Files:
- `internal/messaging/profanity.go`
- `internal/messaging/profanity_test.go`

`CheckProfanity()` previously mutated `goaway.DefaultProfanities` on each call.

How it was fixed:
- The filtered profanity list is now built once at package init from a `slices.Clone` of `goaway.DefaultProfanities` — the third-party library's default slice is never touched.
- A single `*goaway.ProfanityDetector` is constructed at init using `WithCustomDictionary(filtered, goaway.DefaultFalsePositives, goaway.DefaultFalseNegatives)` and reused across calls.
- Added `profanity_test.go` covering clean text, the allowlist (`jerk`/`ass`/`butt`), and a regression that asserts `goaway.DefaultProfanities` is unchanged after calls into `CheckProfanity`.

### 6.3 `cmd/announcer` is still a stub
Status: Acknowledged (no action). The updated README no longer presents `announcer` as a fully implemented command.

Files:
- `cmd/announcer/main.go`

This is not a refactor bug, but it is worth noting because the README presents it as a real part of the system.

## 7. Priority Order
1. Fix repository miss handling and make router use `msg.Phone` for pre-membership flows.
2. Make production Lambda handlers return errors and handle SNS records safely.
3. Make delete/block/assignment flows safe under partial failure.
4. Clear reminder state on requeue and timestamp assignments immediately.
5. Add the missing regression tests above.
6. Unify AWS config flow and update stale documentation.

## Bottom Line
The refactor materially improved structure, readability, and test ergonomics. It also left the codebase closer to the desired architecture than the pre-refactor version. But the current implementation still has one severe repository/router contract bug, several reliability problems around swallowed runtime failures and partial updates, and a few important design-doc gaps.

This should be treated as a successful architectural step that still needs a focused hardening pass before the new boundaries can be considered fully reliable.
