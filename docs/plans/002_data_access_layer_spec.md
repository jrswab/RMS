# Spec 002: Data Access Layer

## Section 1: Context & Constraints

### Milestone Entry

> 002: Data access layer — Go models and repository methods for all tables with TDD coverage (users, restaurants, recipes, ingredients, food, user_restaurants)

### Research Findings — Relevant Context

**Codebase structure (post-milestone 001):**

```
lab37/
├── AGENTS.md
├── README.md
├── Makefile
├── go.mod                 # module lab37, Go 1.22
├── go.sum
├── cmd/
│   └── seed/
│       ├── main.go        # Seed data (raw SQL, no repository layer)
│       └── seed_test.go
├── docs/plans/
│   ├── RMS000_design.md
│   ├── RMS_recipe_manager_milestones.md
│   ├── 001_database_schema_spec.md
│   └── 001_database_schema_implement.md
├── internal/
│   └── db/
│       ├── db.go          # Open(dbPath) → *sql.DB, DefaultPath()
│       ├── db_test.go
│       ├── migrate.go     # RunMigrations(db, fs.FS)
│       └── migrate_test.go
├── migrations/
│   ├── embed.go           # //go:embed *.up.sql → var FS embed.FS
│   └── 001_create_tables.up.sql
└── static/
```

**Database schema (6 tables, from migration `001_create_tables.up.sql`):**

| Table | Columns | Constraints |
|-------|---------|-------------|
| `restaurants` | `id TEXT PK`, `name TEXT NOT NULL` | — |
| `users` | `id TEXT PK`, `username TEXT NOT NULL UNIQUE`, `password TEXT NOT NULL`, `role TEXT NOT NULL`, `created_at INTEGER NOT NULL`, `updated_at INTEGER NOT NULL` | Unique on `username` |
| `food` | `id TEXT PK`, `name TEXT NOT NULL` | — |
| `user_restaurants` | `user_id TEXT NOT NULL`, `restaurant_id TEXT NOT NULL` | `UNIQUE(user_id, restaurant_id)`, FKs to `users` and `restaurants` with `ON DELETE CASCADE` |
| `recipes` | `id TEXT PK`, `name TEXT NOT NULL`, `restaurant_id TEXT NOT NULL`, `instructions TEXT NOT NULL`, `yield INTEGER NOT NULL`, `created_at INTEGER NOT NULL`, `updated_at INTEGER NOT NULL` | FK to `restaurants` with `ON DELETE CASCADE` |
| `ingredients` | `id TEXT PK`, `recipe_id TEXT NOT NULL`, `food_id TEXT NOT NULL`, `quantity REAL NOT NULL`, `unit TEXT NOT NULL`, `sort_order INTEGER NOT NULL` | `UNIQUE(recipe_id, sort_order)`, FK to `recipes` with `ON DELETE CASCADE`, FK to `food` with `ON DELETE RESTRICT` |

**Indexes:** `idx_recipes_restaurant_id`, `idx_recipes_name`, `idx_ingredients_recipe_id`.

**Connection layer (`internal/db/db.go`):**
- `Open(dbPath string) (*sql.DB, error)` — opens SQLite, applies `PRAGMA foreign_keys = ON` and `PRAGMA journal_mode = WAL`, pings to verify.
- `DefaultPath() string` — reads `DB_PATH` env var, returns `"recipes.db"` if unset.

**Dependencies (go.mod):**
- `github.com/golang-jwt/jwt/v5 v5.2.1`
- `github.com/google/uuid v1.6.0`
- `github.com/mattn/go-sqlite3 v1.14.22`
- `golang.org/x/crypto v0.33.0`

No ORM, no web framework, no router.

**Decisions already made (do not re-evaluate):**
1. Tech stack: Go + Templ + Datastar + SQLite.
2. No frontend build step. Server-rendered HTML via Templ.
3. SQLite for now, PostgreSQL later.
4. JWT in Authorization header for all routes except `/login`.
5. Roles are plain text: `admin`, `manager`, `staff`.
6. UUIDs for all primary keys (not auto-increment).
7. Configuration via environment variables.
8. Migrations embedded in binary, run on startup.
9. Password storage: bcrypt hash + salt.
10. Direct SQL with repository pattern — no ORM.

