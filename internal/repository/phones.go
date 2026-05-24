package repository

import (
	"context"
	"errors"

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

type blockedPhonesRow struct {
	Key    string
	Phones []string
}

type blockedPhonesRepository struct {
	repo *DynamoDBRepository[blockedPhonesRow]
}

func NewBlockedPhonesRepository(client DDBClient, table string, timeout int) BlockedPhonesRepository {
	return &blockedPhonesRepository{
		repo: NewDynamoDBRepository[blockedPhonesRow](client, table, phonesKeyField, timeout),
	}
}

func (r *blockedPhonesRepository) Get(ctx context.Context) (*domain.BlockedPhones, error) {
	row, err := r.repo.Get(ctx, blockedPhonesKeyValue)
	if errors.Is(err, ErrNotFound) {
		return &domain.BlockedPhones{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.BlockedPhones{Phones: row.Phones}, nil
}

func (r *blockedPhonesRepository) Save(ctx context.Context, phones *domain.BlockedPhones) error {
	row := &blockedPhonesRow{Key: blockedPhonesKeyValue, Phones: phones.Phones}
	return r.repo.Save(ctx, row)
}

type intercessorPhonesRow struct {
	Key    string
	Phones []string
}

type intercessorPhonesRepository struct {
	repo *DynamoDBRepository[intercessorPhonesRow]
}

func NewIntercessorPhonesRepository(client DDBClient, table string, timeout int) IntercessorPhonesRepository {
	return &intercessorPhonesRepository{
		repo: NewDynamoDBRepository[intercessorPhonesRow](client, table, phonesKeyField, timeout),
	}
}

func (r *intercessorPhonesRepository) Get(ctx context.Context) (*domain.IntercessorPhones, error) {
	row, err := r.repo.Get(ctx, intercessorPhonesKeyValue)
	if errors.Is(err, ErrNotFound) {
		return &domain.IntercessorPhones{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.IntercessorPhones{Phones: row.Phones}, nil
}

func (r *intercessorPhonesRepository) Save(ctx context.Context, phones *domain.IntercessorPhones) error {
	row := &intercessorPhonesRow{Key: intercessorPhonesKeyValue, Phones: phones.Phones}
	return r.repo.Save(ctx, row)
}
