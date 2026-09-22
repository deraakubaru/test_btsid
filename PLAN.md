# Implementation Plan: Product Management & Authentication REST API

---

## 1. Executive Summary & Repository Status

### Current Repository State
* **Workspace Path**: `c:\Users\Dera Akbar\Documents\Test\BTSid`
* **Current Contents**: Contains `SPEC.md` (locked single source of truth). The rest of the project directory is empty.
* **Objective**: Build a clean, maintainable, and testable REST API in Go using Gin and PostgreSQL (`pgx/v5`) adhering strictly to `SPEC.md`.

---

## 2. Granular Task-by-Task Implementation Sequence

### Task 1: Repository Bootstrap & Environment Configuration Parser

* **Objective**: Initialize Go module, project directory layout, `.gitignore`, `.env.example`, and a fail-fast configuration loader.
* **Components / Files**:
  * `[NEW]` `go.mod` / `go.sum`
  * `[NEW]` `.gitignore`
  * `[NEW]` `.env.example`
  * `[NEW]` `cmd/api/main.go` (application skeleton)
  * `[NEW]` `internal/config/config.go` (config parser)
* **Prerequisites**: `SPEC.md` locked.
* **Implementation Notes**:
  * Read production settings directly from environment variables (`os.Getenv`). `[Requirement]` / `[User Decision]`
  * Load `.env` file optionally if present using `joho/godotenv` for local development. `[User Decision]` / `[Engineering Assumption]`
  * Fail fast and terminate application with `log.Fatal` if `DATABASE_URL` or `JWT_SECRET` is missing/empty. No hardcoded default for `JWT_SECRET`. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* App terminates with non-zero exit code if `DATABASE_URL` or `JWT_SECRET` is absent.
  * *Exact Command:* `go run ./cmd/api` (without env set).
  * *Expected Evidence:* Terminal logs `FATAL: missing required environment variable` and process exits with code 1.
  * *Proceed Safety:* Safe to proceed once fail-fast logic is verified.

---

### Task 2: Database Schema & Versioned SQL Migrations

* **Objective**: Write versioned SQL migration files for `users` and `products` tables.
* **Components / Files**:
  * `[NEW]` `db/migrations/000001_create_users_table.up.sql`
  * `[NEW]` `db/migrations/000001_create_users_table.down.sql`
  * `[NEW]` `db/migrations/000002_create_products_table.up.sql`
  * `[NEW]` `db/migrations/000002_create_products_table.down.sql`
* **Prerequisites**: Task 1.
* **Implementation Notes**:
  * `users` table: `id BIGSERIAL PK`, `username VARCHAR(100) NOT NULL UNIQUE`, `password_hash VARCHAR(255) NOT NULL`, `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`. `[Requirement]` / `[User Decision]`
  * `products` table: `id BIGSERIAL PK`, `title VARCHAR(255) NOT NULL`, `price NUMERIC(12,2) NOT NULL`, `description TEXT`, `category VARCHAR(100) NOT NULL`, `images TEXT[] NOT NULL`, `created_by_id BIGINT NOT NULL REFERENCES users(id)`, `updated_by_id BIGINT NOT NULL REFERENCES users(id)`, `created_at TIMESTAMPTZ`, `updated_at TIMESTAMPTZ`. `[Requirement]` / `[User Decision]`
  * Foreign keys intentionally omit `ON DELETE CASCADE` to prevent accidental loss of historical audit data. `[User Decision]`
  * Indexes on `LOWER(category)` and `(created_at DESC, id DESC)`. Omit redundant unique index on `username`. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* SQL files contain valid PostgreSQL DDL syntax.
  * *Exact Command:* `pg_dump` or test execution against local Postgres container.
  * *Expected Evidence:* Clean SQL parsing without syntax errors.
  * *Proceed Safety:* Safe to proceed once migration files are validated.

---

### Task 3: Database Connection Pool & Startup Migrations Runner

* **Objective**: Establish `pgxpool` connection pool, database backoff connection retry loop, and programmatic migration runner using `golang-migrate`.
* **Components / Files**:
  * `[NEW]` `internal/database/database.go` (pgxpool connection & backoff retry loop)
  * `[NEW]` `internal/database/migration.go` (golang-migrate startup execution)
