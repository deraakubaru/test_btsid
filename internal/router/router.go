package router

import (
	"btsid/internal/handler"

	"github.com/gin-gonic/gin"
)

// SetupRouter constructs and wires the Gin HTTP engine with configured routes.
func SetupRouter(authHandler *handler.AuthHandler, productHandler *handler.ProductHandler) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api")
	{
		if authHandler != nil {
			auth := api.Group("/auth")
			{
				auth.POST("/register", authHandler.Register)
				auth.POST("/login", authHandler.Login)
			}
		}

		if productHandler != nil {
			products := api.Group("/products")
			{
				products.GET("", productHandler.GetProducts)
				products.GET("/:id", productHandler.GetProductByID)
			}
		}
	}

	return r
}
