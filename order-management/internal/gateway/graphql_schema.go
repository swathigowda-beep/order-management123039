package gateway

import (
	"context"
	"net/http"

	"order-management/internal/order"

	"github.com/gin-gonic/gin"
	"github.com/graphql-go/graphql"
)

var orderType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Order",
	Fields: graphql.Fields{
		"id":         &graphql.Field{Type: graphql.Int},
		"customerId": &graphql.Field{Type: graphql.Int},
		"productId":  &graphql.Field{Type: graphql.Int},
		"quantity":   &graphql.Field{Type: graphql.Int},
		"totalPrice": &graphql.Field{Type: graphql.Float},
		"status":     &graphql.Field{Type: graphql.String},
	},
})

func BuildGraphQLHandler(svc order.Service) gin.HandlerFunc {
	rootQuery := graphql.NewObject(graphql.ObjectConfig{
		Name: "RootQuery",
		Fields: graphql.Fields{
			"order": &graphql.Field{
				Type: orderType,
				Args: graphql.FieldConfigArgument{
					"id": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.Int)},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					id := uint(p.Args["id"].(int))
					return svc.GetOrder(p.Context, id)
				},
			},
			"orders": &graphql.Field{
				Type: graphql.NewList(orderType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return svc.ListOrders(p.Context)
				},
			},
		},
	})

	schema, _ := graphql.NewSchema(graphql.SchemaConfig{
		Query: rootQuery,
	})

	return func(c *gin.Context) {
		var req struct {
			Query string `json:"query"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid GraphQL query payload"})
			return
		}

		ctx := context.WithValue(c.Request.Context(), "ginContext", c)
		res := graphql.Do(graphql.Params{
			Schema:        schema,
			RequestString: req.Query,
			Context:       ctx,
		})

		c.JSON(http.StatusOK, res)
	}
}