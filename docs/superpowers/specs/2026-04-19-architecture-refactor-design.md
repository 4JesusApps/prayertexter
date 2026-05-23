# Architecture Refactor Design

## Goals

1. Clean separation of concerns — business logic isolated from infrastructure (DB, SMS)
2. Easier testability — services testable with simple mocks, no AWS SDK types in tests
3. Simplified unit tests — reduce code duplication, remove verbose mock setup, use testify + generated mocks
4. Explicit dependency injection — no global Viper state, config passed as a struct

## Package Structure

### Before

```
internal/
  config/          — Viper global init
  db/              — DDBConnecter interface + generic DDB helpers
  messaging/       — TextSender interface (AWS types), SendText, messages, profanity
  object/          — Member, Prayer, BlockedPhones, IntercessorPhones (each with Get/Put/Delete)
  prayertexter/    — MainFlow + all business logic in one file (~725 LOC)
  statecontroller/ — RunJobs, AssignQueuedPrayers, RemindActiveIntercessors
  test/            — Hand-rolled mocks, Case struct, validation helpers
  utility/         — Error handling, AWS config, ID generation
```

### After

```
internal/
  config/          — Config struct + Load() (Viper contained here only)
  domain/          — Pure structs: Member, Prayer, BlockedPhones, IntercessorPhones, TextMessage
  messaging/       — MessageSender interface, Pinpoint implementation, templates, profanity
  repository/      — Generic Repository[T] interface + DDB implementation, domain-specific repos
  service/         — MemberService, PrayerService, AdminService, Router
  utility/         — Error handling, AWS config, ID generation (unchanged)
```

### Deleted Packages

- `internal/object/` — replaced by `internal/domain/`
- `internal/db/` — folded into `internal/repository/`
- `internal/test/` — replaced by generated mocks + testify
- `internal/statecontroller/` — folded into `PrayerService` methods

### Dependency Direction (strict, no cycles)

```
cmd/ → service/ → repository/ → domain/
                → messaging/  → domain/
                → config/
       repository/ → config/ (table names, timeouts)
       messaging/  → config/ (phone pool, timeout)
```

`domain/` imports nothing from the project. `utility/` is available to all layers.

---

## Layer 1: Domain (`internal/domain/`)

Domain objects become pure data structs with no persistence, no messaging, and no Viper imports.

### Files

- `member.go` — Member struct + constants (setup stages, setup statuses)
- `prayer.go` — Prayer struct + constants (key names)
- `phones.go` — BlockedPhones and IntercessorPhones structs with in-memory list manipulation methods (AddPhone, RemovePhone, GenRandPhones)
- `message.go` — TextMessage struct (moved from messaging package)

### Key Changes

- `Member` loses `Get()`, `Put()`, `Delete()`, `SendMessage()` — becomes just fields
- `Prayer` loses `Get()`, `Put()`, `Delete()` — becomes just fields
- `IsMemberActive()` and `IsPrayerActive()` move to repository layer as `Exists()` methods
- `BlockedPhones` and `IntercessorPhones` keep `AddPhone()`, `RemovePhone()` — these are pure in-memory list operations, not I/O
- `GenRandPhones()` stays on `IntercessorPhones` but takes `intercessorsPerPrayer int` as a parameter instead of reading from Viper
- `TextMessage` moves here from `messaging` package since it's a domain concept
- `CheckProfanity()` stays on `TextMessage` — it's pure logic with no I/O

---

## Layer 2: Repository (`internal/repository/`)

### Files

- `repository.go` — Generic `Repository[T]` interface + DynamoDB implementation
- `member.go` — `MemberRepository` interface
- `prayer.go` — `PrayerRepository` interface
- `phones.go` — `IntercessorPhonesRepository` + `BlockedPhonesRepository` interfaces

### Generic Base

```go
type Repository[T any] interface {
    Get(ctx context.Context, key string) (*T, error)
    Save(ctx context.Context, item *T) error
    Delete(ctx context.Context, key string) error
}
```

The DynamoDB implementation takes the table name, key field name, and a "does this item exist?" check function at construction time. One implementation struct, configured per domain type.

### Domain-Specific Extensions

- **`MemberRepository`** — adds `Exists(ctx, phone) (bool, error)` (replaces `IsMemberActive`)
- **`PrayerRepository`** — wraps two tables (active + queued). Methods take a `queued bool` parameter. Adds `GetAll(ctx, queued) ([]Prayer, error)` for statecontroller scans. Adds `Exists(ctx, phone) (bool, error)` (replaces `IsPrayerActive`).
- **`BlockedPhonesRepository`** — just `Repository[BlockedPhones]` with a fixed key value (`BlockedPhonesKeyValue`) internally. No extensions needed.
- **`IntercessorPhonesRepository`** — same as blocked phones, fixed key value internally.

