# Software Specification: Product Management & Authentication REST API

---

## 1. Assessment Objective and Scope

This document serves as the single source of truth for the implementation of the Backend Developer Coding Test. 

The objective is to implement a **clean, maintainable, and testable REST API suitable for the assessment** in **Go** using the **Gin** web framework for **Product Management** and **Authentication**. The delivery includes containerization via **Docker**, **CORS** support, **Swagger/OpenAPI** documentation, and automated tests.

---

## 2. Mandatory Requirements

* **`GET /api/products`**: Retrieve a paginated list of products with search and category filtering support. `[Requirement]`
* **`GET /api/products/:id`**: Retrieve detailed information for a single product by its ID. Return HTTP 404 JSON error if not found. `[Requirement]`
* **`POST /api/products`**: Create a new product record. Validates required fields (`title`, `price`, `category`, `images` $\ge 1$). Requires valid authorization. Returns HTTP 201 Created on success or HTTP 400 Bad Request on validation failure. `[Requirement]`
* **`PUT /api/products/:id`**: Update an existing product record by ID. Return HTTP 404 if not found. Requires valid authorization. `[Requirement]`
* **`DELETE /api/products/:id`**: Remove a product record by ID. Return HTTP 404 if not found. Requires valid authorization. `[Requirement]`
* **`POST /api/auth/register`**: Register a new user account with `username`, `password`, and `password_confirmation`. `[Requirement]`
* **`POST /api/auth/login`**: Authenticate a user with `username` and `password`. Returns `authentication_token` and `refresh_token`. `[Requirement]`
* **Delivery Artifacts**: Docker containerization (`docker compose up`), CORS support, API documentation (Swagger/Postman), public Git repository setup. `[Requirement]`

---

## 3. Optional Requirements

* **Basic Cache**: Basic response caching for `GET /api/products` and `GET /api/products/:id`. `[Requirement]`
* **Rate Limiting**:
  * Product write endpoints (`POST`, `PUT`, `DELETE /api/products`): Maximum 1 request per 5 seconds. `[Requirement]`
  * Auth endpoints (`POST /api/auth/register`, `POST /api/auth/login`): Maximum 3 requests per 60 seconds. `[Requirement]`
* **Optional Features Implementation Scope**: Optional features are strictly assigned to **Phase 2** execution after all mandatory CRUD and Auth features are fully built and verified. They will be implemented using thread-safe in-memory rate limiters and in-memory TTL caches without external infrastructure dependencies like Redis. `[User Decision]`

---

## 4. Approved Technology Stack

* **Programming Language**: Go (v1.22+) `[User Decision]`
* **Web Framework**: Gin Framework (`github.com/gin-gonic/gin`) `[User Decision]`
* **Database Engine**: PostgreSQL 16 `[User Decision]`
* **Database Driver**: `jackc/pgx/v5` (`pgxpool`) using explicit parameterized SQL queries. No ORMs. `[User Decision]`
* **Database Migrations**: `golang-migrate/migrate/v4` executing on application startup. `[User Decision]`
* **Authentication**: Password hashing via `golang.org/x/crypto/bcrypt` and JWT signing using **HS256** (HMAC-SHA256) via `golang-jwt/jwt/v5`. `[User Decision]`
* **API Documentation**: OpenAPI / Swagger UI served live via `swaggo/gin-swagger` at `/swagger/index.html`, alongside an exported `postman_collection.json`. `[User Decision]`

---

## 5. Database Schema and Relationships

### A. Table `users`
```sql
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
``` `[User Decision]`

### B. Table `products`
```sql
CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    price NUMERIC(12, 2) NOT NULL,
    description TEXT NULL,
    category VARCHAR(100) NOT NULL,
    images TEXT[] NOT NULL,
    created_by_id BIGINT NOT NULL REFERENCES users(id),
    updated_by_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
``` `[User Decision]`

* **Foreign Key Integrity Decision**: Foreign keys `created_by_id` and `updated_by_id` use standard SQL foreign key constraints without `ON DELETE CASCADE`. This preserves historical product data integrity and prevents unintended cascading deletions of products. `[User Decision]`

