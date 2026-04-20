# Architecture Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor prayertexter into a layered architecture (domain, repository, messaging, service) with clean separation of concerns, explicit dependency injection, and simplified tests using testify + mockery.

**Architecture:** Build new packages bottom-up (domain → config → repository → messaging → service → entry points), keeping old code compiling throughout. Each layer depends only on layers below it. After all new code is wired up and tested, delete old packages.

**Tech Stack:** Go 1.23, testify (assertions + suites), mockery (mock generation), text/template (message templates), AWS SDK v2 (DynamoDB, Pinpoint), Viper (contained to config.Load())

**Spec:** `docs/superpowers/specs/2026-04-19-architecture-refactor-design.md`

---

### Task 1: Add Dependencies (testify + mockery)

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Install testify**

```bash
cd /Repos/prayertexter_mshort && go get github.com/stretchr/testify
```

- [ ] **Step 2: Install mockery CLI**

```bash
go install github.com/vektra/mockery/v2@latest
```

- [ ] **Step 3: Verify installations**

```bash
cd /Repos/prayertexter_mshort && go mod tidy
mockery --version
```

Expected: mockery version prints, go mod tidy succeeds.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add testify and mockery dependencies"
```

---

### Task 2: Create Domain Layer (`internal/domain/`)

**Files:**
- Create: `internal/domain/member.go`
- Create: `internal/domain/prayer.go`
- Create: `internal/domain/phones.go`
- Create: `internal/domain/message.go`

This task creates pure data structs with no persistence, no messaging, and no Viper imports. The domain package imports nothing from this project.

- [ ] **Step 1: Create `internal/domain/member.go`**

```go
package domain

type Member struct {
	Administrator    bool
	Intercessor      bool
	Name             string
	Phone            string
	PrayerCount      int
	SetupStage       int
	SetupStatus      string
	WeeklyPrayerDate string
	WeeklyPrayerLimit int
}

const (
	MemberSetupInProgress = "IN PROGRESS"
	MemberSetupComplete   = "COMPLETE"
	MemberSignUpStepOne   = 1
	MemberSignUpStepTwo   = 2
	MemberSignUpStepThree = 3
	MemberSignUpStepFinal = 99
)
```

- [ ] **Step 2: Create `internal/domain/prayer.go`**

```go
package domain

type Prayer struct {
	Intercessor      Member
	IntercessorPhone string
	ReminderCount    int
	ReminderDate     string
	Request          string
	Requestor        Member
}
```

- [ ] **Step 3: Create `internal/domain/phones.go`**

```go
package domain

import (
	"log/slog"
	"math/rand/v2"
	"slices"
)

type BlockedPhones struct {
	Key    string
	Phones []string
}

type IntercessorPhones struct {
	Key    string
	Phones []string
}

func (b *BlockedPhones) AddPhone(phone string) {
	if slices.Contains(b.Phones, phone) {
		return
	}
	b.Phones = append(b.Phones, phone)
}

func (b *BlockedPhones) RemovePhone(phone string) {
	removeItem(&b.Phones, phone)
}

func (i *IntercessorPhones) AddPhone(phone string) {
	if slices.Contains(i.Phones, phone) {
		return
	}
	i.Phones = append(i.Phones, phone)
}

func (i *IntercessorPhones) RemovePhone(phone string) {
	removeItem(&i.Phones, phone)
}

func (i *IntercessorPhones) GenRandPhones(intercessorsPerPrayer int) []string {
	if len(i.Phones) == 0 {
		slog.Warn("unable to generate phones, phone list is empty")
		return nil
	}

	if len(i.Phones) <= intercessorsPerPrayer {
		result := make([]string, len(i.Phones))
		copy(result, i.Phones)
		return result
	}

	var selectedPhones []string
	for len(selectedPhones) < intercessorsPerPrayer {
		phone := i.Phones[rand.IntN(len(i.Phones))] //nolint:gosec
		if slices.Contains(selectedPhones, phone) {
			continue
		}
		selectedPhones = append(selectedPhones, phone)
	}

	return selectedPhones
}

func removeItem[T comparable](items *[]T, target T) {
	slice := *items
	var newItems []T
	for _, v := range slice {
		if v != target {
			newItems = append(newItems, v)
		}
	}
	*items = newItems
}
```

Note: `removeItem` is duplicated from `utility.RemoveItem` intentionally — the domain package must not import any project packages. The utility version will be removed when old code is deleted.

- [ ] **Step 4: Create `internal/domain/message.go`**

```go
package domain

type TextMessage struct {
	Body  string `json:"messageBody"`
	Phone string `json:"originationNumber"`
}
```

- [ ] **Step 5: Verify compilation**

```bash
cd /Repos/prayertexter_mshort && go build ./internal/domain/...
```

Expected: compiles with no errors.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/
git commit -m "feat: add domain layer with pure data structs"
```

---

### Task 3: Refactor Config to Struct-Based DI (`internal/config/`)

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Rewrite `internal/config/config.go`**

Keep the existing `InitConfig()` function temporarily (old code still calls it), but add the new `Config` struct and `Load()` function alongside it.

```go
package config

import (
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
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

// Load initializes Viper and returns a Config struct.
// Viper is fully contained here — no other package should import it.
func Load() Config {
	initViper()

	return Config{
		AWS: AWSConfig{
			Region:  viper.GetString("conf.aws.region"),
			Backoff: viper.GetInt("conf.aws.backoff"),
			Retry:   viper.GetInt("conf.aws.retry"),
			DB: DBConfig{
				Timeout:                viper.GetInt("conf.aws.db.timeout"),
				MemberTable:            viper.GetString("conf.aws.db.member.table"),
				ActivePrayerTable:      viper.GetString("conf.aws.db.prayer.activetable"),
				QueuedPrayerTable:      viper.GetString("conf.aws.db.prayer.queuetable"),
				BlockedPhonesTable:     viper.GetString("conf.aws.db.blockedphones.table"),
				IntercessorPhonesTable: viper.GetString("conf.aws.db.intercessorphones.table"),
			},
			SMS: SMSConfig{
				PhonePool: viper.GetString("conf.aws.sms.phonepool"),
				Timeout:   viper.GetInt("conf.aws.sms.timeout"),
			},
		},
		IntercessorsPerPrayer: viper.GetInt("conf.intercessorsperprayer"),
		PrayerReminderHours:   viper.GetInt("conf.prayerreminderhours"),
	}
}

func initViper() {
	defaults := map[string]any{
		"aws": map[string]any{
			"region":  "us-west-1",
			"backoff": 10,
			"retry":   5,
			"db": map[string]any{
				"timeout": 60,
				"blockedphones": map[string]any{
					"table": "General",
				},
				"intercessorphones": map[string]any{
					"table": "General",
				},
				"member": map[string]any{
					"table": "Member",
				},
				"prayer": map[string]any{
					"activetable": "ActivePrayer",
					"queuetable":  "QueuedPrayer",
				},
			},
			"sms": map[string]any{
				"phonepool": "dummy",
				"timeout":   60,
			},
		},
		"intercessorsperprayer": 2,
		"prayerreminderhours":   3,
	}

	viper.SetDefault("conf", defaults)
	viper.SetEnvPrefix("pray")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
}

// InitConfig is the legacy config initializer. Kept for backward compatibility
// during refactor. Will be removed when old packages are deleted.
func InitConfig() {
	initViper()
}
```

- [ ] **Step 2: Verify all existing tests still pass**

```bash
cd /Repos/prayertexter_mshort && go test ./...
```

Expected: all existing tests pass (we only added code, changed nothing).

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat: add Config struct and Load() alongside legacy InitConfig"
```

---

### Task 4: Create Repository Layer (`internal/repository/`)

**Files:**
- Create: `internal/repository/repository.go`
- Create: `internal/repository/dynamodb.go`
- Create: `internal/repository/member.go`
- Create: `internal/repository/prayer.go`
- Create: `internal/repository/phones.go`

- [ ] **Step 1: Create `internal/repository/repository.go` with generic interface**

```go
package repository

import "context"

