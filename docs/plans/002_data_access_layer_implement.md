# Implementation Guide 002: Data Access Layer

**Spec:** `docs/plans/002_data_access_layer_spec.md`

---

## Section 1: Context Summary

**Milestone:** 002 — Data access layer — Go models and repository methods for all tables with TDD coverage.

The database schema exists (milestone 001) but the application has no typed Go code to interact with it. The seed script uses raw SQL, and there are no models, no repository methods, and no query layer. Before HTTP handlers (milestone 003), authentication (milestone 003), or templates (milestone 004) can be built, the system needs a typed data access layer that maps the 6 SQLite tables to Go structs and provides tested repository functions. This milestone creates the `internal/store/` package with models, a `DBTX` interface for transaction support, sentinel errors, and 15 repository methods covering all entities — each backed by real in-memory SQLite tests written via red-green TDD.

---

## Section 2: Implementation Checklist

### Task 1: Create DBTX interface and sentinel error

- [x] Create `internal/store/store.go`
  - Define `DBTX` interface with three methods: `Exec(query string, args ...any) (sql.Result, error)`, `Query(query string, args ...any) (*sql.Rows, error)`, `QueryRow(query string, args ...any) *sql.Row`
  - Define `var ErrNotFound = errors.New("not found")`
  - Package declaration: `package store`
  - Imports: `database/sql`, `errors`

### Task 2: Create model structs

- [x] Create `internal/store/models.go`
  - Define `User` struct: `ID string`, `Username string`, `Password string`, `Role string`, `CreatedAt int64`, `UpdatedAt int64`
  - Define `Restaurant` struct: `ID string`, `Name string`
  - Define `Food` struct: `ID string`, `Name string`
  - Define `UserRestaurant` struct: `UserID string`, `RestaurantID string`
  - Define `Recipe` struct: `ID string`, `Name string`, `RestaurantID string`, `Instructions string`, `Yield int64`, `CreatedAt int64`, `UpdatedAt int64`
  - Define `Ingredient` struct: `ID string`, `RecipeID string`, `FoodID string`, `Quantity float64`, `Unit string`, `SortOrder int64`
  - Package declaration: `package store`

### Task 3: Create test helpers

- [x] Create `internal/store/helpers_test.go`
  - Package declaration: `package store`
  - Imports: `database/sql`, `testing`, `lab37/internal/db`, `lab37/migrations`
  - Function `setupTestDB(t *testing.T) *sql.DB` — calls `t.Helper()`, opens `:memory:` DB via `db.Open(":memory:")`, runs migrations via `db.RunMigrations(database, migrations.FS)`, calls `t.Fatal` on any error, returns the `*sql.DB`
  - Function `seedTestData(t *testing.T, database *sql.DB)` — calls `t.Helper()`, inserts via raw SQL: 1 user (deterministic UUID, username `"testuser"`, bcrypt hash of `"testpass"`, role `"staff"`, fixed timestamps), 1 restaurant (deterministic UUID, name `"Test Restaurant"`), 1 user_restaurant link, 1 food item (deterministic UUID, name `"Flour"`), 1 recipe (deterministic UUID, name `"Test Recipe"`, linked to the restaurant, fixed timestamps), 1 ingredient (deterministic UUID, linked to recipe and food, quantity 1.0, unit `"cup"`, sort_order 1). Uses `INSERT OR IGNORE` for idempotency. Uses `github.com/google/uuid` for deterministic UUIDs via `uuid.NewSHA1(uuid.NameSpaceURL, []byte(name))`.

### Task 4: Implement and test User repository

- [x] Create `internal/store/user_test.go`
  - Package declaration: `package store`
  - Imports: `errors`, `testing`
  - `TestGetUserByUsername` — call `setupTestDB`, call `seedTestData`, call `GetUserByUsername(db, "testuser")`, assert no error, assert all fields match seeded values
  - `TestGetUserByUsername_NotFound` — call `setupTestDB`, call `GetUserByUsername(db, "nonexistent")`, assert `errors.Is(err, ErrNotFound)`, assert result is nil
  - `TestGetUserByID` — call `setupTestDB`, call `seedTestData`, call `GetUserByID(db, <seeded user ID>)`, assert no error, assert all fields match
  - `TestGetUserByID_NotFound` — call `setupTestDB`, call `GetUserByID(db, "nonexistent")`, assert `errors.Is(err, ErrNotFound)`, assert result is nil