* **Prerequisites**: Task 1, Task 2.
* **Implementation Notes**:
  * Retry loop connects to PostgreSQL with exponential backoff to handle container boot delays. `[Engineering Assumption]`
  * Runs pending migrations from `db/migrations/` using `golang-migrate/migrate/v4`. Fails fast on migration error. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* App connects to PostgreSQL and applies migration files automatically on launch.
  * *Exact Command:* `DATABASE_URL="..." JWT_SECRET="..." go run ./cmd/api`.
  * *Expected Evidence:* Log output: `Database connected. Migrations applied successfully.`
  * *Proceed Safety:* Safe to proceed once connection and migration runner are verified.

---

### Task 4: Core Domain Models & Custom UTC Timestamp Serializer

* **Objective**: Define domain structs, request/response DTOs, custom JSON serializer for timestamps in `YYYY-MM-DD HH:mm:ss` UTC format, and error envelope structures.
* **Components / Files**:
  * `[NEW]` `internal/domain/user.go` (User entity & DTOs)
  * `[NEW]` `internal/domain/product.go` (Product entity & DTOs)
  * `[NEW]` `internal/domain/timestamp.go` (Custom UTC JSON time serializer)
  * `[NEW]` `internal/domain/errors.go` (Standard Error Envelope & Domain errors)
  * `[NEW]` `internal/domain/timestamp_test.go`
* **Prerequisites**: Task 1, 2, 3.
* **Implementation Notes**:
  * Custom `CustomTime` type wraps `time.Time` and implements `json.Marshaler` returning string `"YYYY-MM-DD HH:mm:ss"` in **UTC**. `[User Decision]`
  * Response DTO formats `created_by_id` and `updated_by_id` as string (`"1"`), and `created_by` / `updated_by` as user `username`. `[Requirement]` / `[User Decision]`
  * Standard Error Envelope: `{ "success": false, "message": "...", "errors": [...] }`. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* `CustomTime` serializes to exact string format `"2025-01-01 15:01:04"`.
  * *Exact Command:* `go test -v ./internal/domain/...`.
  * *Expected Evidence:* `PASS: TestCustomTime_MarshalJSON`.
  * *Proceed Safety:* Safe to proceed once domain models and custom time serialization test passes.

---

### Task 5: Security — Password Hashing Component

* **Objective**: Implement password hashing and verification using `bcrypt`.
* **Components / Files**:
  * `[NEW]` `internal/security/password.go`
  * `[NEW]` `internal/security/password_test.go`
* **Prerequisites**: Task 4.
* **Implementation Notes**:
  * Use `golang.org/x/crypto/bcrypt` with standard cost 10. `[Requirement]` / `[User Decision]`
  * Functions: `HashPassword(password string) (string, error)` and `ComparePassword(hash, password string) error`.
* **Verification Checkpoint**:
  * *What must be true:* Correct password matches hash; incorrect password returns error.
  * *Exact Command:* `go test -v ./internal/security/...`.
  * *Expected Evidence:* `PASS: TestPasswordHashing`.
  * *Proceed Safety:* Safe to proceed once password security unit test passes.

---

### Task 6: Data Access — User Repository

* **Objective**: Implement explicit SQL data access layer for `users` persistence.
* **Components / Files**:
  * `[NEW]` `internal/repository/user_repository.go`
  * `[NEW]` `internal/repository/user_repository_test.go`