type Repository[T any] interface {
	Get(ctx context.Context, key string) (*T, error)
	Save(ctx context.Context, item *T) error
	Delete(ctx context.Context, key string) error
}
```

- [ ] **Step 2: Create `internal/repository/dynamodb.go` with generic DynamoDB implementation**

This replaces `internal/db/dynamodb.go`. It implements `Repository[T]` using the AWS DynamoDB SDK.

```go
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DDBClient interface {
	GetItem(ctx context.Context, input *dynamodb.GetItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	DeleteItem(ctx context.Context, input *dynamodb.DeleteItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
}

// DynamoDBRepository is a generic Repository implementation backed by DynamoDB.
type DynamoDBRepository[T any] struct {
	client   DDBClient
	table    string
	keyField string
	timeout  int
}

func NewDynamoDBRepository[T any](client DDBClient, table, keyField string, timeout int) *DynamoDBRepository[T] {
	return &DynamoDBRepository[T]{
		client:   client,
		table:    table,
		keyField: keyField,
		timeout:  timeout,
	}
}

func (r *DynamoDBRepository[T]) Get(ctx context.Context, key string) (*T, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	input := &dynamodb.GetItemInput{
		TableName: &r.table,
		Key: map[string]types.AttributeValue{
			r.keyField: &types.AttributeValueMemberS{Value: key},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	resp, err := r.client.GetItem(ctx, input)
	if err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to get item from table %s", r.table))
	}

	var item T
	if err = attributevalue.UnmarshalMap(resp.Item, &item); err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to unmarshal item from table %s", r.table))
	}

	return &item, nil
}

func (r *DynamoDBRepository[T]) Save(ctx context.Context, item *T) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return utility.WrapError(err, fmt.Sprintf("failed to marshal item for table %s", r.table))
	}

	input := &dynamodb.PutItemInput{
		TableName:              &r.table,
		Item:                   av,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	_, err = r.client.PutItem(ctx, input)
	return utility.WrapError(err, fmt.Sprintf("failed to put item in table %s", r.table))
}

func (r *DynamoDBRepository[T]) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	input := &dynamodb.DeleteItemInput{
		TableName: &r.table,
		Key: map[string]types.AttributeValue{
			r.keyField: &types.AttributeValueMemberS{Value: key},
		},
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	_, err := r.client.DeleteItem(ctx, input)
	return utility.WrapError(err, fmt.Sprintf("failed to delete item from table %s", r.table))
}

func (r *DynamoDBRepository[T]) GetAll(ctx context.Context) ([]T, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.timeout)*time.Second)
	defer cancel()

	input := &dynamodb.ScanInput{
		TableName:              &r.table,
		ReturnConsumedCapacity: types.ReturnConsumedCapacityNone,
	}

	resp, err := r.client.Scan(ctx, input)
	if err != nil {
		return nil, utility.WrapError(err, fmt.Sprintf("failed to scan table %s", r.table))
	}

	items := make([]T, 0, len(resp.Items))
	for _, item := range resp.Items {
		var obj T
		if err = attributevalue.UnmarshalMap(item, &obj); err != nil {
			return nil, utility.WrapError(err, fmt.Sprintf("failed to unmarshal item from table %s", r.table))
		}
		items = append(items, obj)
	}

	return items, nil
}
```

- [ ] **Step 3: Create `internal/repository/member.go` with MemberRepository interface**

```go
package repository

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/domain"
)

type MemberRepository interface {
	Get(ctx context.Context, phone string) (*domain.Member, error)
	Save(ctx context.Context, member *domain.Member) error
	Delete(ctx context.Context, phone string) error
	Exists(ctx context.Context, phone string) (bool, error)
}

type memberRepository struct {
	repo *DynamoDBRepository[domain.Member]
}

func NewMemberRepository(client DDBClient, table string, timeout int) MemberRepository {
	return &memberRepository{
		repo: NewDynamoDBRepository[domain.Member](client, table, "Phone", timeout),
	}
}

func (r *memberRepository) Get(ctx context.Context, phone string) (*domain.Member, error) {
	return r.repo.Get(ctx, phone)
}

func (r *memberRepository) Save(ctx context.Context, member *domain.Member) error {
	return r.repo.Save(ctx, member)
}

func (r *memberRepository) Delete(ctx context.Context, phone string) error {
	return r.repo.Delete(ctx, phone)
}

func (r *memberRepository) Exists(ctx context.Context, phone string) (bool, error) {
	mem, err := r.repo.Get(ctx, phone)
	if err != nil {
		return false, err
	}
	return mem.SetupStatus != "", nil
}
```

- [ ] **Step 4: Create `internal/repository/prayer.go` with PrayerRepository interface**

```go
package repository

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/domain"
)

type PrayerRepository interface {
	Get(ctx context.Context, key string, queued bool) (*domain.Prayer, error)
	Save(ctx context.Context, prayer *domain.Prayer, queued bool) error
	Delete(ctx context.Context, key string, queued bool) error
	Exists(ctx context.Context, phone string) (bool, error)
	GetAll(ctx context.Context, queued bool) ([]domain.Prayer, error)
}

type prayerRepository struct {
	activeRepo *DynamoDBRepository[domain.Prayer]
	queuedRepo *DynamoDBRepository[domain.Prayer]
}

func NewPrayerRepository(client DDBClient, activeTable, queuedTable string, timeout int) PrayerRepository {
	return &prayerRepository{
		activeRepo: NewDynamoDBRepository[domain.Prayer](client, activeTable, "IntercessorPhone", timeout),
		queuedRepo: NewDynamoDBRepository[domain.Prayer](client, queuedTable, "IntercessorPhone", timeout),
	}
}

func (r *prayerRepository) selectRepo(queued bool) *DynamoDBRepository[domain.Prayer] {
	if queued {
		return r.queuedRepo
	}
	return r.activeRepo
}

func (r *prayerRepository) Get(ctx context.Context, key string, queued bool) (*domain.Prayer, error) {
	return r.selectRepo(queued).Get(ctx, key)
}

func (r *prayerRepository) Save(ctx context.Context, prayer *domain.Prayer, queued bool) error {
	return r.selectRepo(queued).Save(ctx, prayer)
}

func (r *prayerRepository) Delete(ctx context.Context, key string, queued bool) error {
	return r.selectRepo(queued).Delete(ctx, key)
}

func (r *prayerRepository) Exists(ctx context.Context, phone string) (bool, error) {
	pryr, err := r.activeRepo.Get(ctx, phone)
	if err != nil {
		return false, err
	}
	return pryr.Request != "", nil
}

func (r *prayerRepository) GetAll(ctx context.Context, queued bool) ([]domain.Prayer, error) {
	return r.selectRepo(queued).GetAll(ctx)
}
```

- [ ] **Step 5: Create `internal/repository/phones.go` with phone repository interfaces**

```go
package repository

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/domain"
)

const (
	phonesKeyField         = "Key"
	blockedPhonesKeyValue  = "BlockedPhones"
	intercessorPhonesKeyValue = "IntercessorPhones"
)

type BlockedPhonesRepository interface {
	Get(ctx context.Context) (*domain.BlockedPhones, error)
	Save(ctx context.Context, phones *domain.BlockedPhones) error
}

type IntercessorPhonesRepository interface {
	Get(ctx context.Context) (*domain.IntercessorPhones, error)
	Save(ctx context.Context, phones *domain.IntercessorPhones) error
}

type blockedPhonesRepository struct {
	repo *DynamoDBRepository[domain.BlockedPhones]
}

func NewBlockedPhonesRepository(client DDBClient, table string, timeout int) BlockedPhonesRepository {
	return &blockedPhonesRepository{
		repo: NewDynamoDBRepository[domain.BlockedPhones](client, table, phonesKeyField, timeout),
	}
}

func (r *blockedPhonesRepository) Get(ctx context.Context) (*domain.BlockedPhones, error) {
	return r.repo.Get(ctx, blockedPhonesKeyValue)
}

func (r *blockedPhonesRepository) Save(ctx context.Context, phones *domain.BlockedPhones) error {
	phones.Key = blockedPhonesKeyValue
	return r.repo.Save(ctx, phones)
}

type intercessorPhonesRepository struct {
	repo *DynamoDBRepository[domain.IntercessorPhones]
}

func NewIntercessorPhonesRepository(client DDBClient, table string, timeout int) IntercessorPhonesRepository {
	return &intercessorPhonesRepository{
		repo: NewDynamoDBRepository[domain.IntercessorPhones](client, table, phonesKeyField, timeout),
	}
}

func (r *intercessorPhonesRepository) Get(ctx context.Context) (*domain.IntercessorPhones, error) {
	return r.repo.Get(ctx, intercessorPhonesKeyValue)
}

func (r *intercessorPhonesRepository) Save(ctx context.Context, phones *domain.IntercessorPhones) error {
	phones.Key = intercessorPhonesKeyValue
	return r.repo.Save(ctx, phones)
}
```

- [ ] **Step 6: Verify compilation**

```bash
cd /Repos/prayertexter_mshort && go build ./internal/repository/...
```

Expected: compiles with no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/repository/
git commit -m "feat: add repository layer with generic DynamoDB implementation"
```

---

### Task 5: Create Repository Layer Tests

**Files:**
- Create: `internal/repository/dynamodb_test.go`

This is the only place in the new test suite that uses `AttributeValue` maps. It verifies the generic DynamoDB repository implementation against a mock `DDBClient`.

- [ ] **Step 1: Generate mock for DDBClient**

Create a `.mockery.yaml` config file at project root:

```yaml
with-expecter: true
dir: "internal/mocks/{{.InterfaceDirRelative}}"
outpkg: "mocks"
packages:
  github.com/4JesusApps/prayertexter/internal/repository:
    interfaces:
      DDBClient:
      MemberRepository:
      PrayerRepository:
      BlockedPhonesRepository:
      IntercessorPhonesRepository:
  github.com/4JesusApps/prayertexter/internal/messaging:
    interfaces:
      MessageSender:
```

```bash
cd /Repos/prayertexter_mshort && mockery
```

Note: The `MessageSender` interface doesn't exist yet — mockery will skip it. We'll regenerate after Task 6.

- [ ] **Step 2: Write `internal/repository/dynamodb_test.go`**