- [x] Create `internal/store/user.go`
  - Package declaration: `package store`
  - Imports: `database/sql`, `errors`, `fmt`
  - Function `GetUserByUsername(db DBTX, username string) (*User, error)` — `SELECT id, username, password, role, created_at, updated_at FROM users WHERE username = ?`, scan into `User`, if `sql.ErrNoRows` return `nil, ErrNotFound`, wrap other errors with `fmt.Errorf`
  - Function `GetUserByID(db DBTX, id string) (*User, error)` — `SELECT id, username, password, role, created_at, updated_at FROM users WHERE id = ?`, scan into `User`, if `sql.ErrNoRows` return `nil, ErrNotFound`, wrap other errors with `fmt.Errorf`

### Task 5: Implement and test Restaurant repository

- [x] Create `internal/store/restaurant_test.go`
  - Package declaration: `package store`
  - Imports: `errors`, `testing`
  - `TestGetRestaurantByID` — call `setupTestDB`, call `seedTestData`, call `GetRestaurantByID(db, <seeded restaurant ID>)`, assert no error, assert fields match
  - `TestGetRestaurantByID_NotFound` — call `setupTestDB`, call `GetRestaurantByID(db, "nonexistent")`, assert `errors.Is(err, ErrNotFound)`, assert result is nil
  - `TestListRestaurantsByUserID` — call `setupTestDB`, insert 2 restaurants and 2 user_restaurant links for one user via raw SQL, call `ListRestaurantsByUserID(db, <user ID>)`, assert no error, assert length is 2
  - `TestListRestaurantsByUserID_NoResults` — call `setupTestDB`, call `ListRestaurantsByUserID(db, "nonexistent")`, assert no error, assert length is 0, assert slice is not nil

- [x] Create `internal/store/restaurant.go`
  - Package declaration: `package store`
  - Imports: `database/sql`, `errors`, `fmt`
  - Function `GetRestaurantByID(db DBTX, id string) (*Restaurant, error)` — `SELECT id, name FROM restaurants WHERE id = ?`, scan into `Restaurant`, if `sql.ErrNoRows` return `nil, ErrNotFound`, wrap other errors
  - Function `ListRestaurantsByUserID(db DBTX, userID string) ([]Restaurant, error)` — `SELECT r.id, r.name FROM restaurants r INNER JOIN user_restaurants ur ON r.id = ur.restaurant_id WHERE ur.user_id = ?`, iterate rows, append to slice, return empty slice (not nil) if no rows

### Task 6: Implement and test Food repository

- [x] Create `internal/store/food_test.go`
  - Package declaration: `package store`
  - Imports: `errors`, `testing`
  - `TestGetFoodByID` — call `setupTestDB`, call `seedTestData`, call `GetFoodByID(db, <seeded food ID>)`, assert no error, assert fields match
  - `TestGetFoodByID_NotFound` — call `setupTestDB`, call `GetFoodByID(db, "nonexistent")`, assert `errors.Is(err, ErrNotFound)`, assert result is nil
  - `TestListFood` — call `setupTestDB`, insert 3 food items with names "Zucchini", "Apple", "Milk" via raw SQL, call `ListFood(db)`, assert no error, assert length is 3, assert order is "Apple", "Milk", "Zucchini" (alphabetical)
  - `TestListFood_Empty` — call `setupTestDB`, call `ListFood(db)`, assert no error, assert length is 0, assert slice is not nil

- [x] Create `internal/store/food.go`
  - Package declaration: `package store`
  - Imports: `database/sql`, `errors`, `fmt`
  - Function `GetFoodByID(db DBTX, id string) (*Food, error)` — `SELECT id, name FROM food WHERE id = ?`, scan into `Food`, if `sql.ErrNoRows` return `nil, ErrNotFound`, wrap other errors
  - Function `ListFood(db DBTX) ([]Food, error)` — `SELECT id, name FROM food ORDER BY name`, iterate rows, append to slice, return empty slice if no rows

