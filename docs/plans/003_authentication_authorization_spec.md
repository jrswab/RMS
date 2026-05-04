# Spec 003: Authentication and Authorization

## Section 1: Context & Constraints

### Milestone Entry

> 003: Authentication and authorization — `/login` endpoint with bcrypt verification, JWT generation with user/restaurant claims, auth middleware, RBAC middleware for role-based endpoint protection

### Research Findings — Relevant Context

**Codebase structure (post-milestone 002):**

```
lab37/
├── AGENTS.md
├── README.md
├── Makefile
├── go.mod                 # module lab37, Go 1.22
├── go.sum
├── cmd/
│   └── seed/
│       ├── main.go        # Seed data (raw SQL)
│       └── seed_test.go
├── docs/plans/
│   ├── RMS000_design.md
│   ├── RMS_recipe_manager_milestones.md
│   ├── 001_database_schema_spec.md
│   ├── 001_database_schema_implement.md
│   ├── 002_data_access_layer_spec.md
│   └── 002_data_access_layer_implement.md
├── internal/
│   ├── db/
│   │   ├── db.go          # Open(dbPath) → *sql.DB, DefaultPath()
│   │   ├── db_test.go
│   │   ├── migrate.go     # RunMigrations(db, fs.FS)
│   │   └── migrate_test.go
│   └── store/
│       ├── store.go       # DBTX interface, ErrNotFound sentinel
│       ├── models.go      # User, Restaurant, Food, UserRestaurant, Recipe, Ingredient structs
│       ├── user.go        # GetUserByUsername, GetUserByID
│       ├── restaurant.go  # GetRestaurantByID, ListRestaurantsByUserID
│       ├── food.go        # GetFoodByID, ListFood
│       ├── recipe.go      # GetRecipeByID, SearchRecipes, CreateRecipe, UpdateRecipe, DeleteRecipe
│       ├── ingredient.go  # ListIngredientsByRecipeID, ReplaceIngredients
│       ├── user_restaurant.go  # ListUserRestaurantIDs
│       └── *_test.go      # Full TDD test coverage for all store methods
├── migrations/
│   ├── embed.go           # //go:embed *.up.sql → var FS embed.FS
│   └── 001_create_tables.up.sql
└── static/
```

**Dependencies (go.mod):**
- `github.com/golang-jwt/jwt/v5 v5.2.1` — JWT library (already present)
- `github.com/google/uuid v1.6.0` — UUID generation
- `github.com/mattn/go-sqlite3 v1.14.22` — SQLite driver
- `golang.org/x/crypto v0.33.0` — bcrypt password hashing (already present)

**Store layer methods relevant to this milestone:**
- `store.GetUserByUsername(db DBTX, username string) (*User, error)` — returns user with hashed password, role. Returns `ErrNotFound` if no match.
- `store.GetUserByID(db DBTX, id string) (*User, error)` — returns user by ID.
- `store.ListUserRestaurantIDs(db DBTX, userID string) ([]string, error)` — returns restaurant IDs for a user.
- `store.User` struct: `ID, Username, Password, Role string; CreatedAt, UpdatedAt int64`.

**Connection layer:**
- `db.Open(dbPath string) (*sql.DB, error)` — opens SQLite with `PRAGMA foreign_keys = ON` and `PRAGMA journal_mode = WAL`.
- `db.DefaultPath() string` — reads `DB_PATH` env var, defaults to `"recipes.db"`.

**Decisions already made (do not re-evaluate):**
1. Tech stack: Go + Templ + Datastar + SQLite.
2. JWT in Authorization header for all routes except `/login`.
3. Roles are plain text: `admin`, `manager`, `staff`.
4. Configuration via environment variables.
5. Password storage: bcrypt hash + salt.
6. Direct SQL with repository pattern — no ORM.
7. Auth complexity: keep simple for demo. No registration UI, no password reset.

**Approaches already ruled out:**
- Session-based auth (cookies) — JWT specified.
- ORM — direct SQL chosen.
- Separate auth service — monolith chosen.

