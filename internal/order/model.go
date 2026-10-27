package order

import "time"

type OrderStatus string

const (
	StatusCreated   OrderStatus = "CREATED"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	ID         uint        `gorm:"primaryKey" json:"id"`
	CustomerID uint        `gorm:"not null;index:idx_orders_customer" json:"customerId"`
	ProductID  uint        `gorm:"not null" json:"productId"`
	Quantity   int         `gorm:"not null" json:"quantity"`
	TotalPrice float64     `gorm:"not null" json:"totalPrice"`
	Status     OrderStatus `gorm:"type:varchar(20);not null;default:'CREATED'" json:"status"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

type CreateOrderDTO struct {
	CustomerID uint `json:"customerId" binding:"required,gt=0"`
	ProductID  uint `json:"productId" binding:"required,gt=0"`
	Quantity   int  `json:"quantity" binding:"required,gt=0"`
}

type UpdateStatusDTO struct {
	Status OrderStatus `json:"status" binding:"required,oneof=CREATED CONFIRMED CANCELLED"`
}