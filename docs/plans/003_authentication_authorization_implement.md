# Implementation Guide 003: Authentication and Authorization

**Spec:** `docs/plans/003_authentication_authorization_spec.md`

---

## Section 1: Context Summary

**Milestone:** 003 — Authentication and authorization — `/login` endpoint with bcrypt verification, JWT generation with user/restaurant claims, auth middleware, RBAC middleware for role-based endpoint protection.

The database schema and data access layer are complete (milestones 001 and 002). The application has typed Go models and tested repository methods for all 6 tables, but no HTTP server, no authentication, and no way for users to log in. Before frontend templates (milestone 004) or recipe CRUD endpoints (milestone 005) can be built, the system needs a working HTTP server with JWT-based authentication. This milestone creates the `internal/auth/` package (JWT creation/validation, password verification, auth middleware, RBAC middleware), the `internal/handler/` package (login endpoint), and the `cmd/server/` entry point — wiring everything together with the chi router. All auth code is tested against real JWT tokens and real in-memory SQLite databases, not mocks.

---

## Section 2: Implementation Checklist

### Task 1: Add chi router dependency

- [x] Run `go get github.com/go-chi/chi/v5` from the project root to add the chi router to `go.mod`
- [x] Run `go mod tidy` to clean up `go.sum`

### Task 2: Implement JWT claims struct and token functions

- [x] Create `internal/auth/jwt.go`
  - Package declaration: `package auth`
  - Imports: `fmt`, `time`, `github.com/golang-jwt/jwt/v5`
  - Define `Claims` struct embedding `jwt.RegisteredClaims` with fields: `UserID string` (json:"user_id"), `Role string` (json:"role"), `RestaurantIDs []string` (json:"restaurant_ids")
  - Function `CreateToken(secret []byte, userID string, role string, restaurantIDs []string) (string, error)` — build `Claims` with `UserID`, `Role`, `RestaurantIDs`, set `ExpiresAt` to `time.Now().Add(1 * time.Hour)`, set `IssuedAt` to `time.Now()`, create `jwt.NewWithClaims(jwt.SigningMethodHS256, claims)`, call `token.SignedString(secret)`, wrap errors with `fmt.Errorf`
  - Function `ValidateToken(secret []byte, tokenString string) (*Claims, error)` — call `jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (any, error) { if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok { return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"]) }; return secret, nil })`, if parse error return nil wrapped error, if `!token.Valid` return nil error, type-assert `token.Claims.(*Claims)`, verify `claims.UserID != ""` and `claims.Role != ""` (return error if missing), return claims

### Task 3: Write JWT tests (TDD — write tests before ValidateToken validation logic)

- [x] Create `internal/auth/jwt_test.go`
  - Package declaration: `package auth`
  - Imports: `strings`, `testing`, `time`, `github.com/golang-jwt/jwt/v5`
  - `TestCreateToken` — call `CreateToken([]byte("test-secret"), "user-123", "admin", []string{"rest-1", "rest-2"})`, assert no error, assert token is non-empty string with two dots (JWT format)
  - `TestCreateToken_ValidateRoundTrip` — create token with known claims, call `ValidateToken` with same secret, assert no error, assert `claims.UserID == "user-123"`, assert `claims.Role == "admin"`, assert `claims.RestaurantIDs` has length 2 and contains "rest-1" and "rest-2", assert `claims.ExpiresAt` is approximately 1 hour in the future (within 5 seconds), assert `claims.IssuedAt` is approximately now
  - `TestCreateToken_Expired` — create a `Claims` struct manually with `ExpiresAt` set to 1 hour in the past, sign it with `jwt.NewWithClaims`, call `ValidateToken` with the resulting token string, assert error is returned
  - `TestValidateToken_InvalidSignature` — create token with secret "secret-a", validate with secret "secret-b", assert error is returned
  - `TestValidateToken_MissingUserID` — create a token manually with empty `UserID` but valid `Role`, sign it, call `ValidateToken`, assert error is returned (claims validation rejects missing user_id)
  - `TestValidateToken_MissingRole` — create a token manually with valid `UserID` but empty `Role`, sign it, call `ValidateToken`, assert error is returned
  - `TestValidateToken_Malformed` — call `ValidateToken([]byte("secret"), "not-a-jwt")`, assert error is returned
  - `TestValidateToken_EmptyString` — call `ValidateToken([]byte("secret"), "")`, assert error is returned

