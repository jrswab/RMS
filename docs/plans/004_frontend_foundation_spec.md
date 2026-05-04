# Spec 004: Frontend Foundation

## Section 1: Context & Constraints

### Milestone Entry

> 004: Frontend foundation — Templ templates generating, base HTML layout, Datastar script inclusion, restaurant dropdown selector wired to user's accessible restaurants

### Research Findings — Relevant Context

**Codebase structure (post-milestone 003):**

```
lab37/
├── AGENTS.md
├── README.md
├── Makefile               # Targets: run, test, tidy, clean, seed
├── go.mod                 # module lab37, Go 1.22
├── go.sum
├── cmd/
│   ├── seed/
│   │   ├── main.go        # Seed data (raw SQL)
│   │   └── seed_test.go
│   └── server/
│       └── main.go        # HTTP server with chi router, auth middleware, login handler
├── docs/plans/
│   ├── RMS000_design.md
│   ├── RMS_recipe_manager_milestones.md
│   ├── 001_database_schema_spec.md
│   ├── 001_database_schema_implement.md
│   ├── 002_data_access_layer_spec.md
│   ├── 002_data_access_layer_implement.md
│   ├── 003_authentication_authorization_spec.md
│   └── 003_authentication_authorization_implement.md
├── internal/
│   ├── auth/
│   │   ├── context.go     # ContextWithClaims, ClaimsFromContext
│   │   ├── jwt.go         # CreateToken, ValidateToken, Claims struct
│   │   ├── jwt_test.go
│   │   ├── middleware.go   # Middleware(secret) — reads Authorization header, returns 401 on failure
│   │   ├── middleware_test.go
│   │   ├── password.go    # VerifyPassword (bcrypt)
│   │   ├── password_test.go
│   │   ├── rbac.go        # RequireRole(roles...)
│   │   └── rbac_test.go
│   ├── db/
│   │   ├── db.go          # Open(dbPath) → *sql.DB, DefaultPath()
│   │   ├── db_test.go
│   │   ├── migrate.go     # RunMigrations(db, fs.FS)
│   │   └── migrate_test.go
│   ├── handler/
│   │   ├── login.go       # LoginHandler — POST /login (JSON API)
│   │   └── login_test.go
│   └── store/
│       ├── store.go       # DBTX interface, ErrNotFound sentinel
│       ├── models.go      # User, Restaurant, Food, UserRestaurant, Recipe, Ingredient structs
│       ├── user.go        # GetUserByUsername, GetUserByID
│       ├── restaurant.go  # GetRestaurantByID, ListRestaurantsByUserID
│       ├── food.go        # GetFoodByID, ListFood
│       ├── recipe.go      # GetRecipeByID, SearchRecipes, CreateRecipe, UpdateRecipe, DeleteRecipe
│       ├── ingredient.go  # ListIngredientsByRecipeID, ReplaceIngredients
│       ├── user_restaurant.go  # ListUserRestaurantIDs
│       └── *_test.go      # Full TDD test coverage
├── migrations/
│   ├── embed.go           # //go:embed *.up.sql → var FS embed.FS
│   └── 001_create_tables.up.sql
└── static/                # Empty — to be populated
```

**Dependencies (go.mod):**
- `github.com/go-chi/chi/v5 v5.2.5` — HTTP router
- `github.com/golang-jwt/jwt/v5 v5.3.1` — JWT library
- `github.com/google/uuid v1.6.0` — UUID generation
- `github.com/mattn/go-sqlite3 v1.14.22` — SQLite driver
- `golang.org/x/crypto v0.33.0` — bcrypt password hashing

**Store layer methods relevant to this milestone:**
- `store.ListRestaurantsByUserID(db DBTX, userID string) ([]Restaurant, error)` — returns full Restaurant structs (ID + Name) for a user. Used to populate the restaurant dropdown.
- `store.Restaurant` struct: `ID, Name string`.
- `store.GetUserByID(db DBTX, id string) (*User, error)` — returns user by ID. Used to display username in the header.

**Auth layer relevant to this milestone:**
- `auth.Claims` struct: `UserID string`, `Role string`, `RestaurantIDs []string`, plus standard JWT registered claims.
- `auth.ClaimsFromContext(r *http.Request) (*Claims, error)` — retrieves claims from request context.
- `auth.Middleware(secret []byte) func(http.Handler) http.Handler` — validates JWT from `Authorization: Bearer <token>` header. Returns 401 JSON on failure.
- `auth.RequireRole(roles ...string) func(http.Handler) http.Handler` — restricts by role.