```go
package repository_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type DynamoDBRepoSuite struct {
	suite.Suite
	client *repomocks.DDBClient
	repo   *repository.DynamoDBRepository[domain.Member]
	ctx    context.Context
}

func (s *DynamoDBRepoSuite) SetupTest() {
	s.client = repomocks.NewDDBClient(s.T())
	s.repo = repository.NewDynamoDBRepository[domain.Member](s.client, "Member", "Phone", 60)
	s.ctx = context.Background()
}

func (s *DynamoDBRepoSuite) TestGet_Success() {
	av, _ := attributevalue.MarshalMap(&domain.Member{
		Phone: "+11234567890",
		Name:  "John Doe",
	})

	s.client.EXPECT().
		GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{Item: av}, nil)

	mem, err := s.repo.Get(s.ctx, "+11234567890")
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "John Doe", mem.Name)
	assert.Equal(s.T(), "+11234567890", mem.Phone)
}

func (s *DynamoDBRepoSuite) TestGet_NotFound() {
	s.client.EXPECT().
		GetItem(mock.Anything, mock.Anything).
		Return(&dynamodb.GetItemOutput{}, nil)

	mem, err := s.repo.Get(s.ctx, "+10000000000")
	require.NoError(s.T(), err)
	assert.Empty(s.T(), mem.Phone)
}

func (s *DynamoDBRepoSuite) TestSave_Success() {
	s.client.EXPECT().
		PutItem(mock.Anything, mock.Anything).
		Return(&dynamodb.PutItemOutput{}, nil)

	mem := &domain.Member{Phone: "+11234567890", Name: "Jane"}
	err := s.repo.Save(s.ctx, mem)
	require.NoError(s.T(), err)
}

func (s *DynamoDBRepoSuite) TestDelete_Success() {
	s.client.EXPECT().
		DeleteItem(mock.Anything, mock.Anything).
		Return(&dynamodb.DeleteItemOutput{}, nil)

	err := s.repo.Delete(s.ctx, "+11234567890")
	require.NoError(s.T(), err)
}

func (s *DynamoDBRepoSuite) TestGetAll_Success() {
	mem1, _ := attributevalue.MarshalMap(&domain.Member{Phone: "+11111111111", Name: "A"})
	mem2, _ := attributevalue.MarshalMap(&domain.Member{Phone: "+12222222222", Name: "B"})

	s.client.EXPECT().
		Scan(mock.Anything, mock.Anything).
		Return(&dynamodb.ScanOutput{
			Items: []map[string]types.AttributeValue{mem1, mem2},
		}, nil)

	members, err := s.repo.GetAll(s.ctx)
	require.NoError(s.T(), err)
	assert.Len(s.T(), members, 2)
	assert.Equal(s.T(), "A", members[0].Name)
	assert.Equal(s.T(), "B", members[1].Name)
}

func TestDynamoDBRepoSuite(t *testing.T) {
	suite.Run(t, new(DynamoDBRepoSuite))
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Repos/prayertexter_mshort && go test ./internal/repository/... -v
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add .mockery.yaml internal/repository/dynamodb_test.go internal/mocks/
git commit -m "test: add repository layer tests with generated mocks"
```

---

### Task 6: Create Messaging Layer (`internal/messaging/`)

**Files:**
- Create: `internal/messaging/sender.go`
- Create: `internal/messaging/pinpoint.go`
- Create: `internal/messaging/templates.go`
- Create: `internal/messaging/profanity.go`

The existing `internal/messaging/` files (`textmessage.go`, `messages.go`) remain for now — old code still uses them. We add new files alongside.

- [ ] **Step 1: Create `internal/messaging/sender.go` with MessageSender interface**

```go
package messaging

import "context"

type MessageSender interface {
	SendMessage(ctx context.Context, to string, body string) error
}
```

- [ ] **Step 2: Create `internal/messaging/pinpoint.go` with Pinpoint implementation**

```go
package messaging

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2/types"
	"github.com/aws/smithy-go"
)

type PinpointClient interface {
	SendTextMessage(ctx context.Context, params *pinpointsmsvoicev2.SendTextMessageInput,
		optFns ...func(*pinpointsmsvoicev2.Options)) (*pinpointsmsvoicev2.SendTextMessageOutput, error)
}

type PinpointSender struct {
	client    PinpointClient
	phonePool string
	timeout   int
}

func NewPinpointSender(client PinpointClient, phonePool string, timeout int) *PinpointSender {
	return &PinpointSender{
		client:    client,
		phonePool: phonePool,
		timeout:   timeout,
	}
}

func (s *PinpointSender) SendMessage(ctx context.Context, to string, body string) error {
	wrappedBody := MsgPre + body + "\n\n" + MsgPost

	if utility.IsAwsLocal() {
		slog.InfoContext(ctx, "sent text message (local)", "phone", to, "body", wrappedBody)
		return nil
	}

	input := &pinpointsmsvoicev2.SendTextMessageInput{
		DestinationPhoneNumber: aws.String(to),
		MessageBody:            aws.String(wrappedBody),
		MessageType:            types.MessageTypeTransactional,
		OriginationIdentity:    aws.String(s.phonePool),
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(s.timeout)*time.Second)
	defer cancel()

	const maxAttempts = 3
	const sleepDuration = 500
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := s.client.SendTextMessage(ctx, input)
		if err == nil {
			return nil
		}
		lastErr = err

		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ThrottlingException" && attempt < maxAttempts {
			slog.WarnContext(ctx, "throttled by Pinpoint, retrying", "attempt", attempt, "phone", to)
			time.Sleep(sleepDuration * time.Millisecond)
			continue
		}

		break
	}

	return utility.LogAndWrapError(ctx, lastErr, "failed to send text message", "phone", to, "msg", body)
}
```

- [ ] **Step 3: Create `internal/messaging/templates.go`**

```go
package messaging

import (
	"bytes"
	"text/template"
)

var (
	PrayerIntroTmpl = template.Must(template.New("prayerIntro").Parse(
		"Hello! Please pray for {{.Name}}:\n\n"))

	ProfanityDetectedTmpl = template.Must(template.New("profanity").Parse(
		"There was profanity found in your message:\n\n{{.Word}}\n\nPlease try again"))

	PrayerConfirmationTmpl = template.Must(template.New("prayerConfirmation").Parse(
		"You're prayer request has been prayed for by {{.Name}}."))

	PrayerReminderTmpl = template.Must(template.New("prayerReminder").Parse(
		"This is a friendly reminder to pray for {{.Name}}:\n\n"))
)

func Render(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
```

- [ ] **Step 4: Create `internal/messaging/profanity.go`**

```go
package messaging

import (
	goaway "github.com/TwiN/go-away"
)

func CheckProfanity(text string) string {
	profanityDetector := goaway.NewProfanityDetector().WithSanitizeSpaces(false)
	removedWords := []string{"jerk", "ass", "butt"}
	profanities := &goaway.DefaultProfanities

	for _, word := range removedWords {
		removeFromSlice(profanities, word)
	}

	return profanityDetector.ExtractProfanity(text)
}

func removeFromSlice(items *[]string, target string) {
	var result []string
	for _, v := range *items {
		if v != target {
			result = append(result, v)
		}
	}
	*items = result
}
```

- [ ] **Step 5: Regenerate mocks (now includes MessageSender)**

```bash
cd /Repos/prayertexter_mshort && mockery
```

- [ ] **Step 6: Write template tests in `internal/messaging/templates_test.go`**

```go
package messaging_test

import (
	"testing"

	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPrayerIntro(t *testing.T) {
	result, err := messaging.Render(messaging.PrayerIntroTmpl, struct{ Name string }{"John"})
	require.NoError(t, err)
	assert.Equal(t, "Hello! Please pray for John:\n\n", result)
}

func TestRenderProfanityDetected(t *testing.T) {
	result, err := messaging.Render(messaging.ProfanityDetectedTmpl, struct{ Word string }{"badword"})
	require.NoError(t, err)
	assert.Contains(t, result, "badword")
}

func TestRenderPrayerConfirmation(t *testing.T) {
	result, err := messaging.Render(messaging.PrayerConfirmationTmpl, struct{ Name string }{"Jane"})
	require.NoError(t, err)
	assert.Contains(t, result, "Jane")
}

func TestRenderPrayerReminder(t *testing.T) {
	result, err := messaging.Render(messaging.PrayerReminderTmpl, struct{ Name string }{"Bob"})
	require.NoError(t, err)
	assert.Contains(t, result, "Bob")
}
```

- [ ] **Step 7: Verify compilation and tests**

```bash
cd /Repos/prayertexter_mshort && go test ./internal/messaging/... -v
```

Expected: all tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/messaging/sender.go internal/messaging/pinpoint.go internal/messaging/templates.go internal/messaging/profanity.go internal/messaging/templates_test.go internal/mocks/
git commit -m "feat: add messaging layer with MessageSender interface, Pinpoint impl, and templates"
```

---

### Task 7: Create Service Layer — MemberService (`internal/service/member.go`)

**Files:**
- Create: `internal/service/member.go`
- Create: `internal/service/member_test.go`

- [ ] **Step 1: Create `internal/service/member.go`**

```go
package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

type MemberService struct {
	members      repository.MemberRepository
	intercessors repository.IntercessorPhonesRepository
	prayers      repository.PrayerRepository
	sender       messaging.MessageSender
	cfg          config.Config
}

func NewMemberService(
	members repository.MemberRepository,
	intercessors repository.IntercessorPhonesRepository,
	prayers repository.PrayerRepository,
	sender messaging.MessageSender,
	cfg config.Config,
) *MemberService {
	return &MemberService{
		members:      members,
		intercessors: intercessors,
		prayers:      prayers,
		sender:       sender,
		cfg:          cfg,
	}
}

func (s *MemberService) Help(ctx context.Context, mem domain.Member) error {
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgHelp)
}

func (s *MemberService) Delete(ctx context.Context, mem domain.Member) error {
	if err := s.members.Delete(ctx, mem.Phone); err != nil {
		return err
	}
	if mem.Intercessor {
		if err := s.removeIntercessor(ctx, mem); err != nil {
			return err
		}
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgRemoveUser)
}

