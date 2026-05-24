# prayertexter

This application is a work in progress!

PrayerTexter is a Go application that lets users submit prayer requests via text message (SMS). These requests are then assigned to “intercessors”—other users who have volunteered to pray over incoming requests. Once prayed for, the original requestor receives a confirmation. Below is a thorough overview of how the application works, the business logic, and the overall code layout.


## High-Level Functionality

1. **User Sign-Up and Membership**
   • A user texts “pray” to a designated phone number to begin the sign-up process.
   • The application steps the user through a multi-stage signup, where the user can choose to remain anonymous, whether to become an intercessor, and how many prayers they can handle per week.
   • Upon successful sign-up, the user can send prayer requests any time.
   • A user can also opt out by texting “cancel” or “stop,” which removes them from the system.

2. **Submitting Prayer Requests**
   • Members simply text any request to the same phone number (e.g., “Please pray for XYZ”).
   • The system looks for available intercessors (factors: weekly prayer counts, whether they already have an active prayer, etc.).
   • If intercessors are available, the request is sent out to them over SMS. If none are available, the request remains in a “queued” state until the system can assign it.

3. **Intercessing and Confirmation**
   • When any intercessor has prayed for the request, they reply “prayed.”
   • PrayerTexter then notifies the requestor that someone has prayed. That intercessor’s active prayer is removed from their queue (so they can accept more).

4. **Additional Features**
   • Users can text “help” to receive the phone number’s contact and help information (required by SMS service regulations).
   • Administrators can text a message containing `#block <phone>` to block a phone number; the blocked user is removed and a notification SMS is sent.
   • A separate scheduled Lambda (`statecontroller`) drains the queued-prayer table and reminds intercessors with long-outstanding active prayers.


## Main Technical Flows

1. **Sign-Up Flow**
   1) A user texts “pray.”
   2) The system checks if they are new; if so, sets them as “IN PROGRESS,” step one.
   3) They are asked their name (or choose “2” for anonymous).
   4) They decide whether to be a regular member or an intercessor. If intercessor, how many prayers per week.
   5) The user is flagged “COMPLETE,” enabling them to submit requests. If intercessor, they’re added to the “IntercessorPhones” list.

2. **Prayer Request**
   1) A member texts any arbitrary message with a prayer need.
   2) The system checks for profanity. If detected, an explanatory reply is sent and the request is not assigned.
   3) The service finds available intercessors. Each candidate’s weekly quota is reserved atomically inside `AssignPrayer`, with rollback on save or SMS failure.
   4) The request is saved as an “active prayer” for each intercessor and sent over SMS.
   5) If no intercessors can be assigned, the request goes into “QueuedPrayer” with a stable `QueueID`. The scheduled job picks it up later.

3. **Completing a Prayer**
   1) An intercessor replies “prayed.”
   2) If an active prayer is found for their phone number, the requestor is notified and the active prayer is deleted.
   3) The intercessor is again eligible to receive new prayers.

4. **Member Removal**
   1) A user texts “cancel” or “stop.”
   2) Intercessor cleanup runs first: phone is removed from `IntercessorPhones`, and any active prayer they hold is requeued (with a fresh `QueueID` and cleared reminder state). The member row is only deleted after that completes successfully — otherwise the phone-list update is rolled back.

5. **Block Flow**
   1) An administrator texts a message containing `#block <phone>` (regex-extracted).
   2) The target is removed via the normal delete path (without the user-facing “you have been removed” SMS), then the phone is added to `BlockedPhones`.
   3) The blocked user receives a final notification SMS; the admin gets a success confirmation. Subsequent messages from the blocked phone are dropped at the router stage based on `msg.Phone` (not the loaded member), so blocking remains effective after the member row is gone.

6. **Scheduled Jobs (`statecontroller`)**
   • `AssignQueuedPrayers` walks the queued table, finds intercessors for each queued prayer, marks `RequestorNotified` before deleting the queued row so retries are idempotent.
   • `RemindActiveIntercessors` re-sends a reminder for active prayers whose `ReminderDate` is older than the configured cutoff. `ReminderDate` is stamped at assignment time, so the first reminder fires on the expected schedule.
   • Both jobs use paginated DynamoDB scans, so they do not silently drop rows once tables outgrow a single scan page.
   • Errors from either job are joined and returned to the Lambda runtime so retry/DLQ behavior is preserved.


