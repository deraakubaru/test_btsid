package handler

import (
	"net/http"
	"strconv"

	"btsid/internal/domain"
	"btsid/internal/service"

	"github.com/gin-gonic/gin"
)

type ProductHandler struct {
	productService *service.ProductService
}

func NewProductHandler(productService *service.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// GetProducts handles GET /api/products
func (h *ProductHandler) GetProducts(c *gin.Context) {
	search := c.Query("search")
	category := c.Query("category")
	pageStr := c.Query("page")
	limitStr := c.Query("limit")

	page := 0
	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, domain.NewErrorResponse("Invalid query parameter", "page must be a valid integer"))
			return
		}
		page = p
	}

	limit := 0
	if limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, domain.NewErrorResponse("Invalid query parameter", "limit must be a valid integer"))
			return
		}
		limit = l
	}

	products, pagination, err := h.productService.ListProducts(c.Request.Context(), search, category, page, limit)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"data":       products,
		"pagination": pagination,
	})
}

// GetProductByID handles GET /api/products/:id
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, domain.NewErrorResponse("Invalid product ID", "product ID must be a positive integer"))
		return
	}

	product, err := h.productService.GetProductByID(c.Request.Context(), id)
	if err != nil {
		RespondError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    product,
	})
}
