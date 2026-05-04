# Implementation Guide 004: Frontend Foundation

**Spec:** `docs/plans/004_frontend_foundation_spec.md`

---

## Section 1: Context Summary

**Milestone:** 004 — Frontend foundation — Templ templates generating, base HTML layout, Datastar script inclusion, restaurant dropdown selector wired to user's accessible restaurants.

The database schema, data access layer, and authentication system are complete (milestones 001–003). The application has a working HTTP server with JWT-based login, but no frontend — users cannot interact with the system through a browser. Before recipe CRUD and search (milestone 005) can be built, the system needs server-rendered HTML pages, a login UI, cookie-based auth for browser requests, and a restaurant dropdown that sets a Datastar signal for downstream reactivity. This milestone creates the `views/` directory (Templ templates for layout, pages, and components), the `static/css/` stylesheet, updates the auth middleware to also read JWT from an httpOnly cookie, adds a browser auth middleware that redirects to `/login`, and creates handlers for browser login, logout, and the home page. All new code is tested against real in-memory SQLite databases and real HTTP requests, not mocks.

---

## Section 2: Implementation Checklist

### Task 1: Add templ dependency and update Makefile

- [x] Run `go get github.com/a-h/templ` from the project root to add the templ package to `go.mod`
- [x] Run `go mod tidy` to clean up `go.sum`
- [x] Edit `Makefile` — add a `generate` target that runs `templ generate` from the project root
- [x] Edit `Makefile` — change the `run` target to depend on `generate` (i.e., `run: generate`)
- [x] Edit `Makefile` — add `generate` to the `.PHONY` list
- [x] Run `make generate` to verify templ is installed and generates successfully (no `.templ` files exist yet, so this should be a no-op)

### Task 2: Create static CSS file

- [x] Create `static/css/style.css` with minimal styles:
  - Reset/base: `box-sizing: border-box`, `margin: 0`, `padding: 0` on `*`, `font-family: system-ui, sans-serif` on `body`
  - Layout: `header` with `display: flex`, `justify-content: space-between`, `align-items: center`, `padding: 1rem`, `background: #f5f5f5`, `border-bottom: 1px solid #ddd`. `main` with `padding: 2rem`, `min-height: calc(100vh - 120px)`. `footer` with `padding: 1rem`, `text-align: center`, `border-top: 1px solid #ddd`, `color: #666`
  - Login form: `.login-container` centered with `max-width: 400px`, `margin: 4rem auto`, `padding: 2rem`, `border: 1px solid #ddd`, `border-radius: 8px`. `input` fields with `width: 100%`, `padding: 0.5rem`, `margin-bottom: 1rem`, `border: 1px solid #ccc`, `border-radius: 4px`. `button` with `width: 100%`, `padding: 0.75rem`, `background: #007bff`, `color: white`, `border: none`, `border-radius: 4px`, `cursor: pointer`
  - Error message: `.error` with `color: #dc3545`, `margin-bottom: 1rem`, `padding: 0.5rem`, `background: #f8d7da`, `border-radius: 4px`
  - Dropdown: `select` with `padding: 0.5rem`, `border: 1px solid #ccc`, `border-radius: 4px`
  - Header elements: `.header-left` and `.header-right` for flex layout. `.user-info` for username display. `.logout-link` styled as a link

### Task 3: Create base layout template

- [x] Create `views/layouts/base.templ`
  - Package declaration: `package layouts`
  - Import: `github.com/a-h/templ`
  - Component `Base(title string, content templ.Component)` — renders:
    - `<!DOCTYPE html>`
    - `<html lang="en">`
    - `<head>`: `<meta charset="UTF-8">`, `<meta name="viewport" content="width=device-width, initial-scale=1.0">`, `<title>{title}</title>`, `<link rel="stylesheet" href="/static/css/style.css">`, `<script type="module" src="https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.0/bundles/datastar.js"></script>`
    - `<body>`: call `@content` to render the page content, `</body>`, `</html>`

### Task 4: Create header component