### C. Indexes
```sql
CREATE INDEX IF NOT EXISTS idx_products_category ON products(LOWER(category));
CREATE INDEX IF NOT EXISTS idx_products_created_at_id ON products(created_at DESC, id DESC);
``` `[User Decision]`

### D. Transaction Boundaries
* Each individual repository query is a single atomic SQL statement. Some endpoints such as paginated listing execute multiple independent queries (for example COUNT and SELECT), but no explicit multi-statement transaction is required for the current assessment operations. `[User Decision]`

---

## 6. API Endpoints and HTTP Contracts

| HTTP Method | Route Endpoint | Access | Success Status | Error Statuses | Description |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `POST` | `/api/auth/register` | Public | 201 Created | 400 Bad Request, 429 | Register new user account |
| `POST` | `/api/auth/login` | Public | 200 OK | 400, 401 Unauthorized, 429 | Authenticate and obtain tokens |
| `GET` | `/api/products` | Public | 200 OK | 400 Bad Request | List paginated products with filters |
| `GET` | `/api/products/:id` | Public | 200 OK | 404 Not Found | Get product detail by ID |
| `POST` | `/api/products` | Protected | 201 Created | 400, 401 Unauthorized, 429 | Create new product |
| `PUT` | `/api/products/:id` | Protected | 200 OK | 400, 401, 404 Not Found, 429 | Replace product details by ID |
| `DELETE` | `/api/products/:id` | Protected | 200 OK | 401 Unauthorized, 404, 429 | Delete product by ID |

---

## 7. Request / Response JSON Examples

### A. POST `/api/auth/register`
**Request Payload:**
```json
{
  "username": "jhon_doe",
  "password": "supersecret",
  "password_confirmation": "supersecret"
}
``` `[Requirement]`

**Response Payload (201 Created):**
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "username": "jhon_doe"
  }
}
``` `[User Decision]`

### B. POST `/api/auth/login`
**Request Payload:**
```json
{
  "username": "jhon_doe",
  "password": "supersecret"
}
``` `[Requirement]`

**Response Payload (200 OK):**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "authentication_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
``` `[Requirement]` / `[User Decision]`

