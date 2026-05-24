# PrayerTexter

PrayerTexter is an SMS-based prayer request service. People who want prayer text a
request to the prayer line; volunteer intercessors receive the request on their phones,
pray for it, and reply when they're done. The original requestor gets a confirmation
text once someone has prayed.

Everything happens over plain text messages — no app to install, no account portal.

## How it works for members

1. **Sign up.** Text `pray` to the prayer line. The system walks you through a short
   setup over a few texts: your name (or stay anonymous), whether you'd also like to
   pray for others as an intercessor, and — if so — how many requests you can handle
   per week.
2. **Submit a request.** Once signed up, text any message to the same number and the
   system treats it as a prayer request. There's no special prefix or format.
3. **Get confirmation.** When an intercessor has prayed for your request, you'll get a
   text letting you know.

If no intercessor is immediately available, your request is held in a queue. A
scheduled job assigns queued requests as soon as someone has capacity, and you'll be
notified just the same when it's prayed for.

## How it works for intercessors

Intercessors are members who opted in to receiving others' requests during sign-up.

1. **Receive a request.** When a request comes in and you have capacity for the week,
   the system texts the request to you.
2. **Pray and confirm.** Once you've prayed, reply `prayed`. The original requestor
   gets a confirmation text; you become available for the next request.

If you don't respond after a while, the system will send a reminder.

## Commands

| Text this | What happens |
|-----------|--------------|
| `pray` | Starts sign-up (for new numbers) |
| *any message* | Submits a prayer request (for signed-up members) |
| `prayed` | Confirms you've prayed for an active request (intercessors only) |
| `help` | Returns contact and help information |
| `cancel` or `stop` | Removes you from the system completely |

Messages are case-insensitive and tolerate surrounding whitespace.

## Privacy and safety

- Anonymous sign-up is supported; an anonymous member's name is masked in the requests
  intercessors see.
- Profanity is filtered: a request containing flagged language is rejected with an
  explanatory reply rather than being forwarded.
- Designated administrators can block a phone number by texting a message containing
  `#block +15555550100`. Blocked numbers' messages are silently dropped going
  forward.
- Texting `cancel` or `stop` removes a member entirely. If the member was an
  intercessor with an in-flight request, that request is requeued so it isn't lost.

---

# For developers

## Stack

- **Go** Lambda functions on the `provided.al2023` runtime (ARM64).
- **AWS End User Messaging SMS V2** (the modern successor to Pinpoint SMS) for inbound
  and outbound SMS.
- **SNS** delivers inbound SMS to the prayertexter Lambda; **EventBridge** triggers
  the statecontroller Lambda on a schedule.
- **DynamoDB** for all persistence.
- **AWS SAM** for build and deploy across three CloudFormation stacks (`db`,
  `prayertexter`, `statecontroller`).

## Architecture

Three Lambda entry points under `cmd/`:

- `cmd/prayertexter/` — SNS-triggered handler for every inbound text. Parses the SNS
  envelope, dispatches each record to the router, and returns a joined error so SNS
  retry and DLQ semantics work correctly.
- `cmd/statecontroller/` — EventBridge-triggered scheduled jobs: drains the queued-
  prayer table and reminds intercessors with long-outstanding active prayers.
- `cmd/announcer/` — placeholder for broadcast announcements (not yet implemented).

Inbound flow:

```
Inbound SMS
  → AWS End User Messaging SMS V2
  → SNS topic
  → prayertexter Lambda
      → Router (internal/service/router.go) decides intent from message text + member state
          → MemberService    (sign-up, deletion, help)
          → PrayerService    (request, complete, queue, assign, remind)
          → AdminService     (#block)
      → DynamoDB via internal/repository/
      → Outbound SMS via internal/messaging/ (PinpointSender)
```

Lower-level packages don't import higher-level ones:
`config` → `domain` ← `repository` ← `messaging` ← `service` ← `cmd`.

## Configuration

All settings live in `internal/config/` and load via Viper. Override any setting with
`PRAY_CONF_*` environment variables: AWS region / retry / backoff, DynamoDB table names
and timeouts, SMS phone pool, intercessors per prayer, reminder window.

## Tests

- **Unit tests** (mockery-generated mocks, no AWS calls):
  ```
  go test ./...
  ```
- **Integration tests** against a real DynamoDB Local container (via testcontainers-go,
  requires Docker):
  ```
  make test-integration
  ```
  Covers the full prayer lifecycle, sign-up state machine, scheduler jobs, admin
  block flow, and router behavior with real DDB state. Lives under
  `internal/integration/` behind the `integration` build tag.
- **All tests**:
  ```
  make test-all
  ```

The broader testing strategy (what each layer catches, what's planned for E2E in
the cloud) is documented in `docs/test/2026-05-17-e2e-test-strategy.md`.

## Local development

SAM local + DynamoDB Local in Docker. The walkthrough — prerequisites, table setup,
running the Lambda behind a local API Gateway, sending test messages with curl — is in
`dev/README.md`. A seed script (`dev/addmembers.sh`) populates the local DDB with
sample members and intercessors.

## Deploy

Three SAM stacks deployed in order: `db` first, then `prayertexter`, then
`statecontroller`. Commands and ordering details are in `deploy/README.md`.

## Build version

Production binaries are stamped with their git commit SHA (via
`runtime/debug.ReadBuildInfo()`, no `-ldflags` wiring required). Every Lambda invocation
logs the running version on the first log line, which is handy for "is the new code
actually live yet?" checks in CloudWatch.