func (s *MemberService) removeIntercessor(ctx context.Context, mem domain.Member) error {
	phones, err := s.intercessors.Get(ctx)
	if err != nil {
		return err
	}
	phones.RemovePhone(mem.Phone)
	if err = s.intercessors.Save(ctx, phones); err != nil {
		return err
	}
	return s.moveActivePrayer(ctx, mem)
}

func (s *MemberService) moveActivePrayer(ctx context.Context, mem domain.Member) error {
	isActive, err := s.prayers.Exists(ctx, mem.Phone)
	if err != nil {
		return err
	}
	if !isActive {
		return nil
	}

	pryr, err := s.prayers.Get(ctx, mem.Phone, false)
	if err != nil {
		return err
	}

	if err = s.prayers.Delete(ctx, mem.Phone, false); err != nil {
		return err
	}

	id, err := utility.GenerateID()
	if err != nil {
		return err
	}
	pryr.IntercessorPhone = id
	pryr.Intercessor = domain.Member{}

	return s.prayers.Save(ctx, pryr, true)
}

func (s *MemberService) SignUp(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	cleanMsg := cleanStr(msg.Body)

	switch {
	case cleanMsg == "pray":
		return s.signUpStageOne(ctx, mem)
	case mem.SetupStage == domain.MemberSignUpStepOne:
		return s.signUpStageTwo(ctx, msg, mem)
	case cleanMsg == "1" && mem.SetupStage == domain.MemberSignUpStepTwo:
		return s.signUpFinalPrayer(ctx, mem)
	case cleanMsg == "2" && mem.SetupStage == domain.MemberSignUpStepTwo:
		return s.signUpStageThree(ctx, mem)
	case mem.SetupStage == domain.MemberSignUpStepThree:
		return s.signUpFinalIntercessor(ctx, msg, mem)
	default:
		return s.signUpWrongInput(ctx, mem, msg)
	}
}

func (s *MemberService) signUpStageOne(ctx context.Context, mem domain.Member) error {
	mem.SetupStatus = domain.MemberSetupInProgress
	mem.SetupStage = domain.MemberSignUpStepOne
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgNameRequest)
}

func (s *MemberService) signUpStageTwo(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	profanity := messaging.CheckProfanity(msg.Body)
	if profanity != "" {
		rendered, err := messaging.Render(messaging.ProfanityDetectedTmpl, struct{ Word string }{profanity})
		if err != nil {
			return err
		}
		return s.sender.SendMessage(ctx, mem.Phone, rendered)
	}

	if cleanStr(msg.Body) == "2" {
		mem.Name = "Anonymous"
	} else {
		mem.Name = msg.Body
	}

	if !isNameValid(mem.Name) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgInvalidName)
	}

	mem.SetupStage = domain.MemberSignUpStepTwo
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgMemberTypeRequest)
}

func (s *MemberService) signUpFinalPrayer(ctx context.Context, mem domain.Member) error {
	mem.SetupStatus = domain.MemberSetupComplete
	mem.SetupStage = domain.MemberSignUpStepFinal
	mem.Intercessor = false
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgSignUpConfirmation
	return s.sender.SendMessage(ctx, mem.Phone, body)
}

func (s *MemberService) signUpStageThree(ctx context.Context, mem domain.Member) error {
	mem.SetupStage = domain.MemberSignUpStepThree
	mem.Intercessor = true
	if err := s.members.Save(ctx, &mem); err != nil {
		return err
	}
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerNumRequest)
}

func (s *MemberService) signUpFinalIntercessor(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	num, err := strconv.Atoi(cleanStr(msg.Body))
	if err != nil {
		return s.signUpWrongInput(ctx, mem, msg)
	}

	phones, err := s.intercessors.Get(ctx)
	if err != nil {
		return err
	}

	phones.AddPhone(mem.Phone)
	if err = s.intercessors.Save(ctx, phones); err != nil {
		return err
	}

	mem.SetupStatus = domain.MemberSetupComplete
	mem.SetupStage = domain.MemberSignUpStepFinal
	mem.WeeklyPrayerLimit = num
	mem.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
	if err = s.members.Save(ctx, &mem); err != nil {
		return err
	}

	body := messaging.MsgPrayerInstructions + "\n\n" + messaging.MsgIntercessorInstructions + "\n\n" +
		messaging.MsgSignUpConfirmation
	return s.sender.SendMessage(ctx, mem.Phone, body)
}

func (s *MemberService) signUpWrongInput(ctx context.Context, mem domain.Member, msg domain.TextMessage) error {
	slog.WarnContext(ctx, "wrong input received during sign up", "member", mem.Phone, "msg", msg)
	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgWrongInput)
}

func cleanStr(str string) string {
	var sb strings.Builder
	sb.Grow(len(str))
	for _, ch := range str {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			sb.WriteRune(unicode.ToLower(ch))
		}
	}
	return sb.String()
}

func isNameValid(name string) bool {
	letterCount := 0
	minLetters := 2

	for _, ch := range name {
		switch {
		case unicode.IsLetter(ch):
			letterCount++
		case ch == ' ':
			// Spaces are fine but don't count.
		default:
			return false
		}
	}

	return letterCount >= minLetters
}
```

- [ ] **Step 2: Write `internal/service/member_test.go`**

```go
package service_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	msgmocks "github.com/4JesusApps/prayertexter/internal/mocks/messaging"
	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type MemberServiceSuite struct {
	suite.Suite
	svc          *service.MemberService
	members      *repomocks.MemberRepository
	intercessors *repomocks.IntercessorPhonesRepository
	prayers      *repomocks.PrayerRepository
	sender       *msgmocks.MessageSender
	ctx          context.Context
}

func (s *MemberServiceSuite) SetupTest() {
	s.members = repomocks.NewMemberRepository(s.T())
	s.intercessors = repomocks.NewIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewPrayerRepository(s.T())
	s.sender = msgmocks.NewMessageSender(s.T())
	s.ctx = context.Background()
	s.svc = service.NewMemberService(s.members, s.intercessors, s.prayers, s.sender, config.Config{
		IntercessorsPerPrayer: 2,
	})
}

func (s *MemberServiceSuite) TestHelp() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgHelp).Return(nil)

	err := s.svc.Help(s.ctx, domain.Member{Phone: "+11234567890"})
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageOne() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupInProgress && m.SetupStage == domain.MemberSignUpStepOne
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNameRequest).Return(nil)

	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "pray", Phone: "+11234567890"}, domain.Member{Phone: "+11234567890"})
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageTwo_ValidName() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.Name == "John Doe" && m.SetupStage == domain.MemberSignUpStepTwo
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgMemberTypeRequest).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepOne, SetupStatus: domain.MemberSetupInProgress}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "John Doe", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpStageTwo_InvalidName() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgInvalidName).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepOne, SetupStatus: domain.MemberSetupInProgress}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "1", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestSignUpFinalPrayer() {
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupComplete && !m.Intercessor
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", mock.Anything).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStage: domain.MemberSignUpStepTwo}
	err := s.svc.SignUp(s.ctx, domain.TextMessage{Body: "1", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *MemberServiceSuite) TestDelete_NonIntercessor() {
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: false})
	s.NoError(err)
}

func (s *MemberServiceSuite) TestDelete_Intercessor_NoActivePrayer() {
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{
		Key:    "IntercessorPhones",
		Phones: []string{"+11234567890", "+19999999999"},
	}, nil)
	s.intercessors.EXPECT().Save(s.ctx, mock.MatchedBy(func(p *domain.IntercessorPhones) bool {
		return len(p.Phones) == 1 && p.Phones[0] == "+19999999999"
	})).Return(nil)
	s.prayers.EXPECT().Exists(s.ctx, "+11234567890").Return(false, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)

	err := s.svc.Delete(s.ctx, domain.Member{Phone: "+11234567890", Intercessor: true})
	s.NoError(err)
}

func TestMemberServiceSuite(t *testing.T) {
	suite.Run(t, new(MemberServiceSuite))
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Repos/prayertexter_mshort && go test ./internal/service/... -v
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/service/member.go internal/service/member_test.go
git commit -m "feat: add MemberService with signup, delete, and help logic"
```

---

### Task 8: Create Service Layer — PrayerService (`internal/service/prayer.go`)

**Files:**
- Create: `internal/service/prayer.go`
- Create: `internal/service/prayer_test.go`

- [ ] **Step 1: Create `internal/service/prayer.go`**

```go
package service

import (
	"context"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

type PrayerService struct {
	members      repository.MemberRepository
	intercessors repository.IntercessorPhonesRepository
	prayers      repository.PrayerRepository
	sender       messaging.MessageSender
	cfg          config.Config
}

func NewPrayerService(
	members repository.MemberRepository,
	intercessors repository.IntercessorPhonesRepository,
	prayers repository.PrayerRepository,
	sender messaging.MessageSender,
	cfg config.Config,
) *PrayerService {
	return &PrayerService{
		members:      members,
		intercessors: intercessors,
		prayers:      prayers,
		sender:       sender,
		cfg:          cfg,
	}
}

func (s *PrayerService) Request(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	profanity := messaging.CheckProfanity(msg.Body)
	if profanity != "" {
		rendered, err := messaging.Render(messaging.ProfanityDetectedTmpl, struct{ Word string }{profanity})
		if err != nil {
			return err
		}
		return s.sender.SendMessage(ctx, mem.Phone, rendered)
	}

	if !isRequestValid(msg) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgInvalidRequest)
	}

	handleTriggerWords(&msg, &mem)

	intercessors, err := s.FindIntercessors(ctx, mem.Phone)
	if err != nil && errors.Is(err, utility.ErrNoAvailableIntercessors) {
		slog.WarnContext(ctx, "no intercessors available", "request", msg.Body, "requestor", msg.Phone)
		return s.queuePrayer(ctx, msg, mem)
	} else if err != nil {
		return utility.WrapError(err, "failed to find intercessors")
	}

	for _, intr := range intercessors {
		pryr := domain.Prayer{
			Request:   msg.Body,
			Requestor: mem,
		}
		if err = s.AssignPrayer(ctx, pryr, intr); err != nil {
			return err
		}
	}

	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerAssigned)
}