### Task 4: Implement password verification

- [x] Create `internal/auth/password.go`
  - Package declaration: `package auth`
  - Imports: `fmt`, `golang.org/x/crypto/bcrypt`
  - Function `VerifyPassword(hashedPassword, password string) error` — call `bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))`, wrap error with `fmt.Errorf` on failure, return nil on success

### Task 5: Write password tests

- [x] Create `internal/auth/password_test.go`
  - Package declaration: `package auth`
  - Imports: `testing`, `golang.org/x/crypto/bcrypt`
  - Helper: generate bcrypt hash of "correctpassword" using `bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)` in a `TestMain` or package-level var
  - `TestVerifyPassword_Correct` — call `VerifyPassword(hashed, "correctpassword")`, assert nil error
  - `TestVerifyPassword_Incorrect` — call `VerifyPassword(hashed, "wrongpassword")`, assert non-nil error
  - `TestVerifyPassword_EmptyPassword` — call `VerifyPassword(hashed, "")`, assert non-nil error
  - `TestVerifyPassword_EmptyHash` — call `VerifyPassword("", "password")`, assert non-nil error

### Task 6: Implement context key and claims helper

- [x] Create `internal/auth/context.go`
  - Package declaration: `package auth`
  - Imports: `errors`, `fmt`, `net/http`
  - Define unexported type `contextKey struct{}` (not a string, to avoid collisions)
  - Define unexported var `claimsKey = contextKey{}`
  - Function `ContextWithClaims(r *http.Request, claims *Claims) *http.Request` — returns `r.WithContext(context.WithValue(r.Context(), claimsKey, claims))` (import `context`)
  - Function `ClaimsFromContext(r *http.Request) (*Claims, error)` — call `r.Context().Value(claimsKey)`, type-assert to `*Claims`, if nil return `nil, fmt.Errorf("no claims in context")`, otherwise return claims, nil

### Task 7: Implement auth middleware

- [x] Create `internal/auth/middleware.go`
  - Package declaration: `package auth`
  - Imports: `encoding/json`, `net/http`, `strings`
  - Function `Middleware(secret []byte) func(http.Handler) http.Handler` — returns a closure that: reads `Authorization` header, if missing or not prefixed with `"Bearer "` respond with 401 and `{"error": "unauthorized"}` (set Content-Type to application/json), extract token string by trimming prefix and trimming whitespace, call `ValidateToken(secret, token)`, if error respond with 401, call `ContextWithClaims(r, claims)`, call `next.ServeHTTP(w, r)` with the updated request
  - Use a helper function `writeJSONError(w http.ResponseWriter, status int, message string)` — set `w.Header().Set("Content-Type", "application/json")`, `w.WriteHeader(status)`, `json.NewEncoder(w).Encode(map[string]string{"error": message})` — shared by middleware and handlers

### Task 8: Write auth middleware tests

- [x] Create `internal/auth/middleware_test.go`
  - Package declaration: `package auth`
  - Imports: `net/http`, `net/http/httptest`, `testing`
  - Helper: `testSecret = []byte("test-secret-for-middleware")`
  - Helper: `okHandler` — an `http.HandlerFunc` that writes 200 and `"ok"` (used as the "next" handler in the chain)
  - `TestMiddleware_ValidToken` — create a valid token via `CreateToken(testSecret, "u1", "admin", []string{"r1"})`, build request with `Authorization: Bearer <token>`, wrap `okHandler` with `Middleware(testSecret)`, serve the request, assert 200 status
  - `TestMiddleware_NoHeader` — build request with no Authorization header, serve, assert 401 status, assert body contains `"unauthorized"`
  - `TestMiddleware_MalformedHeader` — build request with `Authorization: Token abc`, serve, assert 401
  - `TestMiddleware_BearerOnly` — build request with `Authorization: Bearer ` (no token after Bearer), serve, assert 401
  - `TestMiddleware_ExpiredToken` — create a token manually with expiration in the past (use `jwt.NewWithClaims` directly), build request with Bearer header, serve, assert 401
  - `TestMiddleware_InvalidSignature` — create token with secret `"other-secret"`, validate with testSecret, serve, assert 401
  - `TestMiddleware_ClaimsInContext` — create valid token, build request, wrap a handler that calls `ClaimsFromContext(r)` and writes the `UserID` to the response body, serve, assert 200, assert body contains the user ID