* **Prerequisites**: Task 3, Task 4, Task 5.
* **Implementation Notes**:
  * Functions: `CreateUser(ctx, username, passwordHash) (*domain.User, error)` and `GetUserByUsername(ctx, username) (*domain.User, error)`.
  * Uses `pgxpool` with parameterized SQL (`$1`, `$2`). `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* User is stored in DB; duplicate username returns unique constraint violation error.
  * *Exact Command:* `go test -v ./internal/repository/...` (against test PostgreSQL instance).
  * *Expected Evidence:* `PASS: TestUserRepository_CreateAndGet`.
  * *Proceed Safety:* Safe to proceed once user repository test passes.

---

### Task 7: Security — HS256 JWT Token Component

* **Objective**: Implement JWT generation and claim validation using `HS256` (HMAC-SHA256).
* **Components / Files**:
  * `[NEW]` `internal/security/jwt.go`
  * `[NEW]` `internal/security/jwt_test.go`
* **Prerequisites**: Task 4.
* **Implementation Notes**:
  * Use `golang-jwt/jwt/v5` with algorithm `HS256`. `[User Decision]`
  * Generates dual stateless tokens: `authentication_token` (15-min TTL, `token_type: access`) and `refresh_token` (7-day TTL, `token_type: refresh`). Claims include `sub` (User ID) and `username`. `[User Decision]`
  * Functions: `GenerateTokens(...)`, `ValidateToken(...)`.
* **Verification Checkpoint**:
  * *What must be true:* Tokens generated with `HS256`, correct claims retrieved, expired/tampered tokens fail validation.
  * *Exact Command:* `go test -v ./internal/security/...`.
  * *Expected Evidence:* `PASS: TestJWT_GenerateAndValidate`.
  * *Proceed Safety:* Safe to proceed once JWT security tests pass.

---

### Task 8: Business Logic — Authentication Service

* **Objective**: Implement user registration and login business workflows.
* **Components / Files**:
  * `[NEW]` `internal/service/auth_service.go`
  * `[NEW]` `internal/service/auth_service_test.go`
* **Prerequisites**: Task 5, Task 6, Task 7.
* **Implementation Notes**:
  * `Register`: Validates non-empty `username`, `password`, matching `password_confirmation`. Checks duplicate username via `user_repository`. Hashes password with `bcrypt`. Persists user. `[Requirement]` / `[User Decision]`
  * `Login`: Fetches user by username. Verifies password with `bcrypt`. Generates dual JWT tokens (`authentication_token`, `refresh_token`). `[Requirement]` / `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Registration succeeds for valid input, fails on duplicate username; login succeeds for valid credentials, fails on wrong password.
  * *Exact Command:* `go test -v ./internal/service/...`.
  * *Expected Evidence:* `PASS: TestAuthService_RegisterAndLogin`.
  * *Proceed Safety:* Safe to proceed once Auth Service tests pass.

---

### Task 9: Delivery — Authentication HTTP Handlers

* **Objective**: Implement Gin route handlers for `/api/auth/register` and `/api/auth/login`.
* **Components / Files**:
  * `[NEW]` `internal/handler/auth_handler.go`