func isRequestValid(msg domain.TextMessage) bool {
	minWords := 5
	return len(strings.Fields(msg.Body)) >= minWords
}

func handleTriggerWords(msg *domain.TextMessage, mem *domain.Member) {
	//nolint:gocritic
	switch {
	case strings.Contains(strings.ToLower(msg.Body), "#anon"):
		mem.Name = "Anonymous"
		re := regexp.MustCompile(`(?i)#anon`)
		msg.Body = strings.TrimSpace(re.ReplaceAllString(msg.Body, ""))
	}
}

func (s *PrayerService) AssignPrayer(ctx context.Context, pryr domain.Prayer, intr domain.Member) error {
	pryr.Intercessor = intr
	pryr.IntercessorPhone = intr.Phone
	if err := s.prayers.Save(ctx, &pryr, false); err != nil {
		return err
	}

	introMsg, err := messaging.Render(messaging.PrayerIntroTmpl, struct{ Name string }{pryr.Requestor.Name})
	if err != nil {
		return err
	}
	msg := introMsg + pryr.Request + "\n\n" + messaging.MsgPrayed
	if err = s.sender.SendMessage(ctx, pryr.Intercessor.Phone, msg); err != nil {
		return err
	}

	slog.InfoContext(ctx, "assigned prayer successfully")
	return nil
}

func (s *PrayerService) FindIntercessors(ctx context.Context, skipPhone string) ([]domain.Member, error) {
	allPhones, err := s.intercessors.Get(ctx)
	if err != nil {
		return nil, err
	}

	allPhones.RemovePhone(skipPhone)

	var intercessors []domain.Member

	for len(intercessors) < s.cfg.IntercessorsPerPrayer {
		randPhones := allPhones.GenRandPhones(s.cfg.IntercessorsPerPrayer)
		if randPhones == nil {
			slog.InfoContext(ctx, "there are no more intercessors left to check")
			if len(intercessors) > 0 {
				slog.InfoContext(ctx, "there is at least one intercessor found, returning this even though it is less "+
					"than the desired number of intercessors per prayer")
				return intercessors, nil
			}
			return nil, utility.ErrNoAvailableIntercessors
		}

		for _, phn := range randPhones {
			if len(intercessors) >= s.cfg.IntercessorsPerPrayer {
				return intercessors, nil
			}

			intr, err := s.processIntercessor(ctx, phn)
			if err != nil && errors.Is(err, utility.ErrIntercessorUnavailable) {
				allPhones.RemovePhone(phn)
				continue
			} else if err != nil {
				return nil, err
			}

			intercessors = append(intercessors, *intr)
			allPhones.RemovePhone(phn)
			slog.InfoContext(ctx, "found one available intercessor")
		}
	}

	return intercessors, nil
}

func (s *PrayerService) processIntercessor(ctx context.Context, phone string) (*domain.Member, error) {
	intr, err := s.members.Get(ctx, phone)
	if err != nil {
		return nil, err
	}

	isActive, err := s.prayers.Exists(ctx, intr.Phone)
	if err != nil {
		return nil, err
	}
	if isActive {
		return nil, utility.ErrIntercessorUnavailable
	}

	if intr.PrayerCount < intr.WeeklyPrayerLimit {
		intr.PrayerCount++
	} else {
		canReset, err := canResetPrayerCount(*intr)
		if err != nil {
			return nil, err
		}
		if canReset {
			intr.PrayerCount = 1
			intr.WeeklyPrayerDate = time.Now().Format(time.RFC3339)
		} else {
			return nil, utility.ErrIntercessorUnavailable
		}
	}

	if err = s.members.Save(ctx, intr); err != nil {
		return nil, err
	}
	return intr, nil
}

func canResetPrayerCount(intr domain.Member) (bool, error) {
	weekDays := 7
	dayHours := 24

	currentTime := time.Now()
	previousTime, err := time.Parse(time.RFC3339, intr.WeeklyPrayerDate)
	if err != nil {
		return false, err
	}
	diffDays := currentTime.Sub(previousTime).Hours() / float64(dayHours)
	return diffDays > float64(weekDays), nil
}

func (s *PrayerService) queuePrayer(ctx context.Context, msg domain.TextMessage, mem domain.Member) error {
	id, err := utility.GenerateID()
	if err != nil {
		return err
	}

	pryr := domain.Prayer{
		IntercessorPhone: id,
		Request:          msg.Body,
		Requestor:        mem,
	}

	if err = s.prayers.Save(ctx, &pryr, true); err != nil {
		return err
	}

	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerQueued)
}

func (s *PrayerService) Complete(ctx context.Context, mem domain.Member) error {
	pryr, err := s.prayers.Get(ctx, mem.Phone, false)
	if err != nil {
		return err
	}

	if pryr.Request == "" {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgNoActivePrayer)
	}

	if err = s.sender.SendMessage(ctx, mem.Phone, messaging.MsgPrayerThankYou); err != nil {
		return err
	}

	confirmMsg, err := messaging.Render(messaging.PrayerConfirmationTmpl, struct{ Name string }{mem.Name})
	if err != nil {
		return err
	}

	isActive, err := s.members.Exists(ctx, pryr.Requestor.Phone)
	if err != nil {
		return err
	}

	if isActive {
		if err = s.sender.SendMessage(ctx, pryr.Requestor.Phone, confirmMsg); err != nil {
			return err
		}
	} else {
		slog.WarnContext(ctx, "Skip sending message, member is not active", "recipient", pryr.Requestor.Phone,
			"body", confirmMsg)
	}

	return s.prayers.Delete(ctx, mem.Phone, false)
}

func (s *PrayerService) RunScheduledJobs(ctx context.Context) {
	if err := s.AssignQueuedPrayers(ctx); err != nil {
		utility.LogError(ctx, err, "failed job", "job", "Assign Queued Prayers")
	} else {
		slog.InfoContext(ctx, "finished job", "job", "Assign Queued Prayers")
	}

	if err := s.RemindActiveIntercessors(ctx); err != nil {
		utility.LogError(ctx, err, "failed job", "job", "Remind Intercessors with Active Prayers")
	} else {
		slog.InfoContext(ctx, "finished job", "job", "Remind Intercessors with Active Prayers")
	}
}

func (s *PrayerService) AssignQueuedPrayers(ctx context.Context) error {
	prayers, err := s.prayers.GetAll(ctx, true)
	if err != nil {
		return utility.WrapError(err, "failed to get queued prayers")
	}

	for _, pryr := range prayers {
		intercessors, err := s.FindIntercessors(ctx, pryr.Requestor.Phone)
		if err != nil && errors.Is(err, utility.ErrNoAvailableIntercessors) {
			slog.WarnContext(ctx, "no intercessors available, exiting job")
			break
		} else if err != nil {
			return utility.WrapError(err, "failed to find intercessors")
		}

		for _, intr := range intercessors {
			if err = s.AssignPrayer(ctx, pryr, intr); err != nil {
				return utility.WrapError(err, "failed to assign prayer")
			}
		}

		if err = s.prayers.Delete(ctx, pryr.IntercessorPhone, true); err != nil {
			return err
		}

		if err = s.sender.SendMessage(ctx, pryr.Requestor.Phone, messaging.MsgPrayerAssigned); err != nil {
			return err
		}
	}

	return nil
}

func (s *PrayerService) RemindActiveIntercessors(ctx context.Context) error {
	prayers, err := s.prayers.GetAll(ctx, false)
	if err != nil {
		return utility.WrapError(err, "failed to get active prayers")
	}

	currentTime := time.Now()
	for _, pryr := range prayers {
		if pryr.ReminderDate == "" {
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = s.prayers.Save(ctx, &pryr, false); err != nil {
				return err
			}
			continue
		}

		previousTime, err := time.Parse(time.RFC3339, pryr.ReminderDate)
		if err != nil {
			return utility.WrapError(err, "failed to parse time")
		}
		diffTime := currentTime.Sub(previousTime).Hours()
		if diffTime > float64(s.cfg.PrayerReminderHours) {
			pryr.ReminderCount++
			pryr.ReminderDate = currentTime.Format(time.RFC3339)
			if err = s.prayers.Save(ctx, &pryr, false); err != nil {
				return err
			}

			reminderMsg, err := messaging.Render(messaging.PrayerReminderTmpl, struct{ Name string }{pryr.Requestor.Name})
			if err != nil {
				return err
			}
			msg := reminderMsg + pryr.Request + "\n\n" + messaging.MsgPrayed
			if err = s.sender.SendMessage(ctx, pryr.Intercessor.Phone, msg); err != nil {
				return err
			}
		}
	}

	return nil
}
```

- [ ] **Step 2: Write `internal/service/prayer_test.go`**

```go
package service_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	msgmocks "github.com/4JesusApps/prayertexter/internal/mocks/messaging"
	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type PrayerServiceSuite struct {
	suite.Suite
	svc          *service.PrayerService
	members      *repomocks.MemberRepository
	intercessors *repomocks.IntercessorPhonesRepository
	prayers      *repomocks.PrayerRepository
	sender       *msgmocks.MessageSender
	ctx          context.Context
}

