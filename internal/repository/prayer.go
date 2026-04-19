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