* **Prerequisites**: Task 8.
* **Implementation Notes**:
  * Handler parses JSON payload via `c.ShouldBindJSON`. `[User Decision]`
  * `POST /api/auth/register` returns HTTP 201 Created on success or HTTP 400 Bad Request envelope on validation/duplicate error. `[Requirement]` / `[User Decision]`
  * `POST /api/auth/login` returns HTTP 200 OK with `authentication_token` and `refresh_token` on success, or HTTP 401 Unauthorized envelope on bad credentials. `[Requirement]` / `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Handlers respond with standard success and error envelope JSON payloads.
  * *Exact Command:* `go test -v ./internal/handler/...`.
  * *Expected Evidence:* `PASS: TestAuthHandler_Endpoints`.
  * *Proceed Safety:* Safe to proceed once Auth Handlers pass tests.

---

### Task 10: Security — Authentication Middleware

* **Objective**: Implement Gin Bearer token authentication middleware for protected routes.
* **Components / Files**:
  * `[NEW]` `internal/middleware/auth.go`
  * `[NEW]` `internal/middleware/auth_test.go`
* **Prerequisites**: Task 7.
* **Implementation Notes**:
  * Extracts token from `Authorization: Bearer <token>` header. `[User Decision]`
  * Validates token signature using `JWT_SECRET` and algorithm `HS256`. Checks `token_type == "access"`. `[User Decision]`
  * Injects `userID` (int64) and `username` (string) into Gin context (`c.Set("userID", ...)`). `[User Decision]`
  * Aborts request with HTTP 401 Unauthorized envelope if missing, invalid, or expired. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Valid access token allows request through and populates context; missing/invalid/refresh token returns HTTP 401.
  * *Exact Command:* `go test -v ./internal/middleware/...`.
  * *Expected Evidence:* `PASS: TestAuthMiddleware`.
  * *Proceed Safety:* Safe to proceed once Auth Middleware test passes.

---

### Task 11: Testing — Authentication Integration Tests

* **Objective**: Execute end-to-end `httptest` integration tests for authentication endpoints.
* **Components / Files**:
  * `[NEW]` `internal/handler/auth_integration_test.go`
* **Prerequisites**: Task 9, Task 10.
* **Implementation Notes**:
  * Tests registration flow, duplicate username rejection (400), login success (200), bad password rejection (401), and unauthorized access protection.
* **Verification Checkpoint**:
  * *What must be true:* All Auth API flows work end-to-end via `httptest`.
  * *Exact Command:* `go test -v -run TestAuthIntegration ./internal/handler/...`.
  * *Expected Evidence:* `PASS: TestAuthIntegration`.
  * *Proceed Safety:* Safe to proceed to Product implementation.

---

### Task 12: Data Access — Product Repository

* **Objective**: Implement explicit SQL data access layer for Product CRUD, search, category filter, and pagination.
* **Components / Files**:
  * `[NEW]` `internal/repository/product_repository.go`
  * `[NEW]` `internal/repository/product_repository_test.go`
* **Prerequisites**: Task 3, Task 4.
* **Implementation Notes**:
  * Parameterized SQL using `pgxpool` (`$1`, `$2`). No ORMs or dynamic string queries. `[User Decision]`
  * `GetProducts(ctx, search, category, offset, limit)`: Filters via `title ILIKE '%' || $1 || '%'` and `LOWER(category) = LOWER($2)`. Sorts deterministically by `created_at DESC, id DESC`. `[User Decision]`
  * `GetProductCount(ctx, search, category)`: Counts total matching items. `[User Decision]`
  * `GetProductByID(ctx, id)`: Fetches product by ID with JOIN on `users` table to fetch `created_by` and `updated_by` usernames. `[User Decision]`
  * `CreateProduct(...)`: Inserts new product record. `[User Decision]`
  * `UpdateProduct(...)`: Updates product fields and sets `updated_at = CURRENT_TIMESTAMP`, leaving `created_at` unchanged. `[User Decision]`
  * `DeleteProduct(ctx, id)`: Deletes product record. `[User Decision]`
  * Single atomic SQL statement for each repository function (no multi-statement transaction blocks). `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* SQL queries execute cleanly against real PostgreSQL instance; JOINs populate username audit fields.
  * *Exact Command:* `go test -v ./internal/repository/...`.
  * *Expected Evidence:* `PASS: TestProductRepository_CRUD`.
  * *Proceed Safety:* Safe to proceed once Product Repository tests pass.

---

### Task 13: Business Logic — Product Service & Validation

* **Objective**: Implement business validation and service methods for products.
* **Components / Files**:
  * `[NEW]` `internal/service/product_service.go`
  * `[NEW]` `internal/service/product_service_test.go`
* **Prerequisites**: Task 12.
* **Implementation Notes**:
  * Validates mandatory fields: `title` (non-empty), `price` (`>= 0`), `category` (non-empty), `images` (`len(images) >= 1` non-empty HTTP/HTTPS URL strings). `[Requirement]` / `[User Decision]` / `[Engineering Assumption]`
  * Validates pagination parameters: `page` (default 1), `limit` (default 10, max 100). If `limit > 100`, returns validation error (HTTP 400). `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Valid payloads pass; empty fields or `limit > 100` return validation errors.
  * *Exact Command:* `go test -v ./internal/service/...`.
  * *Expected Evidence:* `PASS: TestProductService_Validation`.
  * *Proceed Safety:* Safe to proceed once Product Service tests pass.

---

### Task 14: Delivery — Product GET Handlers (List & Detail)

* **Objective**: Implement Gin HTTP route handlers for `GET /api/products` and `GET /api/products/:id`.
* **Components / Files**:
  * `[NEW]` `internal/handler/product_get_handler.go`
* **Prerequisites**: Task 13.
* **Implementation Notes**:
  * `GET /api/products`: Parses query params `search`, `category`, `page`, `limit`. Returns HTTP 200 OK wrapped in pagination envelope `{ "success": true, "data": [...], "pagination": {...} }`. `[Requirement]` / `[User Decision]`
  * `GET /api/products/:id`: Parses `:id`. Returns HTTP 200 OK with product payload or HTTP 404 Not Found error envelope if missing. `[Requirement]` / `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* List endpoint returns paginated data; detail endpoint returns 404 for missing ID.
  * *Exact Command:* `go test -v ./internal/handler/...`.
  * *Expected Evidence:* `PASS: TestProductGetHandlers`.
  * *Proceed Safety:* Safe to proceed once GET handlers pass tests.