func (s *PrayerServiceSuite) SetupTest() {
	s.members = repomocks.NewMemberRepository(s.T())
	s.intercessors = repomocks.NewIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewPrayerRepository(s.T())
	s.sender = msgmocks.NewMessageSender(s.T())
	s.ctx = context.Background()
	s.svc = service.NewPrayerService(s.members, s.intercessors, s.prayers, s.sender, config.Config{
		IntercessorsPerPrayer: 2,
		PrayerReminderHours:  3,
	})
}

func (s *PrayerServiceSuite) TestComplete_NoActivePrayer() {
	s.prayers.EXPECT().Get(s.ctx, "+11234567890", false).Return(&domain.Prayer{}, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNoActivePrayer).Return(nil)

	err := s.svc.Complete(s.ctx, domain.Member{Phone: "+11234567890"})
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestComplete_WithActivePrayer() {
	requestor := domain.Member{Phone: "+19999999999", Name: "Requestor"}
	intercessor := domain.Member{Phone: "+11234567890", Name: "Intercessor"}

	s.prayers.EXPECT().Get(s.ctx, "+11234567890", false).Return(&domain.Prayer{
		Request:          "Please pray for me",
		Requestor:        requestor,
		IntercessorPhone: "+11234567890",
		Intercessor:      intercessor,
	}, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerThankYou).Return(nil)
	s.members.EXPECT().Exists(s.ctx, "+19999999999").Return(true, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+19999999999", mock.Anything).Return(nil)
	s.prayers.EXPECT().Delete(s.ctx, "+11234567890", false).Return(nil)

	err := s.svc.Complete(s.ctx, intercessor)
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestRequest_InvalidRequest() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgInvalidRequest).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStatus: domain.MemberSetupComplete}
	err := s.svc.Request(s.ctx, domain.TextMessage{Body: "pray", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func (s *PrayerServiceSuite) TestRequest_Queued() {
	s.intercessors.EXPECT().Get(s.ctx).Return(&domain.IntercessorPhones{Phones: []string{}}, nil)
	s.prayers.EXPECT().Save(s.ctx, mock.Anything, true).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgPrayerQueued).Return(nil)

	mem := domain.Member{Phone: "+11234567890", SetupStatus: domain.MemberSetupComplete}
	err := s.svc.Request(s.ctx, domain.TextMessage{Body: "please pray for my health and well being today", Phone: "+11234567890"}, mem)
	s.NoError(err)
}

func TestPrayerServiceSuite(t *testing.T) {
	suite.Run(t, new(PrayerServiceSuite))
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Repos/prayertexter_mshort && go test ./internal/service/... -v
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/service/prayer.go internal/service/prayer_test.go
git commit -m "feat: add PrayerService with request, completion, and scheduled jobs"
```

---

### Task 9: Create Service Layer — AdminService (`internal/service/admin.go`)

**Files:**
- Create: `internal/service/admin.go`
- Create: `internal/service/admin_test.go`

- [ ] **Step 1: Create `internal/service/admin.go`**

```go
package service

import (
	"context"
	"errors"
	"regexp"
	"slices"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

type AdminService struct {
	members      repository.MemberRepository
	blocked      repository.BlockedPhonesRepository
	sender       messaging.MessageSender
	memberSvc    *MemberService
}

func NewAdminService(
	members repository.MemberRepository,
	blocked repository.BlockedPhonesRepository,
	sender messaging.MessageSender,
	memberSvc *MemberService,
) *AdminService {
	return &AdminService{
		members:   members,
		blocked:   blocked,
		sender:    sender,
		memberSvc: memberSvc,
	}
}

func (s *AdminService) BlockUser(ctx context.Context, msg domain.TextMessage, mem domain.Member, blockedPhones *domain.BlockedPhones) error {
	if !mem.Administrator {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgUnauthorized)
	}

	phone, err := extractPhone(msg.Body)
	if errors.Is(err, utility.ErrInvalidPhone) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgInvalidPhone)
	}

	phone = "+1" + phone
	if slices.Contains(blockedPhones.Phones, phone) {
		return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgUserAlreadyBlocked)
	}

	blockedPhones.AddPhone(phone)
	if err = s.blocked.Save(ctx, blockedPhones); err != nil {
		return err
	}

	blockedUser, err := s.members.Get(ctx, phone)
	if err != nil {
		return err
	}

	if err = s.memberSvc.Delete(ctx, *blockedUser); err != nil {
		return err
	}

	if err = s.sender.SendMessage(ctx, phone, messaging.MsgBlockedNotification+messaging.MsgHelp); err != nil {
		return err
	}

	return s.sender.SendMessage(ctx, mem.Phone, messaging.MsgSuccessfullyBlocked)
}

func extractPhone(msg string) (string, error) {
	var phoneRE = regexp.MustCompile(`\(?\b(\d{3})\)?[\s\-]?(\d{3})[\s\-]?(\d{4})\b`)

	matchNum := 4
	matches := phoneRE.FindStringSubmatch(msg)
	if len(matches) != matchNum {
		return "", utility.ErrInvalidPhone
	}

	return matches[1] + matches[2] + matches[3], nil
}
```

- [ ] **Step 2: Write `internal/service/admin_test.go`**

```go
package service_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	msgmocks "github.com/4JesusApps/prayertexter/internal/mocks/messaging"
	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type AdminServiceSuite struct {
	suite.Suite
	svc          *service.AdminService
	memberSvc    *service.MemberService
	members      *repomocks.MemberRepository
	blocked      *repomocks.BlockedPhonesRepository
	intercessors *repomocks.IntercessorPhonesRepository
	prayers      *repomocks.PrayerRepository
	sender       *msgmocks.MessageSender
	ctx          context.Context
}

func (s *AdminServiceSuite) SetupTest() {
	s.members = repomocks.NewMemberRepository(s.T())
	s.blocked = repomocks.NewBlockedPhonesRepository(s.T())
	s.intercessors = repomocks.NewIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewPrayerRepository(s.T())
	s.sender = msgmocks.NewMessageSender(s.T())
	s.ctx = context.Background()
	s.memberSvc = service.NewMemberService(s.members, s.intercessors, s.prayers, s.sender, config.Config{})
	s.svc = service.NewAdminService(s.members, s.blocked, s.sender, s.memberSvc)
}

func (s *AdminServiceSuite) TestBlockUser_NotAdmin() {
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgUnauthorized).Return(nil)

	mem := domain.Member{Phone: "+11234567890", Administrator: false}
	blocked := &domain.BlockedPhones{}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 777-777-7777"}, mem, blocked)
	s.NoError(err)
}

func (s *AdminServiceSuite) TestBlockUser_InvalidPhone() {
	s.sender.EXPECT().SendMessage(s.ctx, "+17777777777", messaging.MsgInvalidPhone).Return(nil)

	mem := domain.Member{Phone: "+17777777777", Administrator: true}
	blocked := &domain.BlockedPhones{}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 123"}, mem, blocked)
	s.NoError(err)
}

func (s *AdminServiceSuite) TestBlockUser_AlreadyBlocked() {
	s.sender.EXPECT().SendMessage(s.ctx, "+17777777777", messaging.MsgUserAlreadyBlocked).Return(nil)

	mem := domain.Member{Phone: "+17777777777", Administrator: true}
	blocked := &domain.BlockedPhones{Phones: []string{"+11234567890"}}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 123-456-7890"}, mem, blocked)
	s.NoError(err)
}

func (s *AdminServiceSuite) TestBlockUser_Success_NonIntercessor() {
	s.blocked.EXPECT().Save(s.ctx, mock.Anything).Return(nil)
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{
		Phone: "+11234567890", Name: "Bad User",
	}, nil)
	s.members.EXPECT().Delete(s.ctx, "+11234567890").Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgRemoveUser).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgBlockedNotification+messaging.MsgHelp).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+17777777777", messaging.MsgSuccessfullyBlocked).Return(nil)

	mem := domain.Member{Phone: "+17777777777", Administrator: true}
	blocked := &domain.BlockedPhones{Phones: []string{"+12222222222"}}
	err := s.svc.BlockUser(s.ctx, domain.TextMessage{Body: "#block 123-456-7890"}, mem, blocked)
	s.NoError(err)
}