**Current auth middleware behavior (milestone 003):**
- Reads JWT from `Authorization: Bearer <token>` header only.
- On failure: returns HTTP 401 with `{"error": "unauthorized"}` JSON body.
- Does NOT read from cookies.
- Does NOT redirect to login page.

**Current server entry point (`cmd/server/main.go`):**
- Reads `JWT_SECRET` (required), `PORT` (default 8080), `DB_PATH` (default `recipes.db`).
- Opens DB, runs migrations, creates chi router with Logger and Recoverer middleware.
- Registers `POST /login` (public) and an empty authenticated route group.
- Graceful shutdown on SIGINT/SIGTERM.

**Decisions already made (do not re-evaluate):**
1. Tech stack: Go + Templ + Datastar + SQLite.
2. JWT in Authorization header for API clients.
3. Roles are plain text: `admin`, `manager`, `staff`.
4. Configuration via environment variables.
5. Password storage: bcrypt hash + salt.
6. Direct SQL with repository pattern — no ORM.
7. Auth complexity: keep simple for demo. No registration UI, no password reset.
8. chi router for HTTP routing.
9. JWT expiration: 1 hour.
10. JWT signing: HMAC-SHA256 (HS256).

**Approaches already ruled out (do not re-evaluate):**
- Separate frontend SPA (React/Vue) — server-rendered HTML chosen.
- Session-based auth (cookies) for API routes — JWT in Authorization header specified.
- ORM — direct SQL chosen.
- Separate auth service — monolith chosen.

**Constraints:**
- Go 1.22.
- SQLite as database (`recipes.db` at project root).
- Zero frontend build step — no npm, no webpack, no bundler. Templ CLI is the only required tool.
- Datastar is new — Context7 MCP must be consulted before generating Datastar code (per AGENTS.md).

**Datastar patterns (from Context7 research):**
- Datastar is included via a single `<script>` tag (CDN or local file).
- Datastar uses HTML attributes (`data-bind`, `data-on:click`, `data-on:change`, `data-init`, etc.) for reactivity.
- `data-bind:signal-name` creates a two-way binding between an input/select element and a Datastar signal.
- `data-on:event="@get('/endpoint')"` triggers a fetch request to the server.
- `data-init="@get('/endpoint')"` initializes a component by fetching HTML from the server.
- Server responds with SSE events that patch the DOM (`sse.PatchElements`), update signals (`sse.MarshalAndPatchSignals`), or redirect (`sse.Redirect`).
- Signals are sent with every request (query params for GET, JSON body for others).
- Datastar sends a `Datastar-Request: true` header with requests.

**Templ patterns (from Context7 research):**
- `.templ` files define Go-backed HTML components.
- `templ generate` compiles `.templ` files into `_templ.go` files.
- Components are rendered via `component.Render(ctx, w)` or served via `templ.Handler(component)`.
- Components accept parameters and can compose other components.

### User-Confirmed Decisions (for this spec)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Login page scope | Include in this milestone | Without a login UI, there's no way to use the frontend |
| JWT client storage | httpOnly cookie | More secure than localStorage; not accessible to JavaScript (XSS protection) |
| Datastar inclusion | CDN link | Zero build step; `https://cdn.jsdelivr.net/gh/starfederation/datastar@v1.0.0/bundles/datastar.js` |
| CSS approach | Plain CSS in `static/` | Simplest for a demo; no framework overhead |
| Restaurant dropdown behavior | Updates a Datastar signal | Other components (search, recipe list in milestone 005) react to this signal |
| Logout mechanism | Include in this milestone | Essential for a complete auth flow |
| Base layout | Full layout: header + main + footer | Header shows app name, restaurant dropdown, user info, logout. Footer is minimal. |

---

## Section 2: Requirements

### 2.1 Templ Setup and Tooling

**Dependency:** Add `github.com/a-h/templ` to `go.mod`.

**Makefile target:** Add a `generate` target that runs `templ generate` from the project root. The `run` target must depend on `generate` so that `.templ` files are always compiled before the server starts.

**Directory structure:**

```
views/
├── layouts/
│   └── base.templ
├── pages/
│   ├── login.templ
│   └── home.templ
└── components/
    ├── header.templ
    └── footer.templ
```

**Generated files:** `templ generate` produces `*_templ.go` files alongside each `.templ` file. These generated files are Go source code and are compiled into the binary.

### 2.2 Static Assets

**Directory:** `static/css/style.css`

**Purpose:** Minimal custom CSS for the application. No CSS framework.