- [x] Create `views/components/header.templ`
  - Package declaration: `package components`
  - Import: `lab37/internal/store`
  - Component `Header(username string, restaurants []store.Restaurant)` — renders:
    - `<header>` element
    - Left side: `<span class="header-left">Recipe Manager</span>`
    - Right side: `<span class="header-right">`
      - `<span class="user-info">{username}</span>`
      - `<select data-bind:selected-restaurant>`:
        - If `len(restaurants) == 0`: `<option value="" disabled selected>No restaurants available</option>` and `disabled` attribute on select
        - If `len(restaurants) > 0`: `<option value="" disabled selected>Select a restaurant</option>`, then `for _, r := range restaurants { <option value={r.ID}>{r.Name}</option> }`
      - `<a href="/logout" class="logout-link">Logout</a>`
    - Close `</header>`

### Task 5: Create footer component

- [x] Create `views/components/footer.templ`
  - Package declaration: `package components`
  - Component `Footer()` — renders:
    - `<footer>` element with text: `© 2026 Recipe Manager`

### Task 6: Create login page template

- [x] Create `views/pages/login.templ`
  - Package declaration: `package pages`
  - Component `Login(errorMsg string)` — renders:
    - `<div class="login-container">`
    - `<h1>Log In</h1>`
    - If `errorMsg != ""`: `<div class="error">{errorMsg}</div>`
    - `<form method="POST" action="/login/browser">`
    - `<label for="username">Username</label>`, `<input type="text" id="username" name="username" required>`
    - `<label for="password">Password</label>`, `<input type="password" id="password" name="password" required>`
    - `<button type="submit">Log In</button>`
    - `</form>`, `</div>`

### Task 7: Create home page template

- [x] Create `views/pages/home.templ`
  - Package declaration: `package pages`
  - Import: `lab37/internal/store`, `lab37/views/components`
  - Component `Home(username string, restaurants []store.Restaurant)` — renders:
    - Call `@components.Header(username, restaurants)`
    - `<main>`: `<h1>Welcome, {username}</h1>`, `<p>Select a restaurant from the dropdown to get started.</p>`
    - Call `@components.Footer()`

### Task 8: Run templ generate and verify compilation

- [x] Run `make generate` from the project root to compile all `.templ` files into `_templ.go` files
- [x] Run `go build ./...` to verify the generated code compiles without errors
- [x] Verify that `_templ.go` files exist alongside each `.templ` file in `views/`

### Task 9: Update auth middleware to support cookies (TDD)

- [x] Edit `internal/auth/middleware_test.go` — add new test cases:
  - `TestMiddleware_ValidCookie` — create a valid token via `CreateToken(testSecret, "u1", "admin", []string{"r1"})`, build request with `Cookie: auth_token=<token>` (no Authorization header), wrap `okHandler` with `Middleware(testSecret)`, serve, assert 200 status, assert claims in context
  - `TestMiddleware_ExpiredCookie` — create an expired token manually, build request with cookie, serve, assert 401
  - `TestMiddleware_InvalidCookie` — build request with `Cookie: auth_token=garbage`, serve, assert 401
  - `TestMiddleware_AuthHeaderTakesPrecedence` — create two valid tokens with different user IDs (e.g., "u1" and "u2"), set both Authorization header (with "u2" token) and cookie (with "u1" token), serve, assert 200, assert claims.UserID == "u2" (Authorization header wins)
