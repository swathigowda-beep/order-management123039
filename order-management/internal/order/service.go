package order

import (
	"context"
	"fmt"
	"order-management/internal/product"
)

type Service interface {
	CreateOrder(ctx context.Context, dto CreateOrderDTO) (*Order, error)
	GetOrder(ctx context.Context, id uint) (*Order, error)
	ListOrders(ctx context.Context) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, id uint, status OrderStatus) error
}

type service struct {
	repo          Repository
	productClient product.RemoteClient
}

func NewService(repo Repository, productClient product.RemoteClient) Service {
	return &service{
		repo:          repo,
		productClient: productClient,
	}
}

func (s *service) CreateOrder(ctx context.Context, dto CreateOrderDTO) (*Order, error) {
	type priceResult struct {
		price float64
		err   error
	}

	priceChan := make(chan priceResult, 1)

	// Fetch price asynchronously with Goroutine and bounded Context
	go func() {
		defer close(priceChan)
		price, err := s.productClient.FetchProductPrice(ctx, dto.ProductID)
		priceChan <- priceResult{price: price, err: err}
	}()

	var price float64
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("order processing canceled: %w", ctx.Err())
	case res := <-priceChan:
		if res.err != nil {
			// Fallback placeholder logic for missing mock remote price endpoint
			price = 100.00
		} else {
			price = res.price
		}
	}

	ord := &Order{
		CustomerID: dto.CustomerID,
		ProductID:  dto.ProductID,
		Quantity:   dto.Quantity,
		TotalPrice: price * float64(dto.Quantity),
		Status:     StatusCreated,
	}

	if err := s.repo.Create(ctx, ord); err != nil {
		return nil, err
	}

	return ord, nil
}

func (s *service) GetOrder(ctx context.Context, id uint) (*Order, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) ListOrders(ctx context.Context) ([]Order, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) UpdateOrderStatus(ctx context.Context, id uint, status OrderStatus) error {
	return s.repo.UpdateStatus(ctx, id, status)
}