**Approaches already ruled out:**
- Separate frontend SPA — server-rendered HTML chosen.
- Session-based auth — JWT specified.
- ORM (GORM, etc.) — direct SQL chosen.
- Auto-increment IDs — UUIDs specified.
- Separate auth service — monolith chosen.

**Constraints:**
- Go 1.22.
- SQLite as database (`recipes.db` at project root).
- Zero frontend build step.
- Datastar is new — Context7 MCP must be consulted before generating Datastar code (per AGENTS.md). This milestone does not involve Datastar.

**Resolved open questions:**
- Configuration: environment variables (JWT secret, DB path, server port).
- Migrations: embedded in binary, run on startup.
- Password storage: bcrypt hash + salt.
- Auth complexity: keep simple for demo. No registration UI, no password reset.

### User-Confirmed Decisions (for this spec)

| Decision | Choice |
|----------|--------|
| Package structure | Single package `internal/store/` — models and repository methods co-located |
| Transaction support | Repository methods accept an interface (`DBTX`) that both `*sql.DB` and `*sql.Tx` satisfy. Caller manages transactions. |
| CRUD scope | Minimal — only methods needed by the API endpoints (see Section 2.3) |
| Error handling | Sentinel errors: `var ErrNotFound = errors.New("not found")`. Callers use `errors.Is()`. |
| Testing strategy | Real in-memory SQLite (`:memory:`). Open via `db.Open(":memory:")`, run migrations, test against real SQL. |
| TDD | Red-green TDD: write failing test first, then implement to make it pass. |

---

## Section 2: Requirements

### 2.1 Package Location

All code for this milestone lives in `internal/store/`.

No new packages are created. No changes to existing packages (`internal/db/`, `migrations/`, `cmd/seed/`).

### 2.2 Model Definitions

Define Go structs in `internal/store/models.go` that map 1:1 to the database tables. Each struct field maps to a column of the same name.

**Required models:**

```
User
  ID        string
  Username  string
  Password  string
  Role      string
  CreatedAt int64
  UpdatedAt int64

Restaurant
  ID   string
  Name string

Food
  ID   string
  Name string

UserRestaurant
  UserID       string
  RestaurantID string

Recipe
  ID           string
  Name         string
  RestaurantID string
  Instructions string
  Yield        int64
  CreatedAt    int64
  UpdatedAt    int64

Ingredient
  ID        string
  RecipeID  string
  FoodID    string
  Quantity  float64
  Unit      string
  SortOrder int64
```

All ID fields are UUIDs stored as strings. Timestamps are Unix epoch seconds.

### 2.3 Repository Methods

Define a `DBTX` interface in `internal/store/store.go`:

```
type DBTX interface {
  Exec(query string, args ...any) (sql.Result, error)
  Query(query string, args ...any) (*sql.Rows, error)
  QueryRow(query string, args ...any) *sql.Row
}
```

Both `*sql.DB` and `*sql.Tx` satisfy this interface. All repository functions accept `DBTX` as their first parameter. This allows callers to pass either a direct DB handle or a transaction.

**User methods (`internal/store/user.go`):**

| Function | Signature | Behavior |
|----------|-----------|----------|
| `GetUserByUsername` | `(db DBTX, username string) (*User, error)` | Query `users` by `username`. Return `ErrNotFound` if no row. |
| `GetUserByID` | `(db DBTX, id string) (*User, error)` | Query `users` by `id`. Return `ErrNotFound` if no row. |

**Restaurant methods (`internal/store/restaurant.go`):**

| Function | Signature | Behavior |
|----------|-----------|----------|
| `GetRestaurantByID` | `(db DBTX, id string) (*Restaurant, error)` | Query `restaurants` by `id`. Return `ErrNotFound` if no row. |
| `ListRestaurantsByUserID` | `(db DBTX, userID string) ([]Restaurant, error)` | Join `user_restaurants` with `restaurants` on `user_id`. Return empty slice (not nil) if no rows. |

**Food methods (`internal/store/food.go`):**

| Function | Signature | Behavior |
|----------|-----------|----------|
| `GetFoodByID` | `(db DBTX, id string) (*Food, error)` | Query `food` by `id`. Return `ErrNotFound` if no row. |
| `ListFood` | `(db DBTX) ([]Food, error)` | Query all rows from `food` ordered by `name`. Return empty slice if no rows. |

