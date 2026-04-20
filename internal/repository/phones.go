package repository

import (
	"context"

	"github.com/4JesusApps/prayertexter/internal/domain"
)

const (
	phonesKeyField            = "Key"
	blockedPhonesKeyValue     = "BlockedPhones"
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