---

### Task 15: Delivery — Product POST Handler (Create)

* **Objective**: Implement protected Gin HTTP route handler for `POST /api/products`.
* **Components / Files**:
  * `[NEW]` `internal/handler/product_post_handler.go`
* **Prerequisites**: Task 10, Task 13.
* **Implementation Notes**:
  * Protected by Auth Middleware. Retrieves authenticated `userID` from Gin context. `[User Decision]`
  * Parses JSON body. Populates `created_by_id` and `updated_by_id` with `userID`. `[Requirement]` / `[User Decision]`
  * Returns HTTP 201 Created with created product payload on success, or HTTP 400 Bad Request error envelope on validation failure. `[Requirement]` / `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Authorized request creates product with audit fields; unauthorized request returns HTTP 401.
  * *Exact Command:* `go test -v ./internal/handler/...`.
  * *Expected Evidence:* `PASS: TestProductPostHandler`.
  * *Proceed Safety:* Safe to proceed once POST handler passes tests.

---

### Task 16: Delivery — Product PUT Handler (Update)

* **Objective**: Implement protected Gin HTTP route handler for `PUT /api/products/:id`.
* **Components / Files**:
  * `[NEW]` `internal/handler/product_put_handler.go`
* **Prerequisites**: Task 10, Task 13.
* **Implementation Notes**:
  * Protected by Auth Middleware. Global Write Access for any authenticated user. `[User Decision]`
  * Enforces strict full resource replacement semantics. Requires all mandatory product fields. `[User Decision]`
  * Sets `updated_at = CURRENT_TIMESTAMP` explicitly, updates `updated_by_id`, leaves `created_at` unchanged. `[User Decision]`
  * Returns HTTP 200 OK with updated product payload, or HTTP 404 Not Found if product ID does not exist. `[Requirement]` / `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* PUT updates fields and `updated_at`; returns 404 for non-existent product ID.
  * *Exact Command:* `go test -v ./internal/handler/...`.
  * *Expected Evidence:* `PASS: TestProductPutHandler`.
  * *Proceed Safety:* Safe to proceed once PUT handler passes tests.

---

### Task 17: Delivery — Product DELETE Handler (Delete)

* **Objective**: Implement protected Gin HTTP route handler for `DELETE /api/products/:id`.
* **Components / Files**:
  * `[NEW]` `internal/handler/product_delete_handler.go`
* **Prerequisites**: Task 10, Task 13.
* **Implementation Notes**:
  * Protected by Auth Middleware. Global Write Access for any authenticated user. `[User Decision]`
  * Removes product record by ID. Returns HTTP 200 OK with success message envelope or HTTP 404 Not Found error envelope if product does not exist. `[Requirement]` / `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* DELETE removes product from database; returns 404 if ID does not exist.
  * *Exact Command:* `go test -v ./internal/handler/...`.
  * *Expected Evidence:* `PASS: TestProductDeleteHandler`.
  * *Proceed Safety:* Safe to proceed once DELETE handler passes tests.

---

### Task 18: Security & Network — CORS Middleware

* **Objective**: Implement Gin CORS middleware supporting cross-domain request access.
* **Components / Files**:
  * `[NEW]` `internal/middleware/cors.go`
  * `[NEW]` `internal/middleware/cors_test.go`
* **Prerequisites**: Task 1.
* **Implementation Notes**:
  * Configured via `CORS_ALLOWED_ORIGINS` environment variable (defaults to `*`). `[User Decision]`
  * Supports methods: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`. Headers: `Authorization`, `Content-Type`. Responds to `OPTIONS` preflight requests. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Preflight `OPTIONS` request returns HTTP 204 with CORS headers.
  * *Exact Command:* `go test -v ./internal/middleware/...`.
  * *Expected Evidence:* `PASS: TestCORSMiddleware`.
  * *Proceed Safety:* Safe to proceed once CORS middleware test passes.

---

### Task 19: Testing — Product API Integration & Verification Scope Clarification

