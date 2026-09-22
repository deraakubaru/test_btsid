package router

import (
	"btsid/internal/handler"
	"btsid/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter constructs and wires the Gin HTTP engine with configured routes.
func SetupRouter(authHandler *handler.AuthHandler, productHandler *handler.ProductHandler, jwtSecret string) *gin.Engine {
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
				// Public GET endpoints
				products.GET("", productHandler.GetProducts)
				products.GET("/:id", productHandler.GetProductByID)

				// Protected mutation endpoints
				if jwtSecret != "" {
					protected := products.Group("")
					protected.Use(middleware.Authenticate(jwtSecret))
					{
						protected.POST("", productHandler.CreateProduct)
						protected.PUT("/:id", productHandler.UpdateProduct)
						protected.DELETE("/:id", productHandler.DeleteProduct)
					}
				}
			}
		}
	}

	return r
}