**Constraints:**
- Go 1.22.
- SQLite as database (`recipes.db` at project root).
- Zero frontend build step.
- Datastar is new — Context7 MCP must be consulted before generating Datastar code (per AGENTS.md). This milestone does not involve Datastar.

### User-Confirmed Decisions (for this spec)

| Decision | Choice |
|----------|--------|
| HTTP router | `github.com/go-chi/chi/v5` — lightweight third-party router with middleware chaining |
| JWT claims | `user_id` (string), `role` (string), `restaurant_ids` ([]string), plus standard `exp`, `iat` |
| Login request format | JSON body: `{"username": "...", "password": "..."}` |
| Login response format | JSON body: `{"token": "...", "user": {"id": "...", "role": "...", "restaurants": ["..."]}}` |
| Auth error response | HTTP 401 with JSON body: `{"error": "unauthorized"}` |
| RBAC error response | HTTP 403 with JSON body: `{"error": "forbidden"}` |
| User info passing | `context.WithValue` — middleware sets claims in request context, handlers retrieve via typed helper |
| Server entry point | `cmd/server/main.go` created in this milestone — wires DB, migrations, router, auth, starts HTTP server |
| JWT secret env var | `JWT_SECRET` — required, no default (server fails to start if unset) |
| JWT expiration | 1 hour |
| JWT signing method | HMAC-SHA256 (HS256) |
| Token location | `Authorization: Bearer <token>` header |

---

## Section 2: Requirements

### 2.1 Package Structure

This milestone introduces the following new packages:

- `internal/auth/` — JWT creation/validation, password verification, claims types, context helpers.
- `internal/handler/` — HTTP handler functions (login handler initially; recipe handlers in milestone 005).
- `cmd/server/` — Application entry point.

No changes to existing packages (`internal/db/`, `internal/store/`, `migrations/`, `cmd/seed/`).

### 2.2 JWT Claims

Define a claims struct that extends `jwt.RegisteredClaims` with application-specific fields.

**Required fields:**

| Field | JWT Claim Key | Type | Description |
|-------|---------------|------|-------------|
| User ID | `user_id` | string | UUID of the authenticated user |
| Role | `role` | string | One of `admin`, `manager`, `staff` |
| Restaurant IDs | `restaurant_ids` | []string | List of restaurant UUIDs the user can access |
| Expiration | `exp` | *jwt.NumericDate | 1 hour from token creation |
| Issued At | `iat` | *jwt.NumericDate | Token creation time |

**Behavior:**
- Token is signed with HMAC-SHA256 using the secret from `JWT_SECRET` env var.
- Token is generated during login and validated on every authenticated request.
- Expired tokens must be rejected with a 401 error.
- Tokens with missing required fields (`user_id`, `role`) must be rejected with a 401 error.

### 2.3 Password Verification

**Behavior:**
- Accept a plaintext password and a bcrypt-hashed password string.
- Use `bcrypt.CompareHashAndPassword` from `golang.org/x/crypto/bcrypt`.
- Return nil on match, error on mismatch.
- This function does NOT interact with the database. It is a pure comparison utility.

### 2.4 Login Endpoint

**Route:** `POST /login`

**Request:**
- Content-Type: `application/json`
- Body: `{"username": "<string>", "password": "<string>"}`

**Success response (200 OK):**
```json
{
  "token": "<jwt-string>",
  "user": {
    "id": "<uuid>",
    "role": "<admin|manager|staff>",
    "restaurants": ["<uuid>", "<uuid>"]
  }
}
```

**Error responses:**

| Status | Body | Condition |
|--------|------|-----------|
| 400 | `{"error": "invalid request body"}` | Malformed JSON or missing fields |
| 401 | `{"error": "invalid credentials"}` | Username not found OR password mismatch |
| 500 | `{"error": "internal server error"}` | Database error, JWT signing failure, or other unexpected error |

**Behavior:**
1. Parse JSON request body. If parsing fails, return 400.
2. Look up user by username via `store.GetUserByUsername`. If `ErrNotFound`, return 401 (do not reveal whether username exists).
3. Compare provided password against stored hash via bcrypt. If mismatch, return 401 (same error message as step 2 — "invalid credentials").
4. Fetch user's restaurant IDs via `store.ListUserRestaurantIDs`.
5. Generate JWT with claims (user_id, role, restaurant_ids, exp, iat).
6. Return 200 with token and user details.