* **Objective**: Execute comprehensive API and database integration tests for all product endpoints and data access methods.
* **Components / Files**:
  * `[NEW]` `internal/handler/product_integration_test.go`
  * `[MODIFY]` `internal/repository/product_repository_test.go`
* **Prerequisites**: Task 14, 15, 16, 17, 18.
* **Implementation Notes**:
  * **HTTP/API Integration Testing Scope (`httptest`):** Verifies the HTTP/API end-to-end routing, payload binding, status code mappings (200, 201, 400, 401, 404), headers, and JSON error envelope formatting.
  * **Database Repository Testing Scope (Real PostgreSQL):** Executed against a real PostgreSQL instance to verify actual database persistence, SQL syntax validity, `ILIKE` search logic, `LOWER` category filtering, pagination `COUNT` and offset queries, and audit field JOIN relationships.
* **Verification Checkpoint**:
  * *What must be true:* API routes format HTTP responses correctly; real PostgreSQL queries execute valid SQL and return exact filtered datasets.
  * *Exact Command:* `go test -v ./internal/handler/... ./internal/repository/...`.
  * *Expected Evidence:* `PASS: TestProductIntegration` and `PASS: TestProductRepository_Postgres`.
  * *Proceed Safety:* Safe to proceed to delivery documentation artifacts.

---

### Task 20: Delivery Artifacts — OpenAPI/Swagger Documentation & Postman Collection

* **Objective**: Annotate handlers, generate Swagger/OpenAPI docs, expose `/swagger/index.html`, and create `postman_collection.json`.
* **Components / Files**:
  * `[NEW]` `docs/docs.go` / `docs/swagger.json` / `docs/swagger.yaml` (Generated via `swag init`)
  * `[NEW]` `postman_collection.json` (Exported Postman collection)
  * `[MODIFY]` `cmd/api/main.go` (Register Swagger UI route)
