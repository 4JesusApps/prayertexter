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

### 1.1 Critical: missing-member lookups break new-user routing and block enforcement
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

### 1.2 High: production Lambda handlers swallow errors and silently drop work
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

### 1.3 High: delete and block flows are not safe under partial failure
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

### 1.4 High: prayer assignment has failure windows that can create ghost active prayers or duplicates
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

### 1.5 Medium: requeued prayers keep stale reminder history
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

## 2. Likely Regressions or Reliability Risks

### 2.1 Queued assignment can starve later prayers
Files:
- `internal/service/prayer.go`

`AssignQueuedPrayers()` breaks the entire loop on the first `ErrNoAvailableIntercessors`. Because availability depends partly on the queued prayer's requestor phone, later prayers may still be assignable but will never be attempted in that run.

### 2.2 First reminder starts late
Files:
- `internal/service/prayer.go`

`ReminderDate` is initialized lazily in `RemindActiveIntercessors()` instead of when the prayer is first assigned. That delays the first reminder by at least one scheduler cycle, and the `>` comparison pushes it even later.

### 2.3 Scheduled jobs will miss data at scale
Files:
- `internal/repository/dynamodb.go`
- `internal/repository/prayer.go`
- `internal/service/prayer.go`

`GetAll()` uses a single `Scan` with no pagination. Once the tables outgrow one scan page, queued assignment and reminder jobs will silently ignore rows.

### 2.4 AWS config is split across two sources of truth
Files:
- `internal/config/config.go`
- `internal/awscfg/awscfg.go`

The refactor introduced a `Config` struct with `AWS.Region`, `AWS.Backoff`, and `AWS.Retry`, but `GetAwsConfig()` ignores those values and hardcodes retry/backoff defaults while reading region directly from env.

### 2.5 Intercessor weekly limit accepts `0`
Files:
- `internal/service/member.go`
- `internal/service/prayer.go`

`signUpFinalIntercessor()` accepts `0` as a valid weekly prayer limit. That can leave users on the intercessor list who are effectively unavailable or only become eligible under odd weekly-reset behavior.

## 3. Improvements That Should Be Considered

### 3.1 Introduce an explicit repository not-found contract
Files:
- `internal/repository/dynamodb.go`
- `internal/repository/member.go`
- `internal/repository/prayer.go`

Recommended direction:
- Return `ErrNotFound`, or return `nil, nil` on misses and make callers handle it explicitly.
- Stop using field heuristics like `SetupStatus != ""` and `Request != ""` to infer existence.

Why:
- This removes the zero-value/member-phone ambiguity that is driving the worst current bug.

### 3.2 Route pre-membership behavior off the inbound phone, not the loaded member
Files:
- `internal/service/router.go`

Recommended direction:
- Use `msg.Phone` for block checks, help/delete replies, signup bootstrap, logging, and error context until membership is confirmed.

Why:
- The inbound phone is always authoritative. The loaded member is optional.

### 3.3 Unify AWS bootstrap around `config.Config`
Files:
- `internal/config/config.go`
- `internal/awscfg/awscfg.go`
- `cmd/prayertexter/main.go`
- `cmd/statecontroller/main.go`
- `dev/prayertexter/main.go`

Recommended direction:
- Pass `config.AWSConfig` into `awscfg.GetAwsConfig()` instead of having it read env/hardcoded defaults directly.

Why:
- The current refactor claims explicit dependency injection, but AWS setup still bypasses the injected config model.

### 3.4 Make destructive flows idempotent or compensating
Files:
- `internal/service/member.go`
- `internal/service/admin.go`
- `internal/service/prayer.go`

Recommended direction:
- Reorder operations, add compensating cleanup, or document and implement idempotent recovery behavior for partial failures.

Why:
- Current happy-path ordering is understandable, but brittle.

### 3.5 Timestamp assignments immediately and clear reminder state on requeue
Files:
- `internal/service/prayer.go`
- `internal/service/member.go`

Recommended direction:
- Set reminder state when a prayer first becomes active.
- Clear reminder state whenever an active prayer is moved back to the queue.

Why:
- That makes reminder timing correct and predictable.

### 3.6 Decouple the router from concrete services
Files:
- `internal/service/router.go`
- `internal/service/router_test.go`

Recommended direction:
- Depend on small service interfaces rather than `*MemberService`, `*PrayerService`, and `*AdminService`.

Why:
- This better matches the design doc, keeps the router thinner, and allows dispatch-only tests.

### 3.7 Remove remaining storage/logging leakage from `domain`
Files:
- `internal/domain/phones.go`
- `internal/repository/phones.go`
- `internal/domain/prayer.go`

Recommended direction:
- Move `Key` handling fully into repositories.
- Remove logging from `domain`.
- Stop overloading `Prayer.IntercessorPhone` as both active intercessor phone and queued prayer ID.

Why:
- The current domain layer is better than before, but not fully pure.

## 4. Regressions Against the Design Docs

### 4.1 What landed well
- `internal/config` is the only package importing Viper.
- The `internal/utility` dissolution into `internal/apperr`, `internal/awscfg`, and service-local helpers is mostly clean.
- Messaging is properly abstracted behind `messaging.MessageSender`.
- Services are materially smaller and clearer than the pre-refactor monolith.
- Generated mocks and testify suites are a major testing improvement.

### 4.2 What only partially landed
- The repository layer did not land the fully explicit generic `Repository[T]` interface described in the architecture spec.
- `Exists()` is still based on field heuristics rather than explicit not-found handling.
- The router depends on concrete services instead of service interfaces.
- The domain layer still contains storage/logging concerns (`Key`, `slog`).
- AWS bootstrap still bypasses parts of the loaded config.

### 4.3 Documentation/process regression
Files:
- `README.md`
- `.gitignore`

Observations:
- `README.md` still documents the old architecture (`internal/db`, `internal/object`, `internal/prayertexter`, `internal/utility`).
- `.gitignore` excludes both `docs/superpowers` and `docs/cursor`.

Why this matters:
- The implemented architecture improved, but the durable documentation got worse.
- The design docs and this analysis are local-only unless intentionally tracked elsewhere.

## 5. Test and Coverage Assessment

### 5.1 Strengths
- `internal/service/prayer_test.go` is the strongest suite and covers several important business rules.
- `internal/service/member_test.go` gives decent confidence in signup and delete/requeue happy paths.
- `internal/domain/phones_test.go` is focused and useful.
- `internal/apperr/apperr_test.go` and `internal/config/config_test.go` are proportionate to package complexity.

### 5.2 Gaps that matter
- `internal/awscfg`: no tests
- `internal/messaging/pinpoint.go`: no tests
- `internal/messaging/profanity.go`: no tests
- `internal/repository/member.go`: no direct tests
- `internal/repository/prayer.go`: no direct tests
- `internal/repository/phones.go`: no direct tests
- `internal/service/router.go`: missing important edge/error-path coverage
- `RunScheduledJobs()`: no direct coverage

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
Files:
- `internal/service/router.go`
- `internal/service/prayer.go`
- `internal/service/member.go`
- `internal/messaging/pinpoint.go`

The refactor standardized error/log handling in useful ways, but the code now logs raw message bodies and phone numbers in several places. That may be acceptable operationally, but it is a data-handling decision that should be reviewed explicitly rather than inherited accidentally.

### 6.2 Global-state footgun in profanity handling
Files:
- `internal/messaging/profanity.go`

`CheckProfanity()` mutates `goaway.DefaultProfanities` on each call. It works, but it reintroduces hidden global state into a codebase that otherwise moved toward more explicit dependencies.

### 6.3 `cmd/announcer` is still a stub
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