## Directory and Code Structure

PrayerTexter follows a layered architecture: `config` → `domain` ← `repository` ← `messaging` ← `service` ← `cmd`. Lower-level packages do not depend on higher-level ones.

1. **`cmd/` (Lambda Entrypoints)**
   - `cmd/prayertexter/main.go` — Lambda handler triggered by SNS. Processes every record in the batch, returns a joined error so SNS retries and DLQ behavior work correctly.
   - `cmd/statecontroller/main.go` — Scheduled Lambda that runs `PrayerService.RunScheduledJobs` (queued-prayer drainer + active-prayer reminder loop).
   - `cmd/announcer/main.go` — Placeholder for sending broadcast announcements (not yet implemented).

2. **`dev/prayertexter/main.go`**
   - SAM-local entrypoint that wraps the same router behind an API Gateway trigger for local testing.

3. **`internal/config`**
   - Single place that loads configuration via Viper. No other package imports Viper.
   - Provides defaults for AWS region/retry/backoff, DynamoDB timeouts and table names, SMS phone pool, intercessors-per-prayer, and reminder window.
   - All settings are overridable via `PRAY_CONF_*` environment variables.

4. **`internal/awscfg`**
   - Builds the `aws.Config` used by every AWS SDK client. Takes region, retry attempts, and max-backoff seconds as primitives from `config.Config`.
   - Includes a logging retryer that emits a structured log on each retry attempt.

5. **`internal/apperr`**
   - Small error-wrapping/logging helpers used uniformly across services and repositories.

6. **`internal/domain`**
   - Pure domain types: `Member`, `Prayer`, `TextMessage`, `BlockedPhones`, `IntercessorPhones`. No storage or logging concerns.

7. **`internal/messaging`**
   - SMS sending via Amazon Pinpoint behind a `MessageSender` interface.
   - User-facing message constants and templates.
   - Profanity detection (`go-away` with an allowlist applied once at package init).

8. **`internal/repository`**
   - Generic DynamoDB wrapper `DynamoDBRepository[T]` (paginated scan included).
   - Concrete wrappers: `MemberRepository`, `PrayerRepository` (active vs queued table selection), `BlockedPhonesRepository`, `IntercessorPhonesRepository`.
   - Returns `ErrNotFound` on missing items so callers handle absence explicitly instead of inferring it from zero-value fields.

9. **`internal/service`**
   - Business logic, split by concern:
     - `Router` dispatches incoming messages.
     - `MemberService` handles signup, delete, requeue.
     - `PrayerService` handles request, assign, complete, queued-drain, and reminders.
     - `AdminService` handles the `#block` flow.

10. **`internal/mocks/`**
    - Mockery-generated mocks for the repository and messaging interfaces. Used by the testify suites in each service test file.

11. **`deploy/` and `Makefile`**
    - SAM template and build scripts. The Makefile cross-compiles each `cmd/*` binary for the Lambda runtime.


## Testing

- Each package has co-located `*_test.go` files using `testify/suite` and the generated mocks.
- Strongest coverage lives in `internal/service` (router/member/prayer/admin).
- Run everything with `go test ./...`.


## Summary

- `cmd/prayertexter/main.go` is the SNS-triggered Lambda. `cmd/statecontroller/main.go` is the scheduled-jobs Lambda. `dev/prayertexter/main.go` is the local-dev API Gateway variant.
- `internal/service` orchestrates each message’s flow: signup, prayer requests, completion, removal, blocking. The `Router` is the single entry point.
- `internal/domain` holds pure domain types.
- `internal/repository` provides generic and concrete DynamoDB access with explicit `ErrNotFound` semantics and paginated scans.
- `internal/messaging` builds and sends SMS via Pinpoint.
- `internal/config`, `internal/awscfg`, and `internal/apperr` handle configuration, AWS client bootstrap, and error wrapping.
- Tests use generated mocks so the suite never calls real AWS services.
