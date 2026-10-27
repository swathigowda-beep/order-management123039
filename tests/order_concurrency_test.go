package tests

import (
	"github.com/joho/godotenv"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"order-management/config"
	"order-management/database"
	"order-management/internal/gateway"
	"order-management/internal/order"
	"order-management/internal/product"
	"order-management/middleware"

	"github.com/gin-gonic/gin"
)


func TestOrderCreationConcurrency(t *testing.T) {
    gin.SetMode(gin.TestMode)

    // Load .env from parent folder
    _ = godotenv.Load("../.env")

    cfg := config.LoadConfig()
    // ... rest of your code



	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		t.Skip("Postgres connection not established, skipping integration concurrency test")
	}

	orderRepo := order.NewRepository(db)
	prodClient := product.NewRemoteClient(cfg.ProductSvcURL)
	orderService := order.NewService(orderRepo, prodClient)
	orderCtrl := gateway.NewOrderController(orderService)

	router := gin.New()
	router.Use(middleware.RequestIDMiddleware())
	router.POST("/api/v1/orders", orderCtrl.CreateOrder)

	var wg sync.WaitGroup
	concurrentRequests := 100
	wg.Add(concurrentRequests)

	for i := 1; i <= concurrentRequests; i++ {
		go func(customerID int) {
			defer wg.Done()

			payload := map[string]interface{}{
				"customerId": customerID,
				"productId":  501,
				"quantity":   2,
			}
			body, _ := json.Marshal(payload)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/orders", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Errorf("expected status 201, got %d with response %s", w.Code, w.Body.String())
			}
		}(i)
	}

	wg.Wait()
	fmt.Printf("Successfully handled %d concurrent order creation requests without race conditions.\n", concurrentRequests)
}