* **Prerequisites**: Task 19.
* **Implementation Notes**:
  * Annotate Gin handlers with standard OpenAPI 2.0 annotations. Serve UI via `swaggo/gin-swagger`. `[User Decision]`
  * Document all 7 endpoints, query params, headers, request/response schemas, error envelopes. `[Requirement]` / `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Swagger UI renders interactively at `/swagger/index.html`.
  * *Exact Command:* `swag init -g cmd/api/main.go` followed by starting server and curling `/swagger/index.html`.
  * *Expected Evidence:* HTTP 200 response with HTML payload containing Swagger UI assets.
  * *Proceed Safety:* Safe to proceed once Swagger docs generation succeeds.

---

### Task 21: Delivery Artifacts — Dockerfile & Docker Compose Setup

* **Objective**: Create multi-stage `Dockerfile` and `docker-compose.yml` configuration with environment variable substitution.
* **Components / Files**:
  * `[NEW]` `Dockerfile`
  * `[NEW]` `docker-compose.yml`
* **Prerequisites**: Task 20.
* **Implementation Notes**:
  * Multi-stage `Dockerfile`: Stage 1 compiles static Go binary; Stage 2 copies binary and `db/migrations/` directory into lightweight runner. `[User Decision]`
  * `docker-compose.yml` uses environment variable substitution (`JWT_SECRET: ${JWT_SECRET}`) and `DATABASE_URL`. App continues to fail fast if `JWT_SECRET` is absent. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* `docker compose up --build` compiles Go binary, boots Postgres, runs migrations automatically, and starts API.
  * *Exact Command:* `JWT_SECRET="devsecret" docker compose up --build`.
  * *Expected Evidence:* `app` logs: `Database connected. Migrations applied. Server listening on :8080`.
  * *Proceed Safety:* Safe to proceed once Docker deployment succeeds.

---

### Task 22: Quality Assurance & End-to-End Verification Suite

* **Objective**: Run full QA suite across repository: `gofmt`, `go vet`, unit tests, integration tests, DB tests, and API smoke testing.
* **Components / Files**:
  * Repository-wide audit.
* **Prerequisites**: Task 21.
* **Implementation Notes**:
  * Execute static formatting and linting checks. All implemented unit, integration, and repository tests must pass cleanly without failures. `[User Decision]`
  * *Clarification on 100% Test Pass Rate:* Every test implemented in the project must pass (0 test failures). This does NOT introduce or imply a 100% code coverage requirement. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Zero formatting errors, zero vet issues, zero test failures across all implemented tests.
  * *Exact Commands:*
    1. `gofmt -l .` $\rightarrow$ (returns empty output)
    2. `go vet ./...` $\rightarrow$ (exits code 0)
    3. `go test -v ./...` $\rightarrow$ (all implemented tests PASS)
  * *Expected Evidence:* Terminal confirms clean pass across all 3 commands.
  * *Proceed Safety:* Mandatory Phase 1 is officially complete and ready for signoff.

---

### Task 23: Phase 2 Optional Features — In-Memory Rate Limiter Middleware

> [!IMPORTANT]
> Task 23 and 24 MUST NOT be started until Task 1 through 22 mandatory Phase 1 functionality is 100% complete and verified.

* **Objective**: Implement thread-safe in-memory rate limiting middleware with distinct keying strategies.
* **Components / Files**:
  * `[NEW]` `internal/middleware/ratelimit.go`
  * `[NEW]` `internal/middleware/ratelimit_test.go`
* **Prerequisites**: Task 22 (Phase 1 Complete).
* **Implementation Notes**:
  * Product write endpoints (`POST`, `PUT`, `DELETE /api/products`): Max 1 request per 5 seconds. **Keyed strictly by authenticated user ID** (`c.GetInt64("userID")`). `[Requirement]` / `[User Decision]`
  * Public Auth endpoints (`POST /api/auth/register`, `POST /api/auth/login`): Max 3 requests per 60 seconds. **Keyed strictly by client IP** (`c.ClientIP()`). `[Requirement]` / `[User Decision]`
  * Returns HTTP 429 Too Many Requests envelope when limit is breached. Zero Redis or external infrastructure. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* 4th login attempt within 60s from same IP returns HTTP 429. 2nd product POST within 5s from same user ID returns HTTP 429.
  * *Exact Command:* `go test -v ./internal/middleware/ratelimit_test.go`.
  * *Expected Evidence:* `PASS: TestRateLimiter_UserAndIPKeying`.
  * *Proceed Safety:* Safe to proceed to Task 24.

---

### Task 24: Phase 2 Optional Features — In-Memory TTL Cache & Cache Key Requirements

* **Objective**: Implement thread-safe in-memory TTL caching for product GET endpoints with automatic mutation invalidation and explicit cache key formatting.
* **Components / Files**:
  * `[NEW]` `internal/cache/cache.go` (Thread-safe in-memory cache with mutex)
  * `[NEW]` `internal/service/cached_product_service.go` (Cache wrapper & invalidation logic)
  * `[NEW]` `internal/cache/cache_test.go`
* **Prerequisites**: Task 23.
* **Implementation Notes**:
  * **Cache Key Specification:**
    * Product List Cache Keys (`GET /api/products`): MUST uniquely incorporate all effective query parameters: `products:list:search={search}:category={category}:page={page}:limit={limit}`. `[User Decision]`
    * Product Detail Cache Keys (`GET /api/products/:id`): MUST uniquely incorporate the product ID: `products:detail:{id}`. `[User Decision]`
  * Caches responses for `GET /api/products` and `GET /api/products/:id`. `[Requirement]` / `[User Decision]`
  * Automatically invalidates product cache entries upon product creation, update, or deletion (`POST`, `PUT`, `DELETE`). `[User Decision]`
  * Protected by `sync.RWMutex`. Zero external infrastructure. `[User Decision]`
* **Verification Checkpoint**:
  * *What must be true:* Subsequent GET requests with identical parameters hit cache; different filter parameters generate separate cache keys; POST/PUT/DELETE explicitly purges cached items.
  * *Exact Command:* `go test -v ./internal/cache/...`.
  * *Expected Evidence:* `PASS: TestCache_KeysAndInvalidationOnMutation`.
  * *Proceed Safety:* Phase 2 optional features complete.

---

## 3. Database Integration Test Isolation Strategy

To ensure database integration tests remain deterministic and isolated without introducing application-level transaction overhead:

1. **Test Environment Isolation:** Database integration tests will run against a dedicated test database container (e.g. `btsid_test_db`). `[User Decision]`
2. **Deterministic Cleanup Strategy:** Every database test suite executes an explicit SQL table truncation in its setup/teardown hook:
   ```sql
   TRUNCATE TABLE users, products RESTART IDENTITY CASCADE;
   ``` `[User Decision]`
3. **No Application Transaction Hacks:** Application code relies strictly on single atomic SQL statements per repository call. No transaction rollback wrappers will be introduced into repository interfaces for testing purposes. `[User Decision]`

---

## 4. Requirement-to-Verification Traceability Matrix

| Original Assessment Requirement | Task | Target Files / Components | Verification Evidence |
| :--- | :--- | :--- | :--- |
| `POST /api/auth/register` (Register user payload) | Task 8, 9, 11 | `auth_service.go`, `auth_handler.go` | `httptest` returns HTTP 201 Created & duplicate username returns 400 |
| `POST /api/auth/login` (Return access & refresh tokens) | Task 8, 9, 11 | `auth_service.go`, `auth_handler.go`, `jwt.go` | `httptest` returns 200 OK with `authentication_token` & `refresh_token` |
| Authorization Header Protection (`POST`, `PUT`, `DELETE`) | Task 10, 15, 16, 17 | `auth.go` middleware | `httptest` without Bearer token returns HTTP 401 Unauthorized envelope |
| `GET /api/products` (Search, Category, Pagination) | Task 12, 13, 14, 19 | `product_repository.go`, `product_get_handler.go` | `httptest` verifies HTTP routes; Postgres test verifies `ILIKE`, `LOWER`, and count |
| `GET /api/products/:id` (Detail & 404 error) | Task 12, 14, 19 | `product_get_handler.go` | `httptest` returns 200 for valid ID and 404 for missing ID |
| `POST /api/products` (Create product & Audit fields) | Task 12, 13, 15, 19 | `product_post_handler.go` | `httptest` returns 201 Created & populates `created_by` / `created_by_id` |
| `PUT /api/products/:id` (Full update & 404 error) | Task 12, 13, 16, 19 | `product_put_handler.go` | `httptest` replaces full resource, updates `updated_at`, returns 404 for missing ID |
| `DELETE /api/products/:id` (Delete product & 404 error) | Task 12, 17, 19 | `product_delete_handler.go` | `httptest` removes product, returns 200, returns 404 on subsequent GET |
| Docker Containerization (`docker compose up`) | Task 21 | `Dockerfile`, `docker-compose.yml` | `docker compose up --build` boots app & Postgres with auto-migrations |
| CORS Support (Cross-domain access) | Task 18 | `cors.go` | `httptest` preflight `OPTIONS` request returns HTTP 204 with CORS headers |
| API Documentation (Swagger / Postman) | Task 20 | `docs/`, `postman_collection.json` | Swagger UI responds at `/swagger/index.html` and Postman collection exported |
| Optional Basic Cache & Rate Limiting | Task 23, 24 | `ratelimit.go`, `cache.go` | Unit tests verify HTTP 429 response, explicit cache keys, and invalidation |

---

## 5. Definition of Done for Mandatory Phase 1

Mandatory Phase 1 implementation is complete when:
1. Tasks 1 through 22 are 100% completed with passing verification checkpoints.
2. All 7 HTTP API endpoints strictly conform to the contracts in `SPEC.md`.
3. Database startup connection retry loop handles PostgreSQL container initialization, and `golang-migrate` applies schema migrations automatically on app launch.
4. Timestamps serialize as `YYYY-MM-DD HH:mm:ss` UTC in JSON responses.
5. JWT tokens are signed using `HS256` and enforced via authentication middleware.
6. All non-2xx responses adhere to the standard Error Envelope (`{ "success": false, "message": "...", "errors": [...] }`).
7. `docker compose up --build` compiles and runs the application and database in a containerized environment using environment substitution (`${JWT_SECRET}`).
8. Live interactive Swagger UI is served at `http://localhost:8080/swagger/index.html` and `postman_collection.json` is present in the repository root.
9. `gofmt -l .` returns zero unformatted files, `go vet ./...` exits code 0, and `go test -v ./...` passes all implemented unit, integration, and database tests (zero test failures).

---

## 6. Remaining Ambiguities Checklist

* **None.** Every architectural choice, file component, task sequence, security rule, data contract, verification checkpoint, cache key format, and delivery artifact is completely specified and locked in `SPEC.md`.
