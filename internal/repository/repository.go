package repository

import "context"

type Repository[T any] interface {
	Get(ctx context.Context, key string) (*T, error)
	Save(ctx context.Context, item *T) error
	Delete(ctx context.Context, key string) error
}
