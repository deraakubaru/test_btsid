package main

import (
	"context"
	"fmt"
	"log"

	"btsid/db/migrations"
	"btsid/internal/config"
	"btsid/internal/database"
	"btsid/internal/handler"
	"btsid/internal/repository"
	"btsid/internal/router"
	"btsid/internal/service"
)

func main() {
	// 1. Load Configuration (fails fast if required env vars are missing)
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}
	log.Printf("Configuration loaded successfully. Server configured on port %s", cfg.Port)

	// 2. Establish PostgreSQL Connection Pool (bounded retry loop with exponential backoff)
	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL, database.DefaultRetryConfig())
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// 3. Execute Startup Database Migrations (fails fast on error)
	err = database.RunMigrations(migrations.MigrationsFS, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("FATAL: Database migration failed: %v", err)
	}
	log.Println("Database setup complete.")

	// 4. Initialize Dependency Graph (Repository -> Service -> Handler)
	userRepo := repository.NewUserRepository(pool)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTAccessTTLMinutes, cfg.JWTRefreshTTLDays)
	authHandler := handler.NewAuthHandler(authService)

	productRepo := repository.NewProductRepository(pool)
	productService := service.NewProductService(productRepo)
	productHandler := handler.NewProductHandler(productService)

	// 5. Wire Router & Start HTTP Server
	r := router.SetupRouter(authHandler, productHandler, cfg.JWTSecret)
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting HTTP server on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("FATAL: HTTP server failed to start: %v", err)
	}
}
