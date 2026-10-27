package order

import (
	"context"

	"order-management/middleware"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, o *Order) error
	FindByID(ctx context.Context, id uint) (*Order, error)
	FindAll(ctx context.Context) ([]Order, error)
	UpdateStatus(ctx context.Context, id uint, status OrderStatus) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, o *Order) error {
	if err := r.db.WithContext(ctx).Create(o).Error; err != nil {
		middleware.DBErrorsTotal.Inc()
		return err
	}
	return nil
}

func (r *repository) FindByID(ctx context.Context, id uint) (*Order, error) {
	var o Order
	if err := r.db.WithContext(ctx).First(&o, id).Error; err != nil {
		middleware.DBErrorsTotal.Inc()
		return nil, err
	}
	return &o, nil
}

func (r *repository) FindAll(ctx context.Context) ([]Order, error) {
	var orders []Order
	if err := r.db.WithContext(ctx).Find(&orders).Error; err != nil {
		middleware.DBErrorsTotal.Inc()
		return nil, err
	}
	return orders, nil
}

func (r *repository) UpdateStatus(ctx context.Context, id uint, status OrderStatus) error {
	res := r.db.WithContext(ctx).Model(&Order{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		middleware.DBErrorsTotal.Inc()
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}