func TestAdminServiceSuite(t *testing.T) {
	suite.Run(t, new(AdminServiceSuite))
}
```

- [ ] **Step 3: Run tests**

```bash
cd /Repos/prayertexter_mshort && go test ./internal/service/... -v
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/service/admin.go internal/service/admin_test.go
git commit -m "feat: add AdminService with block user logic"
```

---

### Task 10: Create Router (`internal/service/router.go`)

**Files:**
- Create: `internal/service/router.go`
- Create: `internal/service/router_test.go`

- [ ] **Step 1: Create `internal/service/router.go`**

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/utility"
)

type Router struct {
	members   repository.MemberRepository
	blocked   repository.BlockedPhonesRepository
	memberSvc *MemberService
	prayerSvc *PrayerService
	adminSvc  *AdminService
}

func NewRouter(
	members repository.MemberRepository,
	blocked repository.BlockedPhonesRepository,
	memberSvc *MemberService,
	prayerSvc *PrayerService,
	adminSvc *AdminService,
) *Router {
	return &Router{
		members:   members,
		blocked:   blocked,
		memberSvc: memberSvc,
		prayerSvc: prayerSvc,
		adminSvc:  adminSvc,
	}
}

func (r *Router) Handle(ctx context.Context, msg domain.TextMessage) error {
	mem, err := r.members.Get(ctx, msg.Phone)
	if err != nil {
		return utility.LogAndWrapError(ctx, err, "failure during stage PRE", "phone", msg.Phone, "msg", msg.Body)
	}

	blockedPhones, err := r.blocked.Get(ctx)
	if err != nil {
		return utility.LogAndWrapError(ctx, err, "failure during stage PRE", "phone", msg.Phone, "msg", msg.Body)
	}

	isBlocked := slices.Contains(blockedPhones.Phones, mem.Phone)
	cleanMsg := cleanStr(msg.Body)

	var stageName string
	var stageErr error

	switch {
	case isBlocked:
		stageName = "BLOCKED USER"
		slog.WarnContext(ctx, "blocked user dropping message", "phone", mem.Phone, "msg", msg.Body)

	case strings.Contains(strings.ToLower(msg.Body), "#block"):
		stageName = "ADD BLOCKED USER"
		stageErr = r.adminSvc.BlockUser(ctx, msg, *mem, blockedPhones)

	case cleanMsg == "help":
		stageName = "HELP"
		stageErr = r.memberSvc.Help(ctx, *mem)

	case cleanMsg == "cancel" || cleanMsg == "stop":
		stageName = "MEMBER DELETE"
		stageErr = r.memberSvc.Delete(ctx, *mem)

	case cleanMsg == "pray" || mem.SetupStatus == domain.MemberSetupInProgress:
		stageName = "SIGN UP"
		stageErr = r.memberSvc.SignUp(ctx, msg, *mem)

	case mem.SetupStatus == "":
		stageName = "DROP MESSAGE"
		slog.WarnContext(ctx, "non registered user dropping message", "phone", mem.Phone, "msg", msg.Body)

	case cleanMsg == "prayed":
		stageName = "COMPLETE PRAYER"
		stageErr = r.prayerSvc.Complete(ctx, *mem)

	case mem.SetupStatus == domain.MemberSetupComplete:
		stageName = "PRAYER REQUEST"
		stageErr = r.prayerSvc.Request(ctx, msg, *mem)

	default:
		err = errors.New("unexpected text message input/member status")
		return utility.LogAndWrapError(ctx, err, "could not satisfy any required conditions", "phone", mem.Phone, "msg", msg.Body)
	}

	slog.InfoContext(ctx, fmt.Sprintf("Starting stage: %s", stageName), "phone", mem.Phone, "message", msg.Body)
	if stageErr != nil {
		return utility.LogAndWrapError(ctx, stageErr, "failure during stage "+stageName, "phone", mem.Phone, "msg", msg.Body)
	}

	return nil
}
```

- [ ] **Step 2: Write `internal/service/router_test.go`**

```go
package service_test

import (
	"context"
	"testing"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	msgmocks "github.com/4JesusApps/prayertexter/internal/mocks/messaging"
	repomocks "github.com/4JesusApps/prayertexter/internal/mocks/repository"
)

type RouterSuite struct {
	suite.Suite
	router       *service.Router
	members      *repomocks.MemberRepository
	blocked      *repomocks.BlockedPhonesRepository
	intercessors *repomocks.IntercessorPhonesRepository
	prayers      *repomocks.PrayerRepository
	sender       *msgmocks.MessageSender
	ctx          context.Context
}

func (s *RouterSuite) SetupTest() {
	s.members = repomocks.NewMemberRepository(s.T())
	s.blocked = repomocks.NewBlockedPhonesRepository(s.T())
	s.intercessors = repomocks.NewIntercessorPhonesRepository(s.T())
	s.prayers = repomocks.NewPrayerRepository(s.T())
	s.sender = msgmocks.NewMessageSender(s.T())
	s.ctx = context.Background()

	cfg := config.Config{IntercessorsPerPrayer: 2, PrayerReminderHours: 3}
	memberSvc := service.NewMemberService(s.members, s.intercessors, s.prayers, s.sender, cfg)
	prayerSvc := service.NewPrayerService(s.members, s.intercessors, s.prayers, s.sender, cfg)
	adminSvc := service.NewAdminService(s.members, s.blocked, s.sender, memberSvc)

	s.router = service.NewRouter(s.members, s.blocked, memberSvc, prayerSvc, adminSvc)
}

func (s *RouterSuite) TestRouteHelp() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{}, nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgHelp).Return(nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "HELP", Phone: "+11234567890"})
	s.NoError(err)
}

func (s *RouterSuite) TestRouteBlockedUser() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{Phones: []string{"+11234567890"}}, nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "anything", Phone: "+11234567890"})
	s.NoError(err)
}

func (s *RouterSuite) TestRouteSignUp() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{}, nil)
	s.members.EXPECT().Save(s.ctx, mock.MatchedBy(func(m *domain.Member) bool {
		return m.SetupStatus == domain.MemberSetupInProgress
	})).Return(nil)
	s.sender.EXPECT().SendMessage(s.ctx, "+11234567890", messaging.MsgNameRequest).Return(nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "pray", Phone: "+11234567890"})
	s.NoError(err)
}

func (s *RouterSuite) TestRouteDropMessage() {
	s.members.EXPECT().Get(s.ctx, "+11234567890").Return(&domain.Member{Phone: "+11234567890"}, nil)
	s.blocked.EXPECT().Get(s.ctx).Return(&domain.BlockedPhones{}, nil)

	err := s.router.Handle(s.ctx, domain.TextMessage{Body: "random text", Phone: "+11234567890"})
	s.NoError(err)
}

func TestRouterSuite(t *testing.T) {
	suite.Run(t, new(RouterSuite))
}

- [ ] **Step 3: Run tests**

```bash
cd /Repos/prayertexter_mshort && go test ./internal/service/... -v
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/service/router.go internal/service/router_test.go
git commit -m "feat: add Router to dispatch messages to services"
```

---

### Task 11: Wire Up Lambda Entry Points

**Files:**
- Modify: `cmd/prayertexter/main.go`
- Modify: `cmd/statecontroller/main.go`

- [ ] **Step 1: Update `cmd/prayertexter/main.go`**

```go
package main

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/domain"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

var version string // do not remove or modify

func handler(ctx context.Context, snsEvent events.SNSEvent) {
	slog.InfoContext(ctx, "running prayertexter", "version", version)

	if len(snsEvent.Records) > 1 {
		for _, record := range snsEvent.Records {
			slog.ErrorContext(ctx, "lambda handler: there are more than 1 SNS records! This is unexpected and only "+
				"the first record will be handled", "message", record.SNS.Message, "messageid", record.SNS.MessageID)
		}
	}

	var msg domain.TextMessage
	if err := json.Unmarshal([]byte(snsEvent.Records[0].SNS.Message), &msg); err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to unmarshal api gateway request", "error", err)
		return
	}

	cfg := config.Load()

	awsCfg, err := utility.GetAwsConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to get aws config", "error", err)
		return
	}

	ddbClnt := dynamodb.NewFromConfig(awsCfg)
	smsClnt := pinpointsmsvoicev2.NewFromConfig(awsCfg)

	members := repository.NewMemberRepository(ddbClnt, cfg.AWS.DB.MemberTable, cfg.AWS.DB.Timeout)
	prayers := repository.NewPrayerRepository(ddbClnt, cfg.AWS.DB.ActivePrayerTable, cfg.AWS.DB.QueuedPrayerTable, cfg.AWS.DB.Timeout)
	blocked := repository.NewBlockedPhonesRepository(ddbClnt, cfg.AWS.DB.BlockedPhonesTable, cfg.AWS.DB.Timeout)
	intercessors := repository.NewIntercessorPhonesRepository(ddbClnt, cfg.AWS.DB.IntercessorPhonesTable, cfg.AWS.DB.Timeout)

	sender := messaging.NewPinpointSender(smsClnt, cfg.AWS.SMS.PhonePool, cfg.AWS.SMS.Timeout)

	memberSvc := service.NewMemberService(members, intercessors, prayers, sender, cfg)
	prayerSvc := service.NewPrayerService(members, intercessors, prayers, sender, cfg)
	adminSvc := service.NewAdminService(members, blocked, sender, memberSvc)
	router := service.NewRouter(members, blocked, memberSvc, prayerSvc, adminSvc)

	if err = router.Handle(ctx, msg); err != nil {
		return
	}
}

func main() {
	lambda.Start(handler)
}
```

**Note:** The `utility.GetAwsConfig` still uses Viper internally. Update it in the next step.

- [ ] **Step 2: Update `cmd/statecontroller/main.go`**

```go
package main

import (
	"context"
	"log/slog"

	"github.com/4JesusApps/prayertexter/internal/config"
	"github.com/4JesusApps/prayertexter/internal/messaging"
	"github.com/4JesusApps/prayertexter/internal/repository"
	"github.com/4JesusApps/prayertexter/internal/service"
	"github.com/4JesusApps/prayertexter/internal/utility"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/pinpointsmsvoicev2"
)

var version string // do not remove or modify