**Edge cases:**
- Empty username or empty password: treated as invalid credentials (401), not a validation error.
- Username with special characters: handled normally (SQLite parameterized query).
- User with no restaurant access: `restaurant_ids` is an empty array `[]`, not null.
- Concurrent login requests for the same user: each produces a distinct token (stateless, no session tracking).

### 2.5 Auth Middleware

**Purpose:** Extract and validate JWT from the `Authorization` header. Set claims in the request context. Reject unauthenticated requests.

**Behavior:**
1. Read the `Authorization` header.
2. If missing or not in `Bearer <token>` format, respond with 401 and `{"error": "unauthorized"}`.
3. Parse and validate the JWT (signature, expiration, required claims).
4. If validation fails (expired, invalid signature, missing claims), respond with 401 and `{"error": "unauthorized"}`.
5. Extract claims and store them in the request context via `context.WithValue`.
6. Call the next handler in the chain.

**Context key:** Use an unexported context key type (not a string) to avoid collisions. Provide an exported helper function to retrieve claims from context.

**Helper function:** A function that accepts `*http.Request` and returns the claims (or an error/nil if not present). Handlers use this instead of accessing context directly.

### 2.6 RBAC Middleware

**Purpose:** Restrict access to handlers based on the authenticated user's role.

**Behavior:**
1. Retrieve claims from the request context (set by auth middleware).
2. If claims are missing (auth middleware not applied), respond with 401.
3. Check if the user's role is in the list of allowed roles for this route.
4. If the role is not allowed, respond with 403 and `{"error": "forbidden"}`.
5. If the role is allowed, call the next handler.

**Interface:** The RBAC middleware must be configurable per-route — the caller specifies which roles are allowed when applying the middleware. For example, a route might require `["admin", "manager"]` while another requires `["admin", "manager", "staff"]`.

### 2.7 HTTP Server Entry Point

**Location:** `cmd/server/main.go`

**Responsibilities:**
1. Read configuration from environment variables:
   - `DB_PATH` — database file path (default: `"recipes.db"`, via `db.DefaultPath()`).
   - `JWT_SECRET` — JWT signing secret (required, no default). If unset, the server must exit with a non-zero status and a clear error message.
   - `PORT` — server listen port (default: `"8080"`).
2. Open database connection via `db.Open`.
3. Run migrations via `db.RunMigrations` using the embedded migrations filesystem.
4. Create chi router with middleware:
   - `chi.Logger` (from `chi/middleware`) for request logging.
   - `chi.Recoverer` (from `chi/middleware`) for panic recovery.
5. Register routes:
   - `POST /login` — login handler (public, no auth middleware).
   - A route group for authenticated routes (auth middleware applied). No specific authenticated routes are registered in this milestone — they are added in milestones 004 and 005. The group exists as a placeholder.
6. Start HTTP server on the configured port.
7. Handle graceful shutdown on SIGINT/SIGTERM (optional but recommended for clean DB connection closure).

**Edge cases:**
- `JWT_SECRET` is empty string: treated as unset, server fails to start.
- `DB_PATH` points to non-existent directory: `db.Open` returns error, server fails to start.
- Port already in use: server fails to start with a clear error.

### 2.8 Makefile Updates

**Existing target `run`:** Already runs `go run ./cmd/server`. No change needed — the entry point now exists.

**No new targets.**

### 2.9 Testing Requirements

**Test file locations:**
- `internal/auth/jwt_test.go` — JWT creation and validation tests.
- `internal/auth/password_test.go` — bcrypt comparison tests.
- `internal/handler/login_test.go` — login endpoint integration tests.

**JWT tests (`internal/auth/jwt_test.go`):**

| Test | Description |
|------|-------------|
| `TestCreateToken` | Create a token with known claims, validate it, verify all fields match. |
| `TestCreateToken_Expired` | Create a token with expiration in the past, validate, verify rejection. |
| `TestValidateToken_InvalidSignature` | Validate a token signed with a different secret, verify rejection. |
| `TestValidateToken_MissingClaims` | Validate a token missing `user_id` or `role`, verify rejection. |
| `TestValidateToken_Malformed` | Validate a garbage string, verify rejection. |