### Task 7: Implement and test Recipe repository

- [x] Create `internal/store/recipe_test.go`
  - Package declaration: `package store`
  - Imports: `errors`, `testing`
  - `TestGetRecipeByID` — call `setupTestDB`, call `seedTestData`, call `GetRecipeByID(db, <seeded recipe ID>)`, assert no error, assert all fields match
  - `TestGetRecipeByID_NotFound` — call `setupTestDB`, call `GetRecipeByID(db, "nonexistent")`, assert `errors.Is(err, ErrNotFound)`, assert result is nil
  - `TestSearchRecipes` — call `setupTestDB`, insert 3 recipes for same restaurant ("Butter Chicken", "Butter Cookies", "Fried Rice") via raw SQL, call `SearchRecipes(db, <restaurant ID>, "Butter")`, assert no error, assert length is 2
  - `TestSearchRecipes_EscapeSpecialChars` — call `setupTestDB`, insert recipe named "100% That Recipe" and recipe named "A_B_C" via raw SQL, call `SearchRecipes(db, <restaurant ID>, "100%")`, assert returns only "100% That Recipe"; call `SearchRecipes(db, <restaurant ID>, "A_B")`, assert returns only "A_B_C"
  - `TestSearchRecipes_NoResults` — call `setupTestDB`, call `seedTestData`, call `SearchRecipes(db, <restaurant ID>, "NonexistentTerm")`, assert no error, assert length is 0
  - `TestCreateRecipe` — call `setupTestDB`, call `seedTestData`, build a `Recipe` struct with a new UUID, call `CreateRecipe(db, &recipe)`, assert no error, call `GetRecipeByID` to verify all fields match
  - `TestUpdateRecipe` — call `setupTestDB`, call `seedTestData`, build a `Recipe` struct with the seeded ID but updated name/instructions/yield/UpdatedAt, call `UpdateRecipe(db, &recipe)`, assert no error, call `GetRecipeByID` to verify updated fields
  - `TestUpdateRecipe_NotFound` — call `setupTestDB`, build a `Recipe` struct with a non-existent ID, call `UpdateRecipe(db, &recipe)`, assert `errors.Is(err, ErrNotFound)`
  - `TestDeleteRecipe` — call `setupTestDB`, call `seedTestData`, call `DeleteRecipe(db, <seeded recipe ID>)`, assert no error, call `GetRecipeByID` to verify `ErrNotFound`, call `ListIngredientsByRecipeID` to verify ingredients are also gone (empty slice)
  - `TestDeleteRecipe_NotFound` — call `setupTestDB`, call `DeleteRecipe(db, "nonexistent")`, assert `errors.Is(err, ErrNotFound)`

