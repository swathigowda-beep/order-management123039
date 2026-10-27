package product

import (
	"context"
)

type Service interface {
	GetProduct(ctx context.Context, id uint) (*Product, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetProduct(ctx context.Context, id uint) (*Product, error) {
	return s.repo.FindByID(ctx, id)
}