**Required styles:**
- Base layout: header, main content area, footer positioning.
- Login form: centered form with inputs and button.
- Restaurant dropdown: styled select element.
- Basic typography: readable fonts, appropriate sizing.
- Error message styling: visible error text for login failures.

**Serving:** Static files are served from the `/static/` URL path. The server registers a file server handler for this path.

### 2.3 Base HTML Layout

**Template:** `views/layouts/base.templ`

**Responsibilities:**
- Renders the full HTML document (`<!DOCTYPE html>`, `<html>`, `<head>`, `<body>`).
- Includes the Datastar CDN script in the `<head>`.
- Includes the CSS stylesheet link in the `<head>`.
- Renders a header component.
- Renders a content slot (the page-specific content).
- Renders a footer component.

**Content slot:** The base layout accepts a `templ.Component` parameter for the main content area. Each page passes its content to the layout.

### 2.4 Header Component

**Template:** `views/components/header.templ`

**Responsibilities:**
- Displays the application name (e.g., "Recipe Manager").
- Displays the authenticated user's username.
- Displays the restaurant dropdown selector.
- Displays a logout link/button.

**Parameters:** The header accepts the user's username and the list of restaurants the user can access.

**Restaurant dropdown:**
- A `<select>` element populated with the user's accessible restaurants.
- Uses `data-bind:selected-restaurant` to create a Datastar signal that updates when the selection changes.
- The signal name is `selected-restaurant` (accessible as `$selectedRestaurant` in Datastar expressions).
- If the user has no restaurants, the dropdown is disabled and shows "No restaurants available".
- The first option is a placeholder: "Select a restaurant" with an empty value.

**Logout:** A link or button that navigates to `/logout`.

### 2.5 Footer Component

**Template:** `views/components/footer.templ`

**Responsibilities:**
- Displays a minimal footer (e.g., application name and year).
- No interactive elements.

### 2.6 Login Page

**Template:** `views/pages/login.templ`

**Responsibilities:**
- Renders a login form with username and password inputs and a submit button.
- Displays an error message if login fails (passed as a template parameter).
- The form submits via standard HTML form POST (not Datastar) to `/login/browser`.
- Uses `application/x-www-form-urlencoded` content type.

**Parameters:** An error string. If non-empty, the template displays the error message above the form.

**Behavior:**
- If the user is already authenticated (valid cookie), the login page redirects to `/`.
- The form has two fields: `username` (text) and `password` (password).
- The form has a submit button labeled "Log In".

### 2.7 Auth Middleware Updates

**Current behavior (milestone 003):** The auth middleware reads JWT from the `Authorization: Bearer <token>` header only. On failure, it returns 401 JSON.

**Required changes:**
1. The auth middleware must also check for a JWT in an httpOnly cookie named `auth_token`.
2. Priority: check `Authorization` header first. If not present, check the `auth_token` cookie.
3. If a valid JWT is found (from either source), set claims in context and continue.
4. If no valid JWT is found, return 401 JSON (unchanged failure behavior for API routes).

**Why this change:** The browser stores the JWT in an httpOnly cookie (set during login). The middleware needs to read from this cookie to authenticate browser requests. The Authorization header is still supported for API clients.

**Backward compatibility:** Existing API clients that send `Authorization: Bearer <token>` continue to work unchanged. The cookie check is additive.

**Existing tests:** The auth middleware tests must be updated to cover cookie-based authentication. New test cases:
- Request with valid cookie (no Authorization header) → 200, claims in context.
- Request with invalid/expired cookie → 401.
- Request with both Authorization header and cookie → Authorization header takes precedence.

### 2.8 Browser Auth Middleware

**Purpose:** A new middleware for browser routes that redirects unauthenticated users to `/login` instead of returning 401 JSON.

**Behavior:**
1. Check for JWT in the `auth_token` cookie.
2. If valid, set claims in context and continue.
3. If invalid or missing, redirect to `/login` with HTTP 302.

**Why a separate middleware:** The existing auth middleware returns 401 JSON (appropriate for API routes). Browser routes need a redirect instead. A separate middleware keeps the concerns separated and avoids modifying the existing middleware's behavior.

**Usage:** Applied to browser routes (GET `/`, etc.) instead of the existing auth middleware.

### 2.9 Browser Login Handler

**Route:** `POST /login/browser`

**Purpose:** Handles login form submission from the browser. Sets an httpOnly cookie and redirects on success. Re-renders the login page with an error on failure.

**Request:**
- Content-Type: `application/x-www-form-urlencoded`
- Body: `username=<string>&password=<string>`