**Password tests (`internal/auth/password_test.go`):**

| Test | Description |
|------|-------------|
| `TestVerifyPassword_Correct` | Hash a password with bcrypt, verify correct password returns nil. |
| `TestVerifyPassword_Incorrect` | Hash a password with bcrypt, verify wrong password returns error. |

**Login handler tests (`internal/handler/login_test.go`):**

These are integration tests that exercise the full login flow against an in-memory SQLite database.

| Test | Description |
|------|-------------|
| `TestLogin_Success` | Seed a user, POST valid credentials, verify 200 with token and user details. |
| `TestLogin_WrongPassword` | Seed a user, POST wrong password, verify 401 with "invalid credentials". |
| `TestLogin_UserNotFound` | POST with non-existent username, verify 401 with "invalid credentials". |
| `TestLogin_MalformedJSON` | POST with invalid JSON body, verify 400. |
| `TestLogin_EmptyBody` | POST with empty body, verify 400. |
| `TestLogin_EmptyUsername` | POST with empty username field, verify 401 (invalid credentials, not validation error). |
| `TestLogin_EmptyPassword` | POST with empty password field, verify 401. |

**Auth middleware tests (in `internal/auth/` or `internal/handler/`):**

| Test | Description |
|------|-------------|
| `TestAuthMiddleware_ValidToken` | Request with valid Bearer token, verify next handler is called and claims are in context. |
| `TestAuthMiddleware_NoHeader` | Request with no Authorization header, verify 401. |
| `TestAuthMiddleware_MalformedHeader` | Request with `Authorization: Token abc` (not Bearer), verify 401. |
| `TestAuthMiddleware_ExpiredToken` | Request with expired token, verify 401. |
| `TestAuthMiddleware_InvalidSignature` | Request with token signed by different secret, verify 401. |

**RBAC middleware tests:**

| Test | Description |
|------|-------------|
| `TestRBACMiddleware_AllowedRole` | Request with role in allowed list, verify next handler is called. |
| `TestRBACMiddleware_DeniedRole` | Request with role not in allowed list, verify 403. |
| `TestRBACMiddleware_NoClaims` | Request without auth middleware applied (no claims in context), verify 401. |

**TDD approach:** Red-green TDD. Write failing test first, then implement to make it pass.

### 2.10 Edge Cases

| Case | Expected Behavior |
|------|-------------------|
| `JWT_SECRET` env var not set | Server exits with non-zero status and clear error message |
| `JWT_SECRET` is empty string | Treated as unset, server exits |
| Login with valid credentials but user has no restaurant access | 200 OK, `restaurants` is `[]` |
| JWT expires during a request | Middleware rejects with 401 (token validated before handler runs) |
| Multiple concurrent logins for same user | Each produces a valid, distinct token |
| Bearer token with extra whitespace | Trimmed before validation |
| Authorization header value is just "Bearer" with no token | 401 unauthorized |
| Login with SQL injection attempt in username | Handled safely by parameterized query in store layer |
| Login with very long username/password | No length limit enforced at handler level (SQLite handles it) |
| RBAC middleware applied without auth middleware | 401 (claims not in context) |
| User with `staff` role accessing admin-only route | 403 forbidden |
| Token with `restaurant_ids` containing non-existent restaurant IDs | Allowed — token is valid; restaurant existence is checked at data access time |

### 2.11 Out of Scope

The following are explicitly NOT part of this milestone:
- Templ templates or frontend (Milestone 004).
- Recipe CRUD endpoints (Milestone 005).
- Search endpoint (Milestone 005).
- Restaurant dropdown selector (Milestone 004).
- Changes to `internal/store/` — the data access layer is complete.
- Changes to `internal/db/` — the connection and migration layers are complete.
- Changes to `migrations/` — the schema is complete.
- Changes to `cmd/seed/` — the seed script is complete.
- Any Datastar integration.
- Registration UI or password reset.
- Token refresh mechanism.
- Rate limiting on login endpoint.
