package gateway

import (
	"errors"
	"net/http"
	"strconv"

	"order-management/internal/order"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderController struct {
	svc order.Service
}

func NewOrderController(svc order.Service) *OrderController {
	return &OrderController{svc: svc}
}

func (ctrl *OrderController) CreateOrder(c *gin.Context) {
	var dto order.CreateOrderDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify customer ID matches authenticated user token (Customer RBAC check)
	userRole, _ := c.Get("userRole")
	userID, _ := c.Get("userID")
	if userRole == "customer" && dto.CustomerID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "customers can only create orders for their own ID"})
		return
	}

	created, err := ctrl.svc.CreateOrder(c.Request.Context(), dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (ctrl *OrderController) GetOrder(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	ord, err := ctrl.svc.GetOrder(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ord)
}

func (ctrl *OrderController) ListOrders(c *gin.Context) {
	orders, err := ctrl.svc.ListOrders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (ctrl *OrderController) UpdateOrderStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var dto order.UpdateStatusDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.svc.UpdateOrderStatus(c.Request.Context(), uint(id), dto.Status); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order status updated successfully"})
}