**Success behavior:**
1. Validate credentials (same logic as the existing JSON login handler: `store.GetUserByUsername`, `auth.VerifyPassword`).
2. Generate JWT via `auth.CreateToken`.
3. Set an httpOnly cookie named `auth_token` containing the JWT string.
4. Cookie attributes:
   - `HttpOnly: true` — not accessible to JavaScript.
   - `SameSite: Lax` — CSRF protection.
   - `Path: /` — sent with all requests.
   - `MaxAge: 3600` — matches JWT expiration (1 hour).
   - `Secure: false` for local development (HTTP). Configurable for production (HTTPS).
5. Redirect to `/` with HTTP 302.

**Failure behavior:**
1. Re-render the login page with an error message: "Invalid username or password."
2. Return HTTP 401.
3. Do NOT set a cookie.

**Edge cases:**
- Empty username or password: treated as invalid credentials (same error message).
- User with no restaurant access: login succeeds, redirect to `/` (dropdown will show "No restaurants available").

### 2.10 Logout Handler

**Route:** `GET /logout`

**Purpose:** Clears the authentication cookie and redirects to the login page.

**Behavior:**
1. Set the `auth_token` cookie with an empty value and `MaxAge: -1` (expires immediately).
2. Redirect to `/login` with HTTP 302.

**Auth:** This route does NOT require authentication. A user should be able to log out even if their token is expired.

### 2.11 Home Page

**Template:** `views/pages/home.templ`

**Route:** `GET /` (authenticated, browser auth middleware)

**Purpose:** The main page after login. Displays the header (with restaurant dropdown) and a main content area. In milestone 005, this area will contain search and recipe display.

**Handler behavior:**
1. Read JWT claims from context (set by browser auth middleware).
2. Fetch user's restaurants via `store.ListRestaurantsByUserID(db, claims.UserID)`.
3. Fetch user details via `store.GetUserByID(db, claims.UserID)` (for displaying username in header).
4. Render the home page template with the restaurants and user details.

**Main content area:** For this milestone, the main content area is empty or shows a welcome message. It is a placeholder for milestone 005 content (search, recipe display).

**Edge cases:**
- User with no restaurants: header shows the dropdown disabled with "No restaurants available".
- User with one restaurant: dropdown shows the single restaurant, pre-selected.

### 2.12 Static File Serving

**Route:** `GET /static/*`

**Purpose:** Serve static files (CSS) from the `static/` directory.

**Implementation:** Use chi's `FileServer` or `http.StripPrefix` with `http.FileServer` to serve files from the `static/` directory at the `/static/` URL path.

### 2.13 Route Registration

**Updated route structure in `cmd/server/main.go`:**

| Route | Method | Middleware | Handler | Description |
|-------|--------|------------|---------|-------------|
| `/login` | GET | None | Login page handler | Renders login form |
| `/login` | POST | None | LoginHandler (existing) | JSON API login |
| `/login/browser` | POST | None | Browser login handler | Form login, sets cookie |
| `/logout` | GET | None | Logout handler | Clears cookie, redirects |
| `/static/*` | GET | None | Static file server | Serves CSS |
| `/` | GET | Browser auth | Home page handler | Main page |

**Authenticated route group (existing):** The existing authenticated route group in `cmd/server/main.go` remains for API routes (milestone 005). It uses `auth.Middleware` (returns 401 on failure).

**Browser route group (new):** A new route group for browser routes uses the browser auth middleware (redirects to `/login` on failure).

### 2.14 Testing Requirements

**Test file locations:**
- `internal/handler/login_page_test.go` — login page handler tests.
- `internal/handler/login_browser_test.go` — browser login handler tests.
- `internal/handler/logout_test.go` — logout handler tests.
- `internal/handler/home_test.go` — home page handler tests.
- `internal/auth/middleware_test.go` — updated with cookie test cases.

**Login page handler tests (`internal/handler/login_page_test.go`):**

| Test | Description |
|------|-------------|
| `TestLoginPage_Renders` | GET /login, verify 200, verify response contains form elements (username input, password input, submit button). |
| `TestLoginPage_WithError` | GET /login with error parameter, verify 200, verify response contains error message. |

**Browser login handler tests (`internal/handler/login_browser_test.go`):**