### Task 9: Implement RBAC middleware

- [x] Create `internal/auth/rbac.go`
  - Package declaration: `package auth`
  - Imports: `net/http`
  - Function `RequireRole(roles ...string) func(http.Handler) http.Handler` — returns a closure that: call `ClaimsFromContext(r)`, if error (no claims) respond with 401 and `{"error": "unauthorized"}`, check if `claims.Role` is in the `roles` slice, if not respond with 403 and `{"error": "forbidden"}`, if allowed call `next.ServeHTTP(w, r)`

### Task 10: Write RBAC middleware tests

- [x] Create `internal/auth/rbac_test.go`
  - Package declaration: `package auth`
  - Imports: `net/http`, `net/http/httptest`, `testing`
  - Helper: reuse `okHandler` and `testSecret` from middleware tests (or define locally)
  - Helper: `makeRequestWithRole(t *testing.T, role string) *http.Request` — create a valid token with the given role via `CreateToken(testSecret, "u1", role, []string{"r1"})`, build request with Bearer header, then wrap with `ContextWithClaims` to simulate auth middleware having run
  - `TestRequireRole_AllowedRole` — create request with role "admin", wrap `okHandler` with `RequireRole("admin", "manager")`, serve, assert 200
  - `TestRequireRole_DeniedRole` — create request with role "staff", wrap `okHandler` with `RequireRole("admin", "manager")`, serve, assert 403, assert body contains `"forbidden"`
  - `TestRequireRole_NoClaims` — build request with no claims in context (no Authorization header, no context value), wrap `okHandler` with `RequireRole("admin")`, serve, assert 401
  - `TestRequireRole_SingleRole` — create request with role "manager", wrap with `RequireRole("manager")`, serve, assert 200
  - `TestRequireRole_AllRolesAllowed` — create request with role "staff", wrap with `RequireRole("admin", "manager", "staff")`, serve, assert 200

### Task 11: Implement login handler

- [x] Create `internal/handler/login.go`
  - Package declaration: `package handler`
  - Imports: `encoding/json`, `errors`, `log`, `net/http`, `lab37/internal/auth`, `lab37/internal/store`
  - Define `LoginHandler` struct with fields: `DB store.DBTX`, `JWTSecret []byte`
  - Method `ServeHTTP(w http.ResponseWriter, r *http.Request)` on `*LoginHandler`:
    1. Define `loginRequest` struct: `Username string`, `Password string` (json tags)
    2. Define `loginResponse` struct: `Token string`, `User userResponse` (json tags)
    3. Define `userResponse` struct: `ID string`, `Role string`, `Restaurants []string` (json tags)
    4. Decode request body into `loginRequest` via `json.NewDecoder(r.Body).Decode(&req)`. If error, respond 400 with `{"error": "invalid request body"}` and return
    5. Call `store.GetUserByUsername(h.DB, req.Username)`. If `errors.Is(err, store.ErrNotFound)`, respond 401 with `{"error": "invalid credentials"}` and return. If other error, respond 500 with `{"error": "internal server error"}` and return
    6. Call `auth.VerifyPassword(user.Password, req.Password)`. If error, respond 401 with `{"error": "invalid credentials"}` and return
    7. Call `store.ListUserRestaurantIDs(h.DB, user.ID)`. If error, respond 500 with `{"error": "internal server error"}` and return
    8. Call `auth.CreateToken(h.JWTSecret, user.ID, user.Role, restaurantIDs)`. If error, respond 500 with `{"error": "internal server error"}` and return
    9. Set `Content-Type: application/json`, write 200, encode `loginResponse{Token: token, User: userResponse{ID: user.ID, Role: user.Role, Restaurants: restaurantIDs}}`
  - Use the same `writeJSONError` helper pattern from auth middleware (or define a local helper in handler package)

### Task 12: Write login handler tests