- [x] Create `internal/store/recipe.go`
  - Package declaration: `package store`
  - Imports: `database/sql`, `errors`, `fmt`, `strings`
  - Function `GetRecipeByID(db DBTX, id string) (*Recipe, error)` — `SELECT id, name, restaurant_id, instructions, yield, created_at, updated_at FROM recipes WHERE id = ?`, scan into `Recipe`, if `sql.ErrNoRows` return `nil, ErrNotFound`, wrap other errors
  - Function `SearchRecipes(db DBTX, restaurantID string, query string) ([]Recipe, error)` — escape `query` by replacing `\` → `\\`, `%` → `\%`, `_` → `\_` (order matters: backslash first), build `SELECT id, name, restaurant_id, instructions, yield, created_at, updated_at FROM recipes WHERE restaurant_id = ? AND name LIKE ? ESCAPE '\'`, pass `%` + escaped query + `%` as the LIKE parameter, iterate rows, return empty slice if no rows
  - Function `CreateRecipe(db DBTX, r *Recipe) error` — `INSERT INTO recipes (id, name, restaurant_id, instructions, yield, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, pass `r.ID, r.Name, r.RestaurantID, r.Instructions, r.Yield, r.CreatedAt, r.UpdatedAt`, wrap errors
  - Function `UpdateRecipe(db DBTX, r *Recipe) error` — `UPDATE recipes SET name = ?, instructions = ?, yield = ?, updated_at = ? WHERE id = ?`, pass `r.Name, r.Instructions, r.Yield, r.UpdatedAt, r.ID`, check `RowsAffected()`, if 0 return `ErrNotFound`, wrap other errors
  - Function `DeleteRecipe(db DBTX, id string) error` — `DELETE FROM recipes WHERE id = ?`, check `RowsAffected()`, if 0 return `ErrNotFound`, wrap other errors

### Task 8: Implement and test Ingredient repository

- [x] Create `internal/store/ingredient_test.go`
  - Package declaration: `package store`
  - Imports: `testing`
  - `TestListIngredientsByRecipeID` — call `setupTestDB`, call `seedTestData`, insert 2 more ingredients for the same recipe (sort_order 2 and 3) via raw SQL, call `ListIngredientsByRecipeID(db, <seeded recipe ID>)`, assert no error, assert length is 3, assert order matches sort_order 1, 2, 3
  - `TestListIngredientsByRecipeID_NoResults` — call `setupTestDB`, call `seedTestData`, create a second recipe with no ingredients via raw SQL, call `ListIngredientsByRecipeID(db, <second recipe ID>)`, assert no error, assert length is 0, assert slice is not nil
  - `TestReplaceIngredients` — call `setupTestDB`, call `seedTestData`, call `ReplaceIngredients(db, <seeded recipe ID>, []Ingredient{...})` with 3 new ingredients (new UUIDs, new food IDs, sort_order 1–3), assert no error, call `ListIngredientsByRecipeID` to verify exactly 3 ingredients exist with the new values
  - `TestReplaceIngredients_Empty` — call `setupTestDB`, call `seedTestData`, call `ReplaceIngredients(db, <seeded recipe ID>, []Ingredient{})`, assert no error, call `ListIngredientsByRecipeID` to verify 0 ingredients

- [x] Create `internal/store/ingredient.go`
  - Package declaration: `package store`
  - Imports: `fmt`
  - Function `ListIngredientsByRecipeID(db DBTX, recipeID string) ([]Ingredient, error)` — `SELECT id, recipe_id, food_id, quantity, unit, sort_order FROM ingredients WHERE recipe_id = ? ORDER BY sort_order`, iterate rows, append to slice, return empty slice if no rows
  - Function `ReplaceIngredients(db DBTX, recipeID string, ingredients []Ingredient) error` — `DELETE FROM ingredients WHERE recipe_id = ?`, then if `len(ingredients) > 0` loop and `INSERT INTO ingredients (id, recipe_id, food_id, quantity, unit, sort_order) VALUES (?, ?, ?, ?, ?, ?)` for each ingredient, wrap errors with `fmt.Errorf`

### Task 9: Implement and test UserRestaurant repository

- [x] Create `internal/store/user_restaurant_test.go`
  - Package declaration: `package store`
  - Imports: `testing`
  - `TestListUserRestaurantIDs` — call `setupTestDB`, insert 1 user and 2 restaurants and 2 user_restaurant links via raw SQL, call `ListUserRestaurantIDs(db, <user ID>)`, assert no error, assert length is 2, assert both restaurant IDs are present
  - `TestListUserRestaurantIDs_NoResults` — call `setupTestDB`, call `ListUserRestaurantIDs(db, "nonexistent")`, assert no error, assert length is 0, assert slice is not nil

- [x] Create `internal/store/user_restaurant.go`
  - Package declaration: `package store`
  - Imports: `fmt`
  - Function `ListUserRestaurantIDs(db DBTX, userID string) ([]string, error)` — `SELECT restaurant_id FROM user_restaurants WHERE user_id = ?`, iterate rows, scan each `restaurant_id` into a string, append to slice, return empty slice if no rows

### Task 10: Run full test suite

- [x] Run `go test ./...` from the project root and verify all tests pass with no failures
