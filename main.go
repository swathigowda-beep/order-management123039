package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-management/config"
	"order-management/database"
	"order-management/internal/gateway"
	"order-management/internal/order"
	"order-management/internal/product"
	"order-management/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("Fatal: Database initialization failed: %v", err)
	}

	// Repositories & Services
	orderRepo := order.NewRepository(db)
	prodClient := product.NewRemoteClient(cfg.ProductSvcURL)
	orderService := order.NewService(orderRepo, prodClient)
	orderCtrl := gateway.NewOrderController(orderService)

	router := gin.New()
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.StructuredLoggingMiddleware())
	router.Use(middleware.PrometheusMetricsMiddleware())
	router.Use(middleware.RecoveryMiddleware())

	// Health Endpoints
	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})
	router.GET("/health/ready", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil || sqlDB.Ping() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "DOWN", "error": "Database unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "READY"})
	})

	// Prometheus Metrics Endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Mock Token Mint Endpoint for Testing Authentication
	router.POST("/api/v1/auth/token", func(c *gin.Context) {
		var req struct {
			UserID uint   `json:"user_id"`
			Role   string `json:"role"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		token, err := middleware.GenerateToken(cfg.JWTSecret, req.UserID, req.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	// Authenticated API V1 Routes
	v1 := router.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		// Customer & Admin: Create and Read Orders
		v1.POST("/orders", middleware.AuthorizeRoles("admin", "customer"), orderCtrl.CreateOrder)
		v1.GET("/orders/:id", middleware.AuthorizeRoles("admin", "customer"), orderCtrl.GetOrder)

		// Admin Only: List all orders and Update Status
		v1.GET("/orders", middleware.AuthorizeRoles("admin"), orderCtrl.ListOrders)
		v1.PUT("/orders/:id/status", middleware.AuthorizeRoles("admin"), orderCtrl.UpdateOrderStatus)
	}

	// GraphQL Gateway Endpoint
	router.POST("/graphql", middleware.AuthMiddleware(cfg.JWTSecret), gateway.BuildGraphQLHandler(orderService))

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// Graceful Shutdown with Goroutines
	go func() {
		log.Printf("Order Management API running on port %s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server listen failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server safely...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited cleanly.")
}