**Recipe methods (`internal/store/recipe.go`):**

| Function | Signature | Behavior |
|----------|-----------|----------|
| `GetRecipeByID` | `(db DBTX, id string) (*Recipe, error)` | Query `recipes` by `id`. Return `ErrNotFound` if no row. |
| `SearchRecipes` | `(db DBTX, restaurantID string, query string) ([]Recipe, error)` | Query `recipes` where `restaurant_id` matches and `name` matches with SQL `LIKE` using `%query%` wildcards. Query text must be escaped (replace `%` → `\%`, `_` → `\_`, `\` → `\\`; use `ESCAPE '\'` in the LIKE clause). Return empty slice if no rows. |
| `CreateRecipe` | `(db DBTX, r *Recipe) error` | Insert into `recipes`. The caller is responsible for setting `ID`, `CreatedAt`, `UpdatedAt`. |
| `UpdateRecipe` | `(db DBTX, r *Recipe) error` | Update `recipes` row matching `r.ID`. Updates `name`, `instructions`, `yield`, `updated_at`. Does NOT update `id` or `restaurant_id`. Return `ErrNotFound` if no row affected. |
| `DeleteRecipe` | `(db DBTX, id string) error` | Delete from `recipes` where `id` matches. Return `ErrNotFound` if no row affected. Ingredients are cascade-deleted by the database. |

**Ingredient methods (`internal/store/ingredient.go`):**

| Function | Signature | Behavior |
|----------|-----------|----------|
| `ListIngredientsByRecipeID` | `(db DBTX, recipeID string) ([]Ingredient, error)` | Query `ingredients` where `recipe_id` matches, ordered by `sort_order`. Return empty slice if no rows. |
| `ReplaceIngredients` | `(db DBTX, recipeID string, ingredients []Ingredient) error` | Delete all `ingredients` for `recipeID`, then insert all provided ingredients. Must run within a transaction (caller's responsibility). The caller is responsible for setting `ID`, `RecipeID`, `SortOrder` on each ingredient. |

**UserRestaurant methods (`internal/store/user_restaurant.go`):**

| Function | Signature | Behavior |
|----------|-----------|----------|
| `ListUserRestaurantIDs` | `(db DBTX, userID string) ([]string, error)` | Query `user_restaurants` for `restaurant_id` values where `user_id` matches. Return empty slice if no rows. |

### 2.4 Sentinel Errors

Define in `internal/store/store.go`:

```
var ErrNotFound = errors.New("not found")
```

Any repository function that queries for a single row and finds none must return `ErrNotFound` as the error. Callers distinguish "not found" from real errors using `errors.Is(err, store.ErrNotFound)`.

No other sentinel errors are required for this milestone.

### 2.5 Testing Requirements

**Test file locations:**
- `internal/store/user_test.go`
- `internal/store/restaurant_test.go`
- `internal/store/food_test.go`
- `internal/store/recipe_test.go`
- `internal/store/ingredient_test.go`
- `internal/store/user_restaurant_test.go`

**Test helper (`internal/store/helpers_test.go`):**
- Function `setupTestDB(t *testing.T) *sql.DB` — opens `:memory:` DB via `db.Open(":memory:")`, runs migrations via `db.RunMigrations(db, migrations.FS)`, returns the `*sql.DB`. Calls `t.Fatal` on any error.
- Function `seedTestData(t *testing.T, database *sql.DB)` — inserts minimal test fixtures (1 user, 1 restaurant, 1 user_restaurant link, 1 food item, 1 recipe, 1 ingredient) using deterministic UUIDs. Used by tests that need pre-existing data.

**TDD approach:** Each test file follows red-green TDD. Write the test first (expecting compilation failure or test failure), then write the implementation to make it pass.

**Required test cases per entity:**

**User tests:**
- `TestGetUserByUsername` — seed a user, retrieve by username, verify all fields match.
- `TestGetUserByUsername_NotFound` — query non-existent username, verify `ErrNotFound`.
- `TestGetUserByID` — seed a user, retrieve by ID, verify all fields match.
- `TestGetUserByID_NotFound` — query non-existent ID, verify `ErrNotFound`.

**Restaurant tests:**
- `TestGetRestaurantByID` — seed a restaurant, retrieve by ID, verify fields.
- `TestGetRestaurantByID_NotFound` — query non-existent ID, verify `ErrNotFound`.
- `TestListRestaurantsByUserID` — seed user with 2 restaurant links, list by user ID, verify 2 results.
- `TestListRestaurantsByUserID_NoResults` — query user with no restaurant links, verify empty slice (not nil, not error).

**Food tests:**
- `TestGetFoodByID` — seed a food item, retrieve by ID, verify fields.
- `TestGetFoodByID_NotFound` — query non-existent ID, verify `ErrNotFound`.
- `TestListFood` — seed 3 food items, list all, verify 3 results ordered by name.
- `TestListFood_Empty` — no food items seeded, verify empty slice.

**Recipe tests:**
- `TestGetRecipeByID` — seed a recipe, retrieve by ID, verify all fields.
- `TestGetRecipeByID_NotFound` — query non-existent ID, verify `ErrNotFound`.
- `TestSearchRecipes` — seed 3 recipes (2 matching search term, 1 not), search, verify 2 results.
- `TestSearchRecipes_EscapeSpecialChars` — seed a recipe with `%` and `_` in the name, search for literal `%` and `_`, verify correct results (not wildcard matches).
- `TestSearchRecipes_NoResults` — search with term matching nothing, verify empty slice.
- `TestCreateRecipe` — create a recipe, retrieve by ID, verify all fields match.
- `TestUpdateRecipe` — seed a recipe, update name/instructions/yield, retrieve and verify changes.
- `TestUpdateRecipe_NotFound` — update non-existent ID, verify `ErrNotFound`.
- `TestDeleteRecipe` — seed a recipe with ingredients, delete, verify recipe and ingredients are gone.
- `TestDeleteRecipe_NotFound` — delete non-existent ID, verify `ErrNotFound`.

**Ingredient tests:**
- `TestListIngredientsByRecipeID` — seed recipe with 3 ingredients, list by recipe ID, verify 3 results ordered by `sort_order`.
- `TestListIngredientsByRecipeID_NoResults` — recipe with no ingredients, verify empty slice.
- `TestReplaceIngredients` — seed recipe with 2 ingredients, replace with 3 new ingredients, verify old are gone and 3 new exist.
- `TestReplaceIngredients_Empty` — replace with empty slice, verify all ingredients deleted.

**UserRestaurant tests:**
- `TestListUserRestaurantIDs` — seed user with 2 restaurant links, list, verify 2 IDs returned.
- `TestListUserRestaurantIDs_NoResults` — user with no links, verify empty slice.

### 2.6 Edge Cases

| Case | Expected Behavior |
|------|-------------------|
| Query for non-existent ID | Return `nil, ErrNotFound` |
| Query for non-existent username | Return `nil, ErrNotFound` |
| List query with no matching rows | Return empty slice `[]T{}`, nil error |
| `SearchRecipes` with empty query string | Must still work — matches all recipes for the restaurant (LIKE `%%`) |
| `SearchRecipes` with SQL wildcard chars in query | Must escape `%`, `_`, `\` before building LIKE pattern |
| `ReplaceIngredients` called with empty slice | Deletes all existing ingredients for the recipe, inserts nothing |
| `DeleteRecipe` on recipe with ingredients | Ingredients are cascade-deleted by the database FK constraint |
| `DeleteRecipe` on recipe with no ingredients | Deletes the recipe row only, no error |
| `CreateRecipe` with duplicate ID | Returns SQLite constraint error (caller's responsibility to generate unique IDs) |
| `UpdateRecipe` where `restaurant_id` is changed | Not allowed — `UpdateRecipe` must not update `restaurant_id` |
| Concurrent access | SQLite WAL mode (set by connection layer) allows concurrent reads. Writes are serialized by SQLite. No application-level locking needed. |

### 2.7 Out of Scope

The following are explicitly NOT part of this milestone:
- HTTP server or handlers (Milestone 003).
- Templ templates or frontend (Milestone 004).
- JWT or authentication logic (Milestone 003).
- Recipe CRUD endpoints (Milestone 005).
- Changes to `cmd/seed/` — the seed script continues to use raw SQL.
- Changes to `internal/db/` — the connection and migration layers are complete.
- Changes to `migrations/` — the schema is complete.
- Any Datastar integration.