### What This Eliminates

- All 4 domain objects' duplicated `Get/Put/Delete` methods
- All `viper.GetString(tableConfigPath)` calls from domain objects
- The `db.DDBConnecter` interface import from domain objects
- The existing `internal/db/` package (folded in)

### Config Injection

Each repository constructor takes relevant config values (table name, timeout) as plain parameters. No Viper calls inside the repository.

---

## Layer 3: Messaging (`internal/messaging/`)

### Files

- `sender.go` — `MessageSender` interface definition
- `pinpoint.go` — Pinpoint implementation (retry logic, client setup, MsgPre/MsgPost wrapping)
- `templates.go` — Go `text/template` definitions for all user-facing messages
- `profanity.go` — `CheckProfanity(text string) string` standalone function

### Interface

```go
type MessageSender interface {
    SendMessage(ctx context.Context, to string, body string) error
}
```

### Pinpoint Implementation

Handles all infrastructure concerns:
- Wrapping messages with `MsgPre` / `MsgPost`
- Origination identity from config (passed at construction)
- Timeout from config (passed at construction)
- Throttle retry logic (3 attempts, 500ms sleep on ThrottlingException)
- `IsAwsLocal()` short-circuit for local dev

### Templates

Move from string constants with `PLACEHOLDER` string replacement to Go `text/template`:

```go
var PrayerIntroTmpl = template.Must(template.New("prayerIntro").Parse(
    "Hello! Please pray for {{.Name}}:\n\n"))
```

Services call a `Render(tmpl, data)` helper function which returns a rendered string.

All current message constants that don't use `PLACEHOLDER` stay as plain string constants — no need to template everything.

### Profanity

`CheckProfanity(text string) string` becomes a standalone package-level function. Services call `messaging.CheckProfanity(msg.Body)` directly. The profanity detector whitelist configuration stays in this function.

### What This Eliminates

- AWS SDK types (`pinpointsmsvoicev2.SendTextMessageInput`) leaking into services
- `Member.SendMessage()` method — services call `sender.SendMessage(ctx, member.Phone, body)` directly
- Scattered `strings.Replace(..., "PLACEHOLDER", ...)` calls throughout business logic

---

## Layer 4: Config (`internal/config/`)

### File

- `config.go` — Config struct definition + `Load()` function

### Struct

```go
type Config struct {
    AWS                   AWSConfig
    IntercessorsPerPrayer int
    PrayerReminderHours   int
}

type AWSConfig struct {
    Region  string
    Backoff int
    Retry   int
    DB      DBConfig
    SMS     SMSConfig
}

type DBConfig struct {
    Timeout                int
    MemberTable            string
    ActivePrayerTable      string
    QueuedPrayerTable      string
    BlockedPhonesTable     string
    IntercessorPhonesTable string
}

type SMSConfig struct {
    PhonePool string
    Timeout   int
}
```

### Load Function

`Load()` still uses Viper under the hood to read env vars and apply defaults, but returns a `Config` struct. Viper is fully contained to this one function — nothing else in the codebase imports `github.com/spf13/viper`.

### How It Flows

Lambda entry points call `config.Load()` once, then pass `cfg` (or relevant sub-sections) into repository constructors, messaging constructors, and service constructors.

### Test Benefit

Tests construct a `Config{}` literal with exactly the values they need. No `InitConfig()` call, no env var setup, no Viper state leaking between tests.

---

## Layer 5: Service (`internal/service/`)

### Files

- `member.go` — `MemberService` (signup, deletion, help)
- `prayer.go` — `PrayerService` (requests, completion, queuing, reminders, intercessor finding)
- `admin.go` — `AdminService` (blocking users)
- `router.go` — Routes incoming messages to the right service

### Service Structs

```go
type MemberService struct {
    members      repository.MemberRepository
    intercessors repository.IntercessorPhonesRepository
    prayers      repository.PrayerRepository
    sender       messaging.MessageSender
    cfg          config.Config
}

type PrayerService struct {
    members      repository.MemberRepository
    intercessors repository.IntercessorPhonesRepository
    prayers      repository.PrayerRepository
    blocked      repository.BlockedPhonesRepository
    sender       messaging.MessageSender
    cfg          config.Config
}

type AdminService struct {
    members      repository.MemberRepository
    blocked      repository.BlockedPhonesRepository
    intercessors repository.IntercessorPhonesRepository
    prayers      repository.PrayerRepository
    sender       messaging.MessageSender
    memberSvc    *MemberService
}
```

### Router

Replaces the current `MainFlow` switch statement. Thin orchestrator:

```go
type Router struct {
    members    repository.MemberRepository
    blocked    repository.BlockedPhonesRepository
    memberSvc  *MemberService
    prayerSvc  *PrayerService
    adminSvc   *AdminService
}

func (r *Router) Handle(ctx context.Context, msg domain.TextMessage) error {
    // 1. Lookup member
    // 2. Check blocked phones
    // 3. Route to appropriate service method based on message content + member state
}
```

### What Moves Where

| Current Location | New Location |
|---|---|
| `signUp*` functions | `MemberService.SignUp()` with internal step routing |
| `memberDelete`, `removeIntercessor`, `moveActivePrayer` | `MemberService.Delete()` |
| `prayerRequest`, `FindIntercessors`, `AssignPrayer`, `queuePrayer` | `PrayerService` methods |
| `completePrayer` | `PrayerService.Complete()` |
| `blockUser` | `AdminService.BlockUser()` |
| `cleanStr`, `extractPhone` | Private helpers in respective service files |
| `checkIfNameValid`, `checkIfProfanity`, `handleTriggerWords` | Private helpers in respective service files |
| `statecontroller.RunJobs` | `PrayerService.RunScheduledJobs()` |
| `statecontroller.AssignQueuedPrayers` | `PrayerService.AssignQueuedPrayers()` |
| `statecontroller.RemindActiveIntercessors` | `PrayerService.RemindActiveIntercessors()` |

### Key Design Decisions

- `AdminService` holds a reference to `MemberService` to reuse `Delete()` logic rather than duplicating it
- The `Router` only does lookup + routing — no business logic
- Lambda entry points (`cmd/prayertexter/`, `cmd/statecontroller/`) become pure wiring: load config, construct repos, construct services, construct router, call `router.Handle()` or `prayerSvc.RunScheduledJobs()`

---

## Testing Strategy

### Libraries

- `testify` — assertions (`assert`, `require`) and test suites (`suite`)
- `mockery` — generates mock implementations from interfaces

### Generated Mocks

Mocks are auto-generated for:
- `repository.MemberRepository`
- `repository.PrayerRepository`
- `repository.BlockedPhonesRepository`
- `repository.IntercessorPhonesRepository`
- `messaging.MessageSender`

### Test Structure

**Service tests** (bulk of coverage) — mock repositories and sender, test business logic:

```go
func (s *MemberServiceSuite) TestSignUpStageOne() {
    s.members.On("Get", mock.Anything, "+11234567890").Return(nil, nil)
    s.members.On("Save", mock.Anything, mock.MatchedBy(func(m *domain.Member) bool {
        return m.SetupStatus == domain.MemberSetupInProgress
    })).Return(nil)
    s.sender.On("SendMessage", mock.Anything, "+11234567890", messaging.MsgNameRequest).Return(nil)

    err := s.svc.SignUp(ctx, domain.TextMessage{Body: "pray", Phone: "+11234567890"}, domain.Member{})
    s.NoError(err)
    s.members.AssertExpectations(s.T())
    s.sender.AssertExpectations(s.T())
}
```

**Generic repository tests** — one suite verifying the DynamoDB generic implementation against a mock `DDBConnecter`. This is the only place `AttributeValue` maps appear in tests.

**Router tests** — mock services, verify routing logic dispatches correctly.

**Template tests** — verify templates render correctly with sample data.

### How This Reduces Test Duplication and Complexity

1. **Testify suites** — shared setup per service. Each suite constructs the service with fresh mocks in `SetupTest()`. No repeated boilerplate across test functions.

2. **Test helpers for common domain objects** — simple factory functions:
   ```go
   func NewTestMember(phone string) domain.Member {
       return domain.Member{
           Phone: phone, Name: "Test User",
           SetupStatus: domain.MemberSetupComplete,
           SetupStage: domain.MemberSignUpStepFinal,
       }
   }
   ```

3. **Mock at the right level** — repository mocks return domain structs directly. No more `dynamodb.GetItemOutput` wrapping, no `attributevalue.UnmarshalMap` in test assertions, no `AttributeValue` map construction.

4. **Drop manual call counting** — testify's `AssertExpectations` verifies expected calls happened. No more `ExpectedGetItemCalls: 2` fields. Use `AssertNotCalled` when needed.

5. **Delete the entire `internal/test/` package** — `common.go` (with `ValidateMembers`, `ValidatePrayers`, `ValidatePhones`, `ValidateDeleteItem`, `ValidateTxtMessage`) and hand-rolled mocks are all replaced by generated mocks + testify assertions.

### What Gets Deleted

- `internal/test/common.go` — all validation helpers
- `internal/test/mock/` — all hand-rolled mocks
- The `test.Case` struct with its verbose mock result/expected fields