- [x] Run `go test ./internal/auth/...` and verify the new tests fail (red)
- [x] Edit `internal/auth/middleware.go` — update `Middleware` function:
  - After checking the Authorization header (and if it's missing/empty), add a fallback: check `r.Cookie("auth_token")`
  - If the cookie exists and has a non-empty value, call `ValidateToken(secret, cookie.Value)`
  - If validation succeeds, set claims in context and call `next.ServeHTTP`
  - If validation fails or cookie doesn't exist, fall through to the existing 401 response
  - Priority: Authorization header first, cookie second
- [x] Run `go test ./internal/auth/...` and verify all tests pass (green)

### Task 10: Implement browser auth middleware (TDD)

- [x] Create `internal/auth/browser_middleware_test.go`
  - Package declaration: `package auth`
  - Imports: `net/http`, `net/http/httptest`, `testing`
  - `TestBrowserAuth_ValidCookie` — create a valid token, build request with `Cookie: auth_token=<token>`, wrap `okHandler` with `BrowserMiddleware(testSecret)`, serve, assert 200, assert claims in context via `ClaimsFromContext`
  - `TestBrowserAuth_NoCookie` — build request with no cookie, serve, assert 302 status, assert `Location` header is `/login`
  - `TestBrowserAuth_ExpiredCookie` — create expired token, build request with cookie, serve, assert 302, assert `Location` is `/login`
  - `TestBrowserAuth_InvalidCookie` — build request with `Cookie: auth_token=garbage`, serve, assert 302, assert `Location` is `/login`
- [x] Run `go test ./internal/auth/...` and verify the new tests fail (red)
- [x] Create `internal/auth/browser_middleware.go`
  - Package declaration: `package auth`
  - Imports: `net/http`
  - Function `BrowserMiddleware(secret []byte) func(http.Handler) http.Handler` — returns a closure that:
    1. Read `auth_token` cookie via `r.Cookie("auth_token")`
    2. If cookie is missing or value is empty, redirect to `/login` with `http.StatusFound`
    3. Call `ValidateToken(secret, cookie.Value)`
    4. If error, redirect to `/login` with `http.StatusFound`
    5. Call `ContextWithClaims(r, claims)`
    6. Call `next.ServeHTTP(w, r)` with the updated request
- [x] Run `go test ./internal/auth/...` and verify all tests pass (green)

### Task 11: Implement browser login handler (TDD)

- [x] Create `internal/handler/login_browser_test.go`
  - Package declaration: `package handler`
  - Imports: `net/http`, `net/http/httptest`, `net/url`, `strings`, `testing`, `lab37/internal/auth`, `lab37/internal/db`, `lab37/migrations`, `golang.org/x/crypto/bcrypt`, `github.com/google/uuid`
  - Reuse `setupTestDB` and `seedTestUser` helpers from `login_test.go` (or define locally)
  - `TestBrowserLogin_Success` — setup DB, seed user "testuser" / "password123" / "admin" with 1 restaurant, create `BrowserLoginHandler{DB: db, JWTSecret: testJWTSecret}`, build POST request with `Content-Type: application/x-www-form-urlencoded` and body `username=testuser&password=password123`, serve via httptest.NewRecorder, assert 302 status, assert `Location` header is `/`, assert `Set-Cookie` header contains `auth_token=` and `HttpOnly` and `Path=/` and `Max-Age=3600`
  - `TestBrowserLogin_WrongPassword` — setup DB, seed user, build POST with wrong password, serve, assert 401, assert body contains "Invalid username or password", assert no `Set-Cookie` header
  - `TestBrowserLogin_UserNotFound` — setup DB (no seed), build POST with non-existent username, serve, assert 401, assert body contains "Invalid username or password"
  - `TestBrowserLogin_EmptyUsername` — setup DB, seed user, build POST with empty username, serve, assert 401
  - `TestBrowserLogin_EmptyPassword` — setup DB, seed user, build POST with empty password, serve, assert 401
  - `TestBrowserLogin_RedirectToHome` — setup DB, seed user, serve login request, follow redirect, assert the redirected page renders (requires home page handler to exist — if not yet implemented, this test can be deferred or the redirect target can be a simple placeholder)
- [x] Run `go test ./internal/handler/...` and verify the new tests fail (red)
- [x] Create `internal/handler/login_browser.go`
  - Package declaration: `package handler`
  - Imports: `errors`, `log`, `net/http`, `lab37/internal/auth`, `lab37/internal/store`, `lab37/views/pages`
  - Define `BrowserLoginHandler` struct with fields: `DB store.DBTX`, `JWTSecret []byte`
  - Method `ServeHTTP(w http.ResponseWriter, r *http.Request)` on `*BrowserLoginHandler`:
    1. Call `r.ParseForm()`. If error, respond 400 with rendered login page with error "Invalid request."
    2. Get `username` from `r.FormValue("username")` and `password` from `r.FormValue("password")`
    3. Call `store.GetUserByUsername(h.DB, username)`. If `errors.Is(err, store.ErrNotFound)`, render login page with error "Invalid username or password." and return 401. If other error, render login page with error "Something went wrong." and return 500
    4. Call `auth.VerifyPassword(user.Password, password)`. If error, render login page with error "Invalid username or password." and return 401
    5. Call `store.ListUserRestaurantIDs(h.DB, user.ID)`. If error, render login page with error "Something went wrong." and return 500
    6. If `restaurantIDs` is nil, set to `[]string{}`
    7. Call `auth.CreateToken(h.JWTSecret, user.ID, user.Role, restaurantIDs)`. If error, render login page with error "Something went wrong." and return 500
    8. Set cookie: `http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: 3600})`
    9. Redirect to `/` with `http.StatusFound`
  - Helper: `renderLoginPage(w http.ResponseWriter, errorMsg string)` — calls `pages.Login(errorMsg).Render(context.Background(), w)` (import `context`)
- [x] Run `go test ./internal/handler/...` and verify all tests pass (green)

### Task 12: Implement login page handler (TDD)

- [x] Create `internal/handler/login_page_test.go`
  - Package declaration: `package handler`
  - Imports: `net/http`, `net/http/httptest`, `strings`, `testing`
  - `TestLoginPage_Renders` — create `LoginPageHandler{}`, build GET request to `/login`, serve, assert 200, assert body contains `<form`, `username`, `password`, `Log In`
  - `TestLoginPage_WithError` — create `LoginPageHandler{}`, build GET request to `/login?error=Test+error`, serve, assert 200, assert body contains "Test error"
  - `TestLoginPage_AlreadyAuthenticated` — create a valid token, build GET request with `Cookie: auth_token=<token>`, serve, assert 302 redirect to `/`
- [x] Run `go test ./internal/handler/...` and verify the new tests fail (red)
- [x] Create `internal/handler/login_page.go`
  - Package declaration: `package handler`
  - Imports: `context`, `net/http`, `lab37/internal/auth`, `lab37/views/pages`
  - Define `LoginPageHandler` struct with field: `JWTSecret []byte`
  - Method `ServeHTTP(w http.ResponseWriter, r *http.Request)` on `*LoginPageHandler`:
    1. Check if user is already authenticated: read `auth_token` cookie, if present call `auth.ValidateToken(h.JWTSecret, cookie.Value)`. If valid, redirect to `/` with `http.StatusFound` and return
    2. Get error message from query param: `errorMsg := r.URL.Query().Get("error")`
    3. Render login page: `pages.Login(errorMsg).Render(r.Context(), w)`
- [x] Run `go test ./internal/handler/...` and verify all tests pass (green)

### Task 13: Implement logout handler (TDD)

- [x] Create `internal/handler/logout_test.go`
  - Package declaration: `package handler`
  - Imports: `net/http`, `net/http/httptest`, `strings`, `testing`
  - `TestLogout_ClearsCookie` — create `LogoutHandler{}`, build GET request to `/logout`, serve, assert 302, assert `Location` is `/login`, assert `Set-Cookie` header contains `auth_token=` and `Max-Age=-1` (or `Max-Age<0`)
  - `TestLogout_WithoutCookie` — create `LogoutHandler{}`, build GET request with no cookie, serve, assert 302, assert `Location` is `/login`
- [x] Run `go test ./internal/handler/...` and verify the new tests fail (red)
- [x] Create `internal/handler/logout.go`
  - Package declaration: `package handler`
  - Imports: `net/http`
  - Define `LogoutHandler` struct (empty)
  - Method `ServeHTTP(w http.ResponseWriter, r *http.Request)` on `*LogoutHandler`:
    1. Set cookie: `http.SetCookie(w, &http.Cookie{Name: "auth_token", Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})`
    2. Redirect to `/login` with `http.StatusFound`
- [x] Run `go test ./internal/handler/...` and verify all tests pass (green)

### Task 14: Implement home page handler (TDD)

- [x] Create `internal/handler/home_test.go`
  - Package declaration: `package handler`
  - Imports: `net/http`, `net/http/httptest`, `strings`, `testing`, `lab37/internal/auth`, `lab37/internal/db`, `lab37/migrations`, `golang.org/x/crypto/bcrypt`, `github.com/google/uuid`
  - Reuse `setupTestDB` helper from `login_test.go`
  - Helper: `seedUserWithRestaurants(t *testing.T, database *sql.DB, username string, restaurantNames []string) (userID string)` — creates user with bcrypt-hashed "password123", creates restaurants, links them, returns user ID
  - Helper: `makeAuthenticatedRequest(t *testing.T, secret []byte, userID string) *http.Request` — creates a valid token for the user, builds GET request with `Cookie: auth_token=<token>`
  - `TestHomePage_Renders` — setup DB, seed user with 1 restaurant, create `HomeHandler{DB: db, JWTSecret: testJWTSecret}`, build authenticated request, serve, assert 200, assert body contains "Recipe Manager" (header), "Welcome" (main content), "Recipe Manager" (footer)
  - `TestHomePage_ShowsRestaurants` — setup DB, seed user with 2 restaurants ("The Rusty Spoon", "Copper Kettle"), build authenticated request, serve, assert 200, assert body contains both restaurant names in `<option>` elements
  - `TestHomePage_NoRestaurants` — setup DB, seed user with 0 restaurants, build authenticated request, serve, assert 200, assert body contains "No restaurants available"
  - `TestHomePage_Unauthenticated` — setup DB, create `HomeHandler{DB: db, JWTSecret: testJWTSecret}`, build request with no cookie, wrap with `auth.BrowserMiddleware(testJWTSecret)`, serve, assert 302 redirect to `/login`
- [x] Run `go test ./internal/handler/...` and verify the new tests fail (red)
- [x] Create `internal/handler/home.go`
  - Package declaration: `package handler`
  - Imports: `log`, `net/http`, `lab37/internal/auth`, `lab37/internal/store`, `lab37/views/pages`
  - Define `HomeHandler` struct with fields: `DB store.DBTX`, `JWTSecret []byte`
  - Method `ServeHTTP(w http.ResponseWriter, r *http.Request)` on `*HomeHandler`:
    1. Call `auth.ClaimsFromContext(r)`. If error, redirect to `/login` with `http.StatusFound` and return
    2. Call `store.GetUserByID(h.DB, claims.UserID)`. If error, log and respond 500
    3. Call `store.ListRestaurantsByUserID(h.DB, claims.UserID)`. If error, log and respond 500
    4. Render: `pages.Home(user.Username, restaurants).Render(r.Context(), w)`
- [x] Run `go test ./internal/handler/...` and verify all tests pass (green)

### Task 15: Update server entry point with new routes

- [x] Edit `cmd/server/main.go` — add imports: `lab37/views/pages` (if needed for any direct rendering), `net/http` (already imported)
- [x] Edit `cmd/server/main.go` — create new handlers after the existing `loginHandler`:
  - `loginPageHandler := &handler.LoginPageHandler{JWTSecret: []byte(jwtSecret)}`
  - `browserLoginHandler := &handler.BrowserLoginHandler{DB: database, JWTSecret: []byte(jwtSecret)}`
  - `logoutHandler := &handler.LogoutHandler{}`
  - `homeHandler := &handler.HomeHandler{DB: database, JWTSecret: []byte(jwtSecret)}`
- [x] Edit `cmd/server/main.go` — register public routes (before the authenticated group):
  - `r.Get("/login", loginPageHandler.ServeHTTP)`
  - `r.Post("/login/browser", browserLoginHandler.ServeHTTP)`
  - `r.Get("/logout", logoutHandler.ServeHTTP)`
  - `r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))`
- [x] Edit `cmd/server/main.go` — add a browser route group (after the existing API group):
  ```
  r.Group(func(r chi.Router) {
      r.Use(auth.BrowserMiddleware([]byte(jwtSecret)))
      r.Get("/", homeHandler.ServeHTTP)
  })
  ```
- [x] Run `go build ./cmd/server` to verify compilation

### Task 16: Run full test suite

- [x] Run `go test ./...` from the project root and verify all tests pass with no failures
- [x] Run `go vet ./...` and verify no issues
- [x] Run `make generate && go build ./...` to verify the full build pipeline works