- [x] Create `internal/handler/login_test.go`
  - Package declaration: `package handler`
  - Imports: `bytes`, `encoding/json`, `net/http`, `net/http/httptest`, `testing`, `lab37/internal/auth`, `lab37/internal/db`, `lab37/internal/store`, `lab37/migrations`, `golang.org/x/crypto/bcrypt`, `github.com/google/uuid`
  - Helper: `setupTestDB(t *testing.T) *sql.DB` — open `:memory:` DB via `db.Open(":memory:")`, run migrations via `db.RunMigrations(database, migrations.FS)`, call `t.Fatal` on error
  - Helper: `seedTestUser(t *testing.T, database *sql.DB, username, password, role string) (userID string, restaurantID string)` — generate UUIDs, bcrypt hash the password, insert user and restaurant and user_restaurant link via raw SQL, return the IDs
  - Helper: `testJWTSecret = []byte("test-jwt-secret")`
  - `TestLogin_Success` — setup DB, seed user "testuser" / "password123" / "admin" with 1 restaurant, create `LoginHandler{DB: db, JWTSecret: testJWTSecret}`, build POST request with JSON body `{"username":"testuser","password":"password123"}`, serve via httptest.NewRecorder, assert 200, decode response, assert `Token` is non-empty, assert `User.ID` matches seeded ID, assert `User.Role` is "admin", assert `User.Restaurants` has length 1
  - `TestLogin_WrongPassword` — setup DB, seed user, build POST with wrong password, serve, assert 401, assert body contains `"invalid credentials"`
  - `TestLogin_UserNotFound` — setup DB (no seed), build POST with non-existent username, serve, assert 401, assert body contains `"invalid credentials"`
  - `TestLogin_MalformedJSON` — setup DB, build POST with body `{"username":`, serve, assert 400, assert body contains `"invalid request body"`
  - `TestLogin_EmptyBody` — setup DB, build POST with empty body, serve, assert 400
  - `TestLogin_EmptyUsername` — setup DB, seed user, build POST with `{"username":"","password":"password123"}`, serve, assert 401 (invalid credentials, not validation error)
  - `TestLogin_EmptyPassword` — setup DB, seed user, build POST with `{"username":"testuser","password":""}`, serve, assert 401
  - `TestLogin_UserWithNoRestaurants` — setup DB, insert user via raw SQL but no user_restaurant links, build POST with valid credentials, serve, assert 200, assert `User.Restaurants` is empty slice (not null)
  - `TestLogin_TokenIsValid` — setup DB, seed user, serve login request, extract token from response, call `auth.ValidateToken(testJWTSecret, token)`, assert no error, assert claims match user ID and role

### Task 13: Implement HTTP server entry point

- [x] Create `cmd/server/main.go`
  - Package declaration: `package main`
  - Imports: `context`, `log`, `net/http`, `os`, `os/signal`, `syscall`, `time`, `github.com/go-chi/chi/v5`, `github.com/go-chi/chi/v5/middleware`, `lab37/internal/auth`, `lab37/internal/db`, `lab37/internal/handler`, `lab37/migrations`
  - In `main()`:
    1. Read `JWT_SECRET` env var via `os.Getenv("JWT_SECRET")`. If empty, `log.Fatal("JWT_SECRET environment variable is required")`
    2. Read `PORT` env var via `os.Getenv("PORT")`. If empty, default to `"8080"`
    3. Open database via `db.Open(db.DefaultPath())`. If error, `log.Fatalf("failed to open database: %v", err)`
    4. Defer `database.Close()`
    5. Run migrations via `db.RunMigrations(database, migrations.FS)`. If error, `log.Fatalf("failed to run migrations: %v", err)`
    6. Create chi router via `chi.NewRouter()`
    7. Apply global middleware: `r.Use(middleware.Logger)`, `r.Use(middleware.Recoverer)`
    8. Create login handler: `loginHandler := &handler.LoginHandler{DB: database, JWTSecret: []byte(jwtSecret)}`
    9. Register public route: `r.Post("/login", loginHandler.ServeHTTP)`
    10. Register authenticated route group (placeholder for milestones 004/005):
        ```
        r.Group(func(r chi.Router) {
            r.Use(auth.Middleware([]byte(jwtSecret)))
            // Future authenticated routes go here
        })
        ```
    11. Create `http.Server{Addr: ":" + port, Handler: r}`
    12. Start server in a goroutine: `go func() { log.Printf("server starting on :%s", port); if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { log.Fatalf("server error: %v", err) } }()`
    13. Wait for interrupt signal: `quit := make(chan os.Signal, 1); signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM); <-quit`
    14. Graceful shutdown: `ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second); defer cancel(); srv.Shutdown(ctx)`

### Task 14: Run full test suite

- [x] Run `go test ./...` from the project root and verify all tests pass with no failures
- [x] Run `go vet ./...` and verify no issues