### C. GET `/api/products?search=shirt&category=Clothes&page=1&limit=10`
**Response Payload (200 OK):**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "title": "Awesome T-Shirt",
      "price": 99.99,
      "description": "High-quality cotton t-shirt",
      "category": "Clothes",
      "images": [
        "https://placeimg.com/640/480/any"
      ],
      "created_at": "2025-01-01 15:01:04",
      "created_by": "jhon_doe",
      "created_by_id": "1",
      "updated_at": "2025-01-01 15:01:04",
      "updated_by": "jhon_doe",
      "updated_by_id": "1"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 10,
    "total_items": 1,
    "total_pages": 1
  }
}
``` `[Requirement]` / `[User Decision]`

### D. POST `/api/products`
**Request Header:** `Authorization: Bearer <authentication_token>` `[User Decision]`  
**Request Payload:**
```json
{
  "title": "Awesome T-Shirt",
  "price": 99.99,
  "description": "High-quality cotton t-shirt",
  "category": "Clothes",
  "images": [
    "https://placeimg.com/640/480/any"
  ]
}
``` `[Requirement]`

**Response Payload (201 Created):**
```json
{
  "success": true,
  "message": "Product created successfully",
  "data": {
    "id": 1,
    "title": "Awesome T-Shirt",
    "price": 99.99,
    "description": "High-quality cotton t-shirt",
    "category": "Clothes",
    "images": [
      "https://placeimg.com/640/480/any"
    ],
    "created_at": "2025-01-01 15:01:04",
    "created_by": "jhon_doe",
    "created_by_id": "1",
    "updated_at": "2025-01-01 15:01:04",
    "updated_by": "jhon_doe",
    "updated_by_id": "1"
  }
}
``` `[Requirement]` / `[User Decision]`

---

## 8. Validation Rules

* **Registration Validation**:
  * `username`: Required, non-empty string. `[Requirement]`
  * `password`: Required, non-empty string. `[Requirement]`
  * `password_confirmation`: Required, must equal `password`. `[Requirement]`
* **Product Creation (`POST`) & Modification (`PUT`) Validation**:
  * `title`: Required, non-empty string. `[Requirement]`
  * `price`: Required, valid numeric value. `[Requirement]` (Non-negative numeric validation `price >= 0` applied at payload binding: `[Engineering Assumption]`)
  * `category`: Required, non-empty string. `[Requirement]`
  * `images`: Required, array of strings with `len(images) >= 1` containing non-empty HTTP/HTTPS URL strings. Syntactic URL format check only; no remote HTTP reachability check or duplicate-image detection. `[Requirement]` / `[User Decision]`
  * `description`: Optional string. `[Requirement]`

---

## 9. Authentication and Authorization

* **Authentication**: Clients authenticate using standard HTTP `Authorization` header formatted as `Authorization: Bearer <authentication_token>`. `[User Decision]`
* **Authorization Policy**: **Global Write Access**. Any authenticated user carrying a valid access token is authorized to invoke `POST`, `PUT`, and `DELETE` on products. `[User Decision]`

---

## 10. JWT Access/Refresh Token Behavior

* **Algorithm**: **HS256** (HMAC-SHA256). `[User Decision]`
* **Access Token (`authentication_token`)**: Signed JWT, 15-minute lifespan. Claims contain `sub` (User ID), `username`, `token_type: access`. `[User Decision]`
* **Refresh Token (`refresh_token`)**: Signed JWT, 7-day lifespan. Claims contain `sub` (User ID), `username`, `token_type: refresh`. `[User Decision]`
* **Stateless Token System**: No refresh token database table, no token revocation system, no token rotation mechanism, and no `/api/auth/refresh` endpoint. `[User Decision]`

---

## 11. Error Response Contract and HTTP Status Mapping

All non-2xx responses adhere strictly to the standardized Error Envelope format:
```json
{
  "success": false,
  "message": "Validation failed",
  "errors": [
    "title is required",
    "images must contain at least 1 item"
  ]
}
``` `[User Decision]`

### Status Code Mapping:
* **`400 Bad Request`**: Validation errors, malformed JSON body, duplicate username registration, or `limit > 100`. `[User Decision]`
* **`401 Unauthorized`**: Missing, invalid signature, or expired JWT access token. `[User Decision]`
* **`404 Not Found`**: Target product ID does not exist in the database. `[User Decision]`
* **`429 Too Many Requests`**: Rate limit quota exceeded (Phase 2). `[User Decision]`
* **`500 Internal Server Error`**: Unexpected database error. Sanitized to hide sensitive internal stack traces. `[User Decision]`

---

## 12. Search, Filter, and Pagination Behavior

* **Search (`?search=keyword`)**: Case-insensitive partial string match on product title (`title ILIKE '%' || $1 || '%'`). `[User Decision]`
* **Category (`?category=Clothes`)**: Case-insensitive exact string match on product category (`LOWER(category) = LOWER($1)`). `[User Decision]`
* **Pagination Guardrails**:
  * Default `page`: `1` (if omitted or `<= 0`). `[User Decision]`
  * Default `limit`: `10` (if omitted or `<= 0`). `[User Decision]`
  * Maximum `limit`: `100`. If `limit > 100`, the API returns HTTP 400 Bad Request error envelope. `[User Decision]`
* **Ordering**: Deterministic sort order `created_at DESC, id DESC`. `[User Decision]`

---

## 13. Audit Fields Behavior

* **`created_at` / `updated_at` Timestamps**:
  * `created_at` and `updated_at` in JSON responses MUST be formatted as `YYYY-MM-DD HH:mm:ss` serialized in **UTC**. The database continues to store timestamps as PostgreSQL `TIMESTAMPTZ`. `[User Decision]`
  * On product creation (`POST`), both timestamps use the database default (`CURRENT_TIMESTAMP`). `[User Decision]`
  * On every successful product update (`PUT`), `updated_at` is explicitly set to `CURRENT_TIMESTAMP`, while `created_at` remains unchanged. `[User Decision]`
* **`created_by` / `updated_by` Display Names**: Dynamically mapped to the user's `username` via SQL JOIN with the `users` table. `[User Decision]`
* **`created_by_id` / `updated_by_id` Audit Identifiers**: Formatted as string representations of the user's primary key ID (e.g., `"1"`). `[User Decision]`

---

## 14. CORS Requirements

* Configurable via `CORS_ALLOWED_ORIGINS` environment variable (defaults to `*` for local evaluation). `[User Decision]`
* Allowed HTTP methods: `GET`, `POST`, `PUT`, `DELETE`, `OPTIONS`. `[User Decision]`
* Allowed headers: `Authorization`, `Content-Type`. `[User Decision]`

---

## 15. Configuration / Environment Variables

| Variable Name | Default Value | Description |
| :--- | :--- | :--- |
| `PORT` | `8080` | Application HTTP port `[Engineering Assumption]` |
| `DATABASE_URL` | None (Required) | PostgreSQL connection string `[Engineering Assumption]` |
| `JWT_SECRET` | None (Required) | Secret key for signing JWTs. Must be explicitly provided via environment. Application fails to start if omitted. `[User Decision]` / `[Engineering Assumption]` |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Access token lifespan in minutes `[Engineering Assumption]` |
| `JWT_REFRESH_TTL_DAYS` | `7` | Refresh token lifespan in days `[Engineering Assumption]` |
| `CORS_ALLOWED_ORIGINS` | `*` | Allowed origins for CORS `[Engineering Assumption]` |

---

## 16. Docker and Startup Migration Behavior

* Single command execution: `docker compose up --build`. `[Requirement]`
* Startup migration flow:
  1. Parse environment configuration. Fail fast if required variables (`DATABASE_URL`, `JWT_SECRET`) are missing.
  2. Connect to PostgreSQL using connection retry loop with exponential backoff.
  3. Execute pending SQL migrations from `db/migrations/` via `golang-migrate`. Fail fast with exit code 1 if migration fails.
  4. Initialize dependency graph and start Gin HTTP server. `[User Decision]`

---

## 17. Testing and Verification Requirements

* **Unit Tests (`go test ./internal/...`)**: Verify password hashing/comparison, JWT claim generation and validation, token expiration, request DTO validations, and pagination calculations. `[User Decision]`
* **API Integration Tests (`httptest`)**: Cover all 7 HTTP endpoints for both success cases and error paths (400, 401, 404). `[User Decision]`
* **Database Integration Tests**: Run tests against a real PostgreSQL instance to verify SQL syntax, `ILIKE` search, `LOWER` category filtering, pagination counts, and UNIQUE constraints. `[User Decision]`
* **Verification Suite**: `go test ./...`, `go vet ./...`, `gofmt`, Docker startup validation, and Swagger UI interactive check. `[User Decision]`

---

## 18. Security Requirements

* Passwords hashed using `bcrypt` (cost 10). Plaintext passwords never stored. `[Requirement]` / `[User Decision]`
* JWTs signed using **HS256** algorithm. `JWT_SECRET` must be loaded from environment configuration; application fails fast if missing. No hardcoded fallback secret in application code. `[User Decision]`
* All database queries parameterized using `pgx` placeholders (`$1`, `$2`). Zero dynamic SQL string formatting. `[User Decision]`
* Sensitive database traces masked in API responses. `[User Decision]`

---

## 19. Explicit Engineering Assumptions

* Environment variables (`PORT`, `DATABASE_URL`, `JWT_SECRET`) will be passed via container configuration or `.env` file during development. `[Engineering Assumption]`
* DB connection retries handle container initialization race conditions during `docker compose up`. `[Engineering Assumption]`
* `price` validation checks numeric validity (`price >= 0`). `[Engineering Assumption]`

---

## 20. Out-of-Scope Items (Intentionally Omitted)

* No `POST /api/auth/refresh` endpoint. `[User Decision]`
* No refresh token database table, token revocation, or rotation mechanism. `[User Decision]`
* No RBAC or resource ownership enforcement. `[User Decision]`
* No Redis or external caching/rate limiting services. `[User Decision]`
* No explicit multi-statement SQL transactions. `[User Decision]`
* No remote checking or fetching of image URLs. `[User Decision]`
* No fallback default `JWT_SECRET` string in code. `[User Decision]`
* No `ON DELETE CASCADE` foreign key clauses on audit fields. `[User Decision]`

---

## 21. Remaining Ambiguity

* **None.** All functional, structural, data contract, security, and delivery requirements are fully specified, disambiguated, and ready for execution.
