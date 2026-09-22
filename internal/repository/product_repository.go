package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"btsid/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductRepository contract for product database operations.
type ProductRepository interface {
	GetProducts(ctx context.Context, search, category string, page, limit int) ([]*domain.Product, int64, error)
	GetProductByID(ctx context.Context, id int64) (*domain.Product, error)
	CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error)
	DeleteProduct(ctx context.Context, id int64) error
}

type PgProductRepository struct {
	pool *pgxpool.Pool
}

func NewProductRepository(pool *pgxpool.Pool) *PgProductRepository {
	return &PgProductRepository{pool: pool}
}

// GetProducts queries products with optional search and category filters, deterministic ordering, and pagination.
func (r *PgProductRepository) GetProducts(ctx context.Context, search, category string, page, limit int) ([]*domain.Product, int64, error) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if strings.TrimSpace(search) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("p.title ILIKE '%%' || $%d || '%%'", argIdx))
		args = append(args, strings.TrimSpace(search))
		argIdx++
	}

	if strings.TrimSpace(category) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(p.category) = LOWER($%d)", argIdx))
		args = append(args, strings.TrimSpace(category))
		argIdx++
	}

	whereStmt := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products p WHERE %s", whereStmt)
	var totalItems int64
	err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&totalItems)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count products: %w", err)
	}

	offset := (page - 1) * limit
	listQuery := fmt.Sprintf(`
		SELECT 
			p.id, p.title, p.price, p.description, p.category, p.images,
			p.created_by_id, p.updated_by_id, p.created_at, p.updated_at,
			u1.username AS created_by,
			u2.username AS updated_by
		FROM products p
		JOIN users u1 ON p.created_by_id = u1.id
		JOIN users u2 ON p.updated_by_id = u2.id
		WHERE %s
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $%d OFFSET $%d
	`, whereStmt, argIdx, argIdx+1)

	listArgs := append(args, limit, offset)
	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query products list: %w", err)
	}
	defer rows.Close()

	products := []*domain.Product{}
	for rows.Next() {
		var p domain.Product
		err := rows.Scan(
			&p.ID,
			&p.Title,
			&p.Price,
			&p.Description,
			&p.Category,
			&p.Images,
			&p.CreatedByID,
			&p.UpdatedByID,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.CreatedBy,
			&p.UpdatedBy,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan product row: %w", err)
		}
		products = append(products, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("row iteration error: %w", err)
	}

	return products, totalItems, nil
}

// GetProductByID fetches a single product by ID joined with users for audit usernames.
func (r *PgProductRepository) GetProductByID(ctx context.Context, id int64) (*domain.Product, error) {
	query := `
		SELECT 
			p.id, p.title, p.price, p.description, p.category, p.images,
			p.created_by_id, p.updated_by_id, p.created_at, p.updated_at,
			u1.username AS created_by,
			u2.username AS updated_by
		FROM products p
		JOIN users u1 ON p.created_by_id = u1.id
		JOIN users u2 ON p.updated_by_id = u2.id
		WHERE p.id = $1
	`

	var p domain.Product
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.Title,
		&p.Price,
		&p.Description,
		&p.Category,
		&p.Images,
		&p.CreatedByID,
		&p.UpdatedByID,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.CreatedBy,
		&p.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.NewNotFoundError("product not found")
		}
		return nil, fmt.Errorf("failed to query product by ID: %w", err)
	}

	return &p, nil
}

// CreateProduct inserts a new product record and returns the persisted product with resolved usernames.
func (r *PgProductRepository) CreateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	query := `
		INSERT INTO products (title, price, description, category, images, created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id
	`

	var id int64
	err := r.pool.QueryRow(ctx, query,
		product.Title,
		product.Price,
		product.Description,
		product.Category,
		product.Images,
		product.CreatedByID,
		product.UpdatedByID,
	).Scan(&id)

	if err != nil {
		return nil, fmt.Errorf("failed to insert product: %w", err)
	}

	return r.GetProductByID(ctx, id)
}

// UpdateProduct performs a full field update on a product, updating updated_by_id and updated_at.
func (r *PgProductRepository) UpdateProduct(ctx context.Context, product *domain.Product) (*domain.Product, error) {
	query := `
		UPDATE products
		SET title = $1, price = $2, description = $3, category = $4, images = $5, updated_by_id = $6, updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
	`

	cmdTag, err := r.pool.Exec(ctx, query,
		product.Title,
		product.Price,
		product.Description,
		product.Category,
		product.Images,
		product.UpdatedByID,
		product.ID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return nil, domain.NewNotFoundError("product not found")
	}

	return r.GetProductByID(ctx, product.ID)
}

// DeleteProduct removes a product by ID. Returns domain.ErrNotFound if ID does not exist.
func (r *PgProductRepository) DeleteProduct(ctx context.Context, id int64) error {
	query := `DELETE FROM products WHERE id = $1`

	cmdTag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return domain.NewNotFoundError("product not found")
	}

	return nil
}