| Test | Description |
|------|-------------|
| `TestBrowserLogin_Success` | Seed user, POST valid form data, verify 302 redirect to `/`, verify `auth_token` cookie is set with `HttpOnly=true`. |
| `TestBrowserLogin_WrongPassword` | Seed user, POST wrong password, verify 401, verify response contains error message, verify no cookie is set. |
| `TestBrowserLogin_UserNotFound` | POST non-existent username, verify 401, verify error message, verify no cookie. |
| `TestBrowserLogin_EmptyFields` | POST with empty username/password, verify 401, verify error message. |
| `TestBrowserLogin_RedirectToHome` | After successful login, follow redirect to `/`, verify 200 (requires home page handler). |

**Logout handler tests (`internal/handler/logout_test.go`):**

| Test | Description |
|------|-------------|
| `TestLogout_ClearsCookie` | GET /logout, verify 302 redirect to `/login`, verify `auth_token` cookie is cleared (MaxAge < 0 or empty value). |
| `TestLogout_WithoutCookie` | GET /logout without any cookie, verify 302 redirect to `/login` (no error). |

**Home page handler tests (`internal/handler/home_test.go`):**

| Test | Description |
|------|-------------|
| `TestHomePage_Renders` | Authenticated request (valid cookie), verify 200, verify response contains header, restaurant dropdown, footer. |
| `TestHomePage_ShowsRestaurants` | Seed user with 2 restaurants, authenticated request, verify dropdown contains both restaurant names. |
| `TestHomePage_NoRestaurants` | Seed user with no restaurants, authenticated request, verify dropdown shows "No restaurants available". |
| `TestHomePage_Unauthenticated` | Request without valid cookie, verify redirect to `/login` (handled by browser auth middleware). |

**Auth middleware tests (updates to `internal/auth/middleware_test.go`):**

| Test | Description |
|------|-------------|
| `TestMiddleware_ValidCookie` | Request with valid JWT in `auth_token` cookie (no Authorization header), verify 200, verify claims in context. |
| `TestMiddleware_ExpiredCookie` | Request with expired JWT in cookie, verify 401. |
| `TestMiddleware_InvalidCookie` | Request with garbage in cookie, verify 401. |
| `TestMiddleware_AuthHeaderTakesPrecedence` | Request with both valid Authorization header and valid cookie, verify Authorization header claims are used. |

**Browser auth middleware tests:**

| Test | Description |
|------|-------------|
| `TestBrowserAuth_ValidCookie` | Request with valid cookie, verify next handler is called, verify claims in context. |
| `TestBrowserAuth_NoCookie` | Request without cookie, verify 302 redirect to `/login`. |
| `TestBrowserAuth_ExpiredCookie` | Request with expired cookie, verify 302 redirect to `/login`. |

**TDD approach:** Red-green TDD. Write failing test first, then implement to make it pass.

### 2.15 Edge Cases

| Case | Expected Behavior |
|------|-------------------|
| User visits `/` without authentication | Redirect to `/login` (browser auth middleware) |
| User visits `/login` when already authenticated | Redirect to `/` (login page handler checks for valid cookie) |
| User has no restaurants | Dropdown is disabled, shows "No restaurants available" |
| User has exactly one restaurant | Dropdown shows the single restaurant, pre-selected |
| Cookie expires during a session | Next request triggers redirect to `/login` |
| User clears browser cookies | Next request triggers redirect to `/login` |
| User logs out in one tab | Other tabs will redirect to `/login` on next request |
| Login with valid credentials but user has no restaurant access | Login succeeds, redirect to `/`, dropdown shows "No restaurants available" |
| Static file does not exist | 404 Not Found |
| `auth_token` cookie contains garbage | Middleware treats as invalid, redirects to `/login` |
| Both Authorization header and cookie present | Authorization header takes precedence |
| Login form submitted with GET instead of POST | 405 Method Not Allowed (chi router handles this) |
| Datastar CDN unavailable | Page loads without reactivity; restaurant dropdown still renders (server-rendered) but `data-bind` signal does not function |

### 2.16 Out of Scope

The following are explicitly NOT part of this milestone:
- Recipe CRUD endpoints (Milestone 005).
- Search endpoint (Milestone 005).
- Recipe view/create/edit/delete pages (Milestone 005).
- Changes to `internal/store/` — the data access layer is complete.
- Changes to `internal/db/` — the connection and migration layers are complete.
- Changes to `migrations/` — the schema is complete.
- Changes to `cmd/seed/` — the seed script is complete.
- Registration UI or password reset.
- Token refresh mechanism.
- Rate limiting on login endpoint.
- CSRF token protection (httpOnly cookie with SameSite=Lax provides basic protection).
- Responsive design or mobile-specific layouts.
- Dark mode or theme switching.
- Internationalization (i18n).