func handler(ctx context.Context) {
	slog.InfoContext(ctx, "running statecontroller", "version", version)

	cfg := config.Load()

	awsCfg, err := utility.GetAwsConfig(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "lambda handler: failed to get aws config", "error", err)
		return
	}

	ddbClnt := dynamodb.NewFromConfig(awsCfg)
	smsClnt := pinpointsmsvoicev2.NewFromConfig(awsCfg)

	members := repository.NewMemberRepository(ddbClnt, cfg.AWS.DB.MemberTable, cfg.AWS.DB.Timeout)
	prayers := repository.NewPrayerRepository(ddbClnt, cfg.AWS.DB.ActivePrayerTable, cfg.AWS.DB.QueuedPrayerTable, cfg.AWS.DB.Timeout)
	intercessors := repository.NewIntercessorPhonesRepository(ddbClnt, cfg.AWS.DB.IntercessorPhonesTable, cfg.AWS.DB.Timeout)

	sender := messaging.NewPinpointSender(smsClnt, cfg.AWS.SMS.PhonePool, cfg.AWS.SMS.Timeout)

	prayerSvc := service.NewPrayerService(members, intercessors, prayers, sender, cfg)
	prayerSvc.RunScheduledJobs(ctx)
}

func main() {
	lambda.Start(handler)
}
```

- [ ] **Step 3: Update `internal/utility/aws.go` to accept config parameters**

Replace `GetAwsConfig` to accept parameters instead of reading Viper. Since `config.Load()` is called before this, pass the values in:

```go
package utility

import (
	"context"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
)

const (
	DefaultAwsRegion           = "us-west-1"
	DefaultAwsSvcRetryAttempts = 5
	DefaultAwsSvcMaxBackoff    = 10
)

func GetAwsConfig(ctx context.Context) (aws.Config, error) {
	region := DefaultAwsRegion
	maxRetry := DefaultAwsSvcRetryAttempts
	maxBackoff := DefaultAwsSvcMaxBackoff

	if r := os.Getenv("PRAY_CONF_AWS_REGION"); r != "" {
		region = r
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region),
		config.WithRetryer(func() aws.Retryer {
			retryer := retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = maxRetry
				o.MaxBackoff = time.Duration(maxBackoff) * time.Second
			})
			return &LoggingRetryer{delegate: retryer}
		}))

	return cfg, WrapError(err, "failed to get aws config")
}

func IsAwsLocal() bool {
	return os.Getenv("AWS_SAM_LOCAL") == "true"
}
```

Note: This removes the Viper dependency from `utility/aws.go`. The AWS config values (region, retry, backoff) are now either defaults or env vars. This is acceptable because these values are AWS infrastructure config, not application business config.

- [ ] **Step 4: Verify compilation**

```bash
cd /Repos/prayertexter_mshort && go build ./cmd/prayertexter/... && go build ./cmd/statecontroller/...
```

Expected: both compile.

- [ ] **Step 5: Commit**

```bash
git add cmd/prayertexter/main.go cmd/statecontroller/main.go internal/utility/aws.go
git commit -m "feat: wire up Lambda entry points to new service layer"
```

---

### Task 12: Delete Old Packages

**Files:**
- Delete: `internal/object/` (all files)
- Delete: `internal/db/` (all files)
- Delete: `internal/prayertexter/` (all files)
- Delete: `internal/statecontroller/` (all files)
- Delete: `internal/test/` (all files)
- Modify: `internal/messaging/textmessage.go` (delete)
- Modify: `internal/messaging/messages.go` (keep only constants still referenced)
- Modify: `internal/config/config.go` (remove `InitConfig`)
- Modify: `internal/utility/aws.go` (remove old Viper config path constants)
- Modify: `go.mod` (clean up)

- [ ] **Step 1: Delete old packages**

```bash
cd /Repos/prayertexter_mshort
rm -rf internal/object/
rm -rf internal/db/
rm -rf internal/prayertexter/
rm -rf internal/statecontroller/
rm -rf internal/test/
```

- [ ] **Step 2: Remove legacy `TextSender` interface and `SendText`/`GetSmsClient`/`CheckProfanity` from messaging**

Delete `internal/messaging/textmessage.go`.

```bash
rm internal/messaging/textmessage.go
```

- [ ] **Step 3: Clean up `internal/messaging/messages.go`**

Keep only the message constants that are still referenced by the new service layer. Remove the old config path constants (`PhonePoolConfigPath`, `TimeoutConfigPath`, `DefaultPhonePool`, `DefaultTimeout`).

Update `internal/messaging/messages.go` to:

```go
package messaging

const (
	MsgNameRequest = "Reply with your name, or 2 to stay anonymous."
	MsgInvalidName = "Sorry, that name is not valid. Please reply with a name that is at least 2 letters long and " +
		"only contains letters or spaces."
	MsgMemberTypeRequest = "Reply 1 to send prayer request, or 2 to be added to the intercessors list (to pray for " +
		"others). 2 will also allow you to send in prayer requests."
	MsgPrayerInstructions = "You are now signed up to send prayer requests! You can text them directly to this number" +
		" at any time."
	MsgPrayerNumRequest = "Reply with the number of maximum prayer texts that you are willing to receive and pray for " +
		"each week."
	MsgIntercessorInstructions = "You are now signed up to receive prayer requests. Please try to pray for the " +
		"requests as soon as you receive them. " + MsgPrayed
	MsgWrongInput         = "Incorrect input received during sign up, please try again."
	MsgSignUpConfirmation = "You have opted into PrayerTexter. Msg & data rates may apply."
	MsgRemoveUser         = "You have been removed from PrayerTexter. To sign back up, text the word pray to this " +
		"number."
)

const (
	MsgInvalidRequest = "Sorry, that request is not valid. Prayer requests must contain at least 5 words."
	MsgPrayed         = "Once you have prayed, reply with the word prayed so that the prayer can be confirmed."
	MsgPrayerQueued   = "We could not find any available intercessors. Your prayer has been added to the queue and " +
		"will get sent out as soon as someone is available."
	MsgPrayerAssigned = "Your prayer request has been sent out and assigned!"
)

const (
	MsgNoActivePrayer = "You have no active prayers to mark as prayed."
	MsgPrayerThankYou = "Thank you for praying! We let the prayer requestor know that you have prayed for them."
)

const (
	MsgUnauthorized        = "You are unauthorized to perform this action."
	MsgInvalidPhone        = "The phone number provided is invalid. Please use this format: 123-456-7890."
	MsgUserAlreadyBlocked  = "The phone number provided already exists on the block list."
	MsgSuccessfullyBlocked = "The phone number provided has been successfully added to the block list."
	MsgBlockedNotification = "You have been blocked from using PrayerTexter. If you feel this is an error, feel free " +
		"to reach out to us. "
)

const (
	MsgHelp = "To receive support, please email info@4jesusministries.com or call/text (949) 313-4375. " +
		"Thank you!"
	MsgPre  = "PrayerTexter: "
	MsgPost = "Reply HELP for help or STOP to cancel."
)
```

- [ ] **Step 4: Remove `InitConfig` from config.go**

Edit `internal/config/config.go` — remove the `InitConfig()` function and the old default constant imports. The file should only contain `Config` struct, `Load()`, and `initViper()`.

- [ ] **Step 5: Remove old Viper config path constants and imports from `internal/utility/aws.go`**

Remove the Viper config path constants (`AwsRegionConfigPath`, `AwsSvcRetryAttemptsConfigPath`, `AwsSvcMaxBackoffConfigPath`) that are no longer used.

- [ ] **Step 6: Remove unused `utility.RemoveItem` if domain has its own**

Check if `utility.RemoveItem` is still referenced. If not, remove it from `internal/utility/general.go`.

- [ ] **Step 7: Clean up go.mod**

```bash
cd /Repos/prayertexter_mshort && go mod tidy
```

- [ ] **Step 8: Verify full compilation and all new tests pass**

```bash
cd /Repos/prayertexter_mshort && go build ./... && go test ./... -v
```

Expected: everything compiles. All new tests pass. No references to deleted packages.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "refactor: delete old packages (object, db, prayertexter, statecontroller, test)"
```

---

### Task 13: Final Verification and Cleanup

**Files:**
- No new files — verification only

- [ ] **Step 1: Run full test suite**

```bash
cd /Repos/prayertexter_mshort && go test ./... -v -count=1
```

Expected: all tests pass, no cached results.

- [ ] **Step 2: Run go vet**

```bash
cd /Repos/prayertexter_mshort && go vet ./...
```

Expected: no issues.

- [ ] **Step 3: Verify no Viper imports outside config package**

```bash
cd /Repos/prayertexter_mshort && grep -r "github.com/spf13/viper" internal/ --include="*.go" -l
```

Expected: only `internal/config/config.go`.

- [ ] **Step 4: Verify no AWS SDK types in service layer**

```bash
cd /Repos/prayertexter_mshort && grep -r "aws-sdk-go-v2" internal/service/ --include="*.go" -l
```

Expected: no matches.

- [ ] **Step 5: Verify no old package imports remain**

```bash
cd /Repos/prayertexter_mshort && grep -r "internal/object\|internal/db\|internal/prayertexter\|internal/statecontroller\|internal/test" internal/ --include="*.go" -l
```

Expected: no matches.

- [ ] **Step 6: Verify dependency direction**

```bash
cd /Repos/prayertexter_mshort && grep -r "internal/service\|internal/repository" internal/domain/ --include="*.go" -l
```

Expected: no matches (domain imports nothing from the project).

- [ ] **Step 7: Build all Lambda binaries**

```bash
cd /Repos/prayertexter_mshort && go build ./cmd/prayertexter/... && go build ./cmd/statecontroller/... && go build ./cmd/announcer/...
```

Expected: all compile successfully.
