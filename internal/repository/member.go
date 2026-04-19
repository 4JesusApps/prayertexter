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
