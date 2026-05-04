# Implementation Guide 005: Recipe CRUD and Search

**Spec:** `docs/plans/005_recipe_crud_search_spec.md`

---

## Section 1: Context Summary

Milestone 005 — Recipe CRUD and search. The database schema, data access layer, authentication system, and frontend foundation are complete (milestones 001–004). The application has a working HTTP server with JWT-based login, cookie-based browser auth, server-rendered Templ pages, and a restaurant dropdown that sets a Datastar signal. However, users cannot yet search, view, create, edit, or delete recipes. This milestone adds the Datastar Go SDK for server-side SSE responses, two new store methods for food lookup by name and food creation, a search handler that returns HTML fragments via Datastar SSE, a full-page recipe view, create and edit forms with inline ingredient management (free-text food names with server-side food creation), a Datastar-driven delete confirmation flow, and route registration. All new code is tested against real in-memory SQLite databases and real HTTP requests.

---

## Section 2: Implementation Checklist

### Task 1: Add Datastar Go SDK dependency

- [x] Run `go get github.com/starfederation/datastar-go` from the project root so the handler files can import `github.com/starfederation/datastar-go/datastar`.
- [x] Run `go mod tidy` to update `go.mod` and `go.sum`.
- [x] Run `go build ./...` immediately after the dependency change to verify the new package resolves cleanly before any milestone 005 code is added.

### Task 2: Add store methods for food lookup by name and food creation (TDD)

- [x] Edit `internal/store/food_test.go` and add `TestGetFoodByName_Found`:
  - Use `setupTestDB(t)` and `seedTestData(t, db)`.
  - Call `GetFoodByName(db, testFoodName)`.
  - Assert the returned `*Food` is non-nil and matches the seeded `ID` and `Name`.
- [x] Edit `internal/store/food_test.go` and add `TestGetFoodByName_NotFound`:
  - Use `setupTestDB(t)` with no matching `food` row.
  - Call `GetFoodByName(db, "does-not-exist")`.
  - Assert `errors.Is(err, ErrNotFound)` and the returned food is `nil`.
- [x] Edit `internal/store/food_test.go` and add `TestCreateFood`:
  - Create a `Food{ID: uuid.New().String(), Name: "Butter"}`.
  - Call `CreateFood(db, &food)`.
  - Verify persistence with `GetFoodByID(db, food.ID)`.
- [x] Edit `internal/store/food_test.go` and add `TestCreateFood_DuplicateName`:
  - Create two `Food` rows with different `ID` values and the same `Name`.
  - Call `CreateFood` twice.
  - Verify both rows exist with a raw `SELECT COUNT(*) FROM food WHERE name = ?` assertion equal to `2`.
- [x] Run `go test ./internal/store -run 'Test(GetFoodByName|CreateFood)'` and verify the new tests fail before implementation.
- [x] Edit `internal/store/food.go` and add `func GetFoodByName(db DBTX, name string) (*Food, error)`:
  - Use `SELECT id, name FROM food WHERE name = ? LIMIT 1`.
  - Translate `sql.ErrNoRows` to `ErrNotFound`.
  - Wrap all other database errors with context in the same style as the existing store functions.
- [x] Edit `internal/store/food.go` and add `func CreateFood(db DBTX, f *Food) error`:
  - Insert `id` and `name` into `food`.
  - Return wrapped errors on insert failure.
- [x] Run `go test ./internal/store -run 'Test(GetFoodByName|CreateFood)'` again and verify the new tests pass.
- [x] Run `go test ./internal/store` to confirm the added methods do not break the existing store package test suite.

### Task 3: Create search handler (TDD)

- [x] Create `internal/handler/search_test.go` and add `TestSearch_Success`:
  - Reuse `setupTestDB(t)` from `internal/handler/login_test.go`.
  - Seed a user and restaurant with `seedTestUser` or equivalent test SQL.
  - Insert two recipes in the accessible restaurant and at least one non-matching recipe.
  - Build a `POST /search` request whose JSON body matches Datastar signals, for example `{"searchQuery":"Soup","selectedRestaurant":"<restaurant-id>"}`.
  - Attach claims to the request context and serve `SearchHandler.ServeHTTP` directly.
  - Assert `200`, `Content-Type` includes `text/event-stream`, and the SSE body contains the matching recipe link/name while excluding non-matching recipes.
- [x] In `internal/handler/search_test.go`, add `TestSearch_NoResults` and assert the SSE body contains a stable no-results string such as `No recipes found.`.
- [x] In `internal/handler/search_test.go`, add `TestSearch_EmptyQuery` and assert an empty query returns all recipes for the selected restaurant.
- [x] In `internal/handler/search_test.go`, add `TestSearch_NoRestaurant` and assert the SSE body contains a stable prompt such as `Select a restaurant to search recipes.`.
- [x] In `internal/handler/search_test.go`, add `TestSearch_UnauthorizedRestaurant` and assert the handler returns `403` when `selectedRestaurant` is not present in `claims.RestaurantIDs`.
- [x] In `internal/handler/search_test.go`, add `TestSearch_Unauthenticated` and assert the bare handler returns `401` when no claims are present in context.
- [x] Run `go test ./internal/handler -run 'TestSearch'` and verify the new tests fail before implementation.
- [x] Create `internal/handler/search.go` and define `type SearchSignals struct` with:
  - A `SearchQuery string` field tagged as `json:"searchQuery"`
  - A `SelectedRestaurant string` field tagged as `json:"selectedRestaurant"`
- [x] In `internal/handler/search.go`, define `type SearchHandler struct { DB store.DBTX }`.
- [x] Implement `func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)` in `internal/handler/search.go`:
  1. Call `auth.ClaimsFromContext(r)`; if it fails, return `http.StatusUnauthorized` before any SSE work starts.
  2. Call `datastar.ReadSignals(r, &signals)` before creating the SSE writer.
  3. If `signals.SelectedRestaurant == ""`, render the no-restaurant prompt fragment and patch `#search-results` instead of calling the store.
  4. If `signals.SelectedRestaurant != ""` and it is not in `claims.RestaurantIDs`, return `http.StatusForbidden`.
  5. Call `store.SearchRecipes(h.DB, signals.SelectedRestaurant, signals.SearchQuery)`.
  6. Render the fragment with a reusable page component such as `pages.SearchResults(...)` from `views/pages/home.templ` rather than hand-building HTML in the handler.
  7. Create `sse := datastar.NewSSE(w, r)` only after `datastar.ReadSignals` has succeeded.
  8. Send the fragment with `sse.PatchElements(html, datastar.WithSelectorID("search-results"))`.
- [x] Keep all search errors that happen before `datastar.NewSSE` as normal HTTP errors/status codes; only successful interactive responses should open an SSE stream.
- [x] Run `go test ./internal/handler -run 'TestSearch'` again and verify the new tests pass.

### Task 4: Update home page template with search UI

- [x] Edit `views/pages/home.templ` and keep `templ Home(username string, restaurants []store.Restaurant)` as the full-page home component.
- [x] Replace the placeholder welcome-only body in `views/pages/home.templ` with a conditional UI:
  - If `len(restaurants) == 0`, show a no-restaurant state and do not render the search form, results container, or create link.
  - If `len(restaurants) > 0`, render the search UI and create link.
- [x] In the `len(restaurants) > 0` branch of `views/pages/home.templ`, render:
  - `<input data-bind:search-query type="text" placeholder="Search recipes...">`
  - `<button data-on:click="@post('/search')">Search</button>`
  - `<div id="search-results"></div>`
  - `<a href="/recipe/new">Create New Recipe</a>`
- [x] In `views/pages/home.templ`, add `templ SearchResults(recipes []store.Recipe, selectedRestaurant string)` for `SearchHandler.ServeHTTP` to reuse:
  - If `selectedRestaurant == ""`, render the prompt `Select a restaurant to search recipes.`.
  - If `selectedRestaurant != ""` and `len(recipes) == 0`, render `No recipes found.`.
  - Otherwise render a list of normal `<a href="/recipe/{id}">...</a>` links to recipe view pages.
- [x] Edit `views/components/header.templ` and add a Datastar change action to the existing restaurant selector so switching restaurants refreshes the results area without leaving stale results on screen:
  - Add `data-on:change="@post('/search')"` to both versions of the `<select data-bind:selected-restaurant>` element.
- [x] Edit `internal/handler/home_test.go` and extend the existing tests instead of creating a new home handler:
  - Update `TestHomePage_Renders` to assert the rendered body now contains `Search recipes...`, `id="search-results"`, and `Create New Recipe` when restaurants exist.
  - Update `TestHomePage_NoRestaurants` to assert the search form and create link are absent when the user has no restaurants.
- [x] Run `go test ./internal/handler -run 'TestHomePage'` to verify the template changes still satisfy the home-page tests.

### Task 5: Create recipe view template and handler (TDD)

- [x] Create `views/pages/types.go` and define shared page types in package `pages`:
  - `type IngredientWithFood struct { FoodName string; Quantity float64; Unit string; SortOrder int64 }`
  - `type IngredientRow struct { FoodName string; Quantity string; Unit string }` so Task 6 can reuse the same package-local type file.
- [x] Create `views/pages/recipe.templ` and define `templ RecipeView(username string, restaurants []store.Restaurant, recipe store.Recipe, ingredients []IngredientWithFood, canEdit bool, canDelete bool)`.
- [x] In `views/pages/recipe.templ`, render:
  - `components.Header(username, restaurants)` and `components.Footer()`.
  - A page root that predefines the existing `selected-restaurant` Datastar signal from `recipe.RestaurantID` so the shared header stays aligned with the viewed recipe.
  - Recipe name, yield, and instructions.
  - An ingredient section that preserves `ListIngredientsByRecipeID` order.
  - A stable empty state such as `No ingredients added yet.` when `len(ingredients) == 0`.
  - An edit link only when `canEdit` is `true`.
  - A delete button only when `canDelete` is `true`.
  - A placeholder container `<div id="delete-confirmation"></div>` for Task 9.
- [x] In `views/pages/recipe.templ`, add a small fragment component such as `templ RecipeDeleteDialog(recipe store.Recipe)` that Task 9 can patch into `#delete-confirmation`.
- [x] Make `RecipeDeleteDialog` render both confirmation actions:
  - A confirm button that calls `@delete('/recipe/{id}')`
  - A cancel button that calls `@post('/recipe/{id}/delete-cancel')`
- [x] Make the delete button trigger the Datastar endpoint with `data-on:click="@post('/recipe/{id}/delete-confirm')"`.
- [x] Create `internal/handler/recipe_view_test.go` and add:
  - `TestRecipeView_Success` — seed a recipe with ingredients and food rows, request `GET /recipe/{id}`, assert `200`, recipe name, yield, instructions, and ingredient names are present.
  - `TestRecipeView_NoIngredients` — seed a recipe with no ingredient rows, assert the empty-ingredients state text appears.
  - `TestRecipeView_NotFound` — request a missing recipe ID and assert `404`.
  - `TestRecipeView_ForbiddenRestaurant` — seed a recipe in a restaurant outside the JWT claim set and assert `403`.
  - `TestRecipeView_StaffCanView` — seed a `staff` claim with restaurant access and assert the page renders without edit/delete controls.
  - `TestRecipeView_AdminCanEditDelete` — seed an `admin` claim with restaurant access and assert the page includes edit and delete controls.
- [x] Run `go test ./internal/handler -run 'TestRecipeView'` and verify the new tests fail before implementation.
- [x] Create `internal/handler/recipe_view.go` and define `type RecipeViewHandler struct { DB store.DBTX }`.
- [x] Implement `func (h *RecipeViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)` in `internal/handler/recipe_view.go`:
  1. Read the recipe ID with `chi.URLParam(r, "id")`.
  2. Read claims from context; for browser-page behavior, follow the existing `HomeHandler` pattern and redirect to `/login` if claims are missing.
  3. Call `store.GetRecipeByID(h.DB, id)` and return `404` on `store.ErrNotFound`.
  4. Verify `recipe.RestaurantID` is present in `claims.RestaurantIDs`; otherwise return `403`.
  5. Call `store.ListIngredientsByRecipeID(h.DB, id)`.
  6. For each ingredient, call `store.GetFoodByID(h.DB, ingredient.FoodID)` and map the result to `pages.IngredientWithFood`.
  7. Set `canEdit` and `canDelete` to `true` only when the user has restaurant access and `claims.Role` is `admin` or `manager`.
  8. Fetch the current user with `store.GetUserByID(h.DB, claims.UserID)` and accessible restaurants with `store.ListRestaurantsByUserID(h.DB, claims.UserID)` for the shared header.
  9. Render `pages.RecipeView(user.Username, restaurants, *recipe, ingredientsWithFood, canEdit, canDelete)`.
- [x] Run `go test ./internal/handler -run 'TestRecipeView'` again and verify the new tests pass.

### Task 6: Create recipe form template

- [x] Create `views/pages/recipe_form.templ` and define `templ RecipeForm(username string, restaurants []store.Restaurant, recipe store.Recipe, ingredients []IngredientRow, errors map[string]string, isEdit bool)`.
- [x] Make `RecipeForm` a full-page component, not an SSE-only fragment:
  - Include `components.Header(username, restaurants)` and `components.Footer()`.
  - Wrap the full page in a stable root container such as `<div id="recipe-form-page">...</div>` so create/edit validation failures can patch the page back into place with SSE.
  - Predefine the shared `selected-restaurant` Datastar signal from `recipe.RestaurantID` on that root container so edit pages and validation rerenders keep the header restaurant selection in sync.
- [x] In `views/pages/recipe_form.templ`, render fields for recipe `name`, `yield`, and `instructions`, pre-populated from the `recipe` argument.
- [x] Keep restaurant context tied to the existing header selector rather than introducing a second restaurant field inside the form:
  - The page should continue to use the existing `data-bind:selected-restaurant` signal from `views/components/header.templ`.
  - Re-rendered create forms should preserve the previously selected restaurant by carrying the submitted restaurant ID in `recipe.RestaurantID` and reusing the same Datastar signal name.
- [x] In `views/pages/recipe_form.templ`, predefine and render the ingredient-row state from `ingredients []IngredientRow` using a Datastar signal such as `data-signals:ingredients`.
- [x] Each ingredient row in `views/pages/recipe_form.templ` must render:
  - Food-name input
  - Quantity input
  - Unit input
  - Remove button for that row
- [x] Add an `Add Ingredient` button in `views/pages/recipe_form.templ` that appends a blank row to the Datastar `ingredients` signal.
- [x] Make each row’s remove button remove only that row while preserving the remaining rows’ order and current values.
- [x] Render validation errors from `errors map[string]string` inline next to the relevant recipe and ingredient fields.
- [x] Set the submit action in `views/pages/recipe_form.templ` by `isEdit`:
  - Create mode submits with `@post('/recipe/new')`
  - Edit mode submits with `@patch('/recipe/{id}')`

### Task 7: Create recipe create handler (TDD)

- [x] Create `internal/handler/recipe_create_test.go` and add `TestRecipeCreate_GetRendersForm`:
  - Reuse `setupTestDB(t)`.
  - Seed an authenticated user with at least one restaurant.
  - Wrap the handler with `auth.BrowserMiddleware(testJWTSecret)`.
  - Request `GET /recipe/new` and assert the returned HTML contains the form fields, `Add Ingredient`, and the create submit action.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_Success`:
  - Submit Datastar JSON signals for a valid recipe with no ingredients.
  - Assert the response is SSE and contains a redirect to `/recipe/{id}`.
  - Query the database and assert exactly one new recipe row exists with the submitted values.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_WithIngredients` and assert two submitted rows are persisted through `store.ListIngredientsByRecipeID` with the correct order.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_NewFood` and assert a missing food name causes a new `food` row to be created and linked.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_ExistingFood` and assert an existing `food` row is reused rather than duplicated.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_MissingName` and assert the SSE body re-renders the form with `Name is required.` and no saved recipe.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_InvalidYield` and assert the SSE body re-renders the form with a stable yield error such as `Yield must be a valid integer.`.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_IngredientMissingFoodName` and assert the SSE body re-renders with an ingredient-row error and no saved recipe.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_UnauthorizedRestaurant` and assert a submitted restaurant ID outside `claims.RestaurantIDs` returns `403`.
- [x] In `internal/handler/recipe_create_test.go`, add `TestRecipeCreate_Unauthenticated` and assert the browser-middleware-wrapped route redirects to `/login`.
- [x] Run `go test ./internal/handler -run 'TestRecipeCreate'` and verify the new tests fail before implementation.
- [x] Create `internal/handler/recipe_create.go` and define:
  - `type IngredientSignal struct` with fields `FoodName string`, `Quantity string`, and `Unit string`, tagged as `json:"foodName"`, `json:"quantity"`, and `json:"unit"`
  - `type RecipeFormSignals struct` with fields `Name string`, `Yield string`, `Instructions string`, `Ingredients []IngredientSignal`, and `RestaurantID string`, tagged as `json:"name"`, `json:"yield"`, `json:"instructions"`, `json:"ingredients"`, and `json:"selectedRestaurant"`
  - `type RecipeCreateHandler struct { DB store.DBTX }`
- [x] Implement `func (h *RecipeCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)` in `internal/handler/recipe_create.go`:
  - `GET`:
    1. Read claims from context; if missing, redirect to `/login`.
    2. Fetch the current user and accessible restaurants.
    3. Render `pages.RecipeForm(user.Username, restaurants, store.Recipe{}, []pages.IngredientRow{{}}, map[string]string{}, false)`.
  - `POST`:
    1. Read claims from context; if missing, redirect to `/login`.
    2. Call `datastar.ReadSignals(r, &signals)` before creating the SSE writer.
    3. Validate `signals.RestaurantID` is non-empty; if blank, re-render the form with a restaurant error such as `Select a restaurant.`.
    4. Validate `signals.RestaurantID` is present in `claims.RestaurantIDs`; if not, return `403`.
    5. Validate recipe fields:
       - `Name` is required after `strings.TrimSpace`.
       - `Yield` parses with `strconv.ParseInt(..., 10, 64)`.
       - Each ingredient row requires non-blank `FoodName`, valid `Quantity` via `strconv.ParseFloat`, and non-blank `Unit`.
    6. If validation fails, rebuild `store.Recipe` and `[]pages.IngredientRow` from the submitted signals, render `pages.RecipeForm(...)`, open `datastar.NewSSE(w, r)`, and `PatchElements` back into `#recipe-form-page`.
    7. For successful submissions, start a database transaction by type-asserting `h.DB` to something with `Begin() (*sql.Tx, error)`; use the resulting `*sql.Tx` for all store calls so recipe, ingredient, and food changes are all-or-nothing.
    8. Resolve food names in submission order after trimming whitespace:
       - Deduplicate repeated names within the same submission.
       - Call `store.GetFoodByName(tx, name)` first.
       - If `errors.Is(err, store.ErrNotFound)`, create a new `store.Food{ID: uuid.New().String(), Name: name}` with `store.CreateFood(tx, &food)`.
       - Reuse the resolved food ID for every matching ingredient row.
    9. Create the recipe with `store.CreateRecipe(tx, &recipe)` where `recipe.ID = uuid.New().String()`, `recipe.RestaurantID = signals.RestaurantID`, and `CreatedAt`/`UpdatedAt` use `time.Now().Unix()`.
    10. Convert submitted ingredients to `[]store.Ingredient` with generated IDs, resolved `FoodID` values, parsed quantities, and `SortOrder` values starting at `1`.
    11. Persist ingredients with `store.ReplaceIngredients(tx, recipe.ID, ingredients)`.
    12. Commit the transaction.
    13. Open `sse := datastar.NewSSE(w, r)` and call `sse.Redirect("/recipe/" + recipe.ID)`.
- [x] Keep create authorization restaurant-scoped only; do not add an admin/manager role requirement to `RecipeCreateHandler`.
- [x] Run `go test ./internal/handler -run 'TestRecipeCreate'` again and verify the new tests pass.

### Task 8: Create recipe edit handler (TDD)

- [x] Create `internal/handler/recipe_edit_test.go` and add `TestRecipeEdit_GetRendersForm`:
  - Seed a recipe with at least one ingredient.
  - Request `GET /recipe/{id}/edit` as an authorized `admin` or `manager`.
  - Assert the form is pre-populated with the saved recipe values and ingredient rows.
- [x] In `internal/handler/recipe_edit_test.go`, add `TestRecipeEdit_Success` and assert `PATCH /recipe/{id}` updates recipe name, yield, and instructions.
- [x] In `internal/handler/recipe_edit_test.go`, add `TestRecipeEdit_UpdatesIngredients` and assert old ingredient rows are removed and replaced with the submitted rows/order.
- [x] In `internal/handler/recipe_edit_test.go`, add `TestRecipeEdit_ForbiddenRole` and assert a `staff` user with restaurant access gets `403`.
- [x] In `internal/handler/recipe_edit_test.go`, add `TestRecipeEdit_ForbiddenRestaurant` and assert a user without restaurant access gets `403`.
- [x] In `internal/handler/recipe_edit_test.go`, add `TestRecipeEdit_NotFound` and assert a missing recipe returns `404`.
- [x] In `internal/handler/recipe_edit_test.go`, add `TestRecipeEdit_ValidationFailure` and assert the SSE body re-renders the form with errors and leaves the stored recipe unchanged.
- [x] Run `go test ./internal/handler -run 'TestRecipeEdit'` and verify the new tests fail before implementation.
- [x] Create `internal/handler/recipe_edit.go` and define `type RecipeEditHandler struct { DB store.DBTX }`.
- [x] Reuse `RecipeFormSignals` and `IngredientSignal` from `internal/handler/recipe_create.go` so create and edit share the same request payload structure and validation rules.
- [x] Implement `func (h *RecipeEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)` in `internal/handler/recipe_edit.go`:
  - `GET`:
    1. Read claims; if missing, redirect to `/login`.
    2. Load the recipe with `store.GetRecipeByID(h.DB, chi.URLParam(r, "id"))`.
    3. Return `404` on `store.ErrNotFound`.
    4. Return `403` if the recipe restaurant is not in `claims.RestaurantIDs`.
    5. Return `403` unless `claims.Role` is `admin` or `manager`.
    6. Load ingredients with `store.ListIngredientsByRecipeID` and map each row to `pages.IngredientRow` by looking up the food name through `store.GetFoodByID`.
    7. Load the current user and restaurant list for the header.
    8. Render `pages.RecipeForm(user.Username, restaurants, *recipe, ingredientRows, map[string]string{}, true)`.
  - `PATCH`:
    1. Read claims; if missing, redirect to `/login`.
    2. Load the existing recipe and repeat the same not-found, restaurant-access, and role checks as `GET`.
    3. Call `datastar.ReadSignals(r, &signals)` before creating the SSE writer.
    4. Reuse the same validation rules and food-resolution flow as `RecipeCreateHandler`.
    5. Do not change `recipe.RestaurantID`; edit stays pinned to the stored recipe’s restaurant.
    6. Start a transaction, resolve food rows, call `store.UpdateRecipe(tx, &recipe)`, then replace ingredients with `store.ReplaceIngredients(tx, recipe.ID, ingredients)`.
    7. Set `recipe.UpdatedAt = time.Now().Unix()` before `store.UpdateRecipe`.
    8. On validation failure, rebuild the form state, render `pages.RecipeForm(...)`, and patch `#recipe-form-page` via SSE.
    9. On success, commit the transaction and `sse.Redirect("/recipe/" + recipe.ID)`.
- [x] Run `go test ./internal/handler -run 'TestRecipeEdit'` again and verify the new tests pass.

### Task 9: Create recipe delete handler (TDD)

- [x] Create `internal/handler/recipe_delete_test.go` and add `TestDeleteConfirm_ShowsDialog`:
  - Seed an accessible recipe.
  - Request `POST /recipe/{id}/delete-confirm` as an authorized `admin` or `manager`.
  - Assert the response is SSE and contains the confirmation-dialog HTML plus the recipe name.
- [x] In `internal/handler/recipe_delete_test.go`, add `TestDeleteConfirm_ForbiddenRole` and assert `staff` gets `403`.
- [x] In `internal/handler/recipe_delete_test.go`, add `TestDeleteConfirm_ForbiddenRestaurant` and assert wrong-restaurant access gets `403`.
- [x] In `internal/handler/recipe_delete_test.go`, add `TestDeleteCancel_RemovesDialog` and assert `POST /recipe/{id}/delete-cancel` returns SSE that restores the empty `#delete-confirmation` container.
- [x] In `internal/handler/recipe_delete_test.go`, add `TestDelete_Success` and assert `DELETE /recipe/{id}` removes the recipe, cascades ingredients, and redirects to `/` via SSE.
- [x] In `internal/handler/recipe_delete_test.go`, add `TestDelete_NotFound` and assert deleting a missing recipe returns `404`.
- [x] In `internal/handler/recipe_delete_test.go`, add `TestDelete_ForbiddenRole` and assert `staff` cannot issue the final delete.
- [x] Run `go test ./internal/handler -run 'TestDelete'` and verify the new tests fail before implementation.
- [x] Create `internal/handler/recipe_delete.go` and define `type RecipeDeleteHandler struct { DB store.DBTX }`.
- [x] Implement `func (h *RecipeDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request)` in `internal/handler/recipe_delete.go` to dispatch by route/method:
  - `POST /recipe/{id}/delete-confirm`
  - `POST /recipe/{id}/delete-cancel`
  - `DELETE /recipe/{id}`
- [x] In the `delete-confirm` branch of `RecipeDeleteHandler.ServeHTTP`:
  1. Read claims from context; if missing, redirect to `/login`.
  2. Load the recipe with `store.GetRecipeByID` and return `404` on `store.ErrNotFound`.
  3. Return `403` if the recipe restaurant is not in `claims.RestaurantIDs`.
  4. Return `403` unless `claims.Role` is `admin` or `manager`.
  5. Render `pages.RecipeDeleteDialog(*recipe)` and patch it into `#delete-confirmation` with `datastar.NewSSE(w, r)` and `PatchElements`.
- [x] In the `delete-cancel` branch of `RecipeDeleteHandler.ServeHTTP`:
  - Return an SSE response that patches `#delete-confirmation` back to an empty placeholder such as `<div id="delete-confirmation"></div>`.
- [x] In the `DELETE /recipe/{id}` branch of `RecipeDeleteHandler.ServeHTTP`:
  1. Read claims from context; if missing, redirect to `/login`.
  2. Load the recipe and repeat the same not-found, restaurant-access, and role checks as the confirmation branch.
  3. Call `store.DeleteRecipe(h.DB, id)`.
  4. Open an SSE response and call `sse.Redirect("/")`.
- [x] Keep delete confirmation backend-driven by rendering the dialog only from `POST /recipe/{id}/delete-confirm`; do not move confirmation markup generation into client-authored JavaScript.
- [x] Run `go test ./internal/handler -run 'TestDelete'` again and verify the new tests pass.

### Task 10: Update CSS for recipe pages

- [x] Edit `static/css/style.css` and add styles that match the new markup introduced in `views/pages/home.templ`, `views/pages/recipe.templ`, and `views/pages/recipe_form.templ`.
- [x] Add search styles in `static/css/style.css` for the home-page controls and results area:
  - Inline search form layout for the input and button.
  - Results list styling for recipe links.
  - Empty-state styling for the select-a-restaurant and no-results messages.
- [x] Add recipe-view styles in `static/css/style.css` for:
  - Recipe detail layout
  - Yield/instruction blocks
  - Ingredient list or table
  - Action buttons row
- [x] Add recipe-form styles in `static/css/style.css` for:
  - Overall form layout
  - Field groups
  - Ingredient row layout
  - Add/remove button placement
  - Inline validation error text
- [x] Add delete-confirmation styles in `static/css/style.css` for the confirmation dialog or inline confirmation block rendered into `#delete-confirmation`.
- [x] Add button variants in `static/css/style.css` so the new templates can use a consistent class set such as:
  - `.button-primary` for create/save/search
  - `.button-danger` for delete/confirm delete
  - `.button-secondary` for cancel/back actions

### Task 11: Register routes in `cmd/server/main.go`

- [x] Edit `cmd/server/main.go` and instantiate the new handlers after the existing auth/home handlers:
  - `searchHandler := &handler.SearchHandler{DB: database}`
  - `recipeViewHandler := &handler.RecipeViewHandler{DB: database}`
  - `recipeCreateHandler := &handler.RecipeCreateHandler{DB: database}`
  - `recipeEditHandler := &handler.RecipeEditHandler{DB: database}`
  - `recipeDeleteHandler := &handler.RecipeDeleteHandler{DB: database}`
- [x] Keep the Datastar import where it is actually used:
  - The direct `github.com/starfederation/datastar-go/datastar` imports belong in `internal/handler/search.go`, `internal/handler/recipe_create.go`, `internal/handler/recipe_edit.go`, and `internal/handler/recipe_delete.go`.
  - Do not leave an unused `datastar` import in `cmd/server/main.go`.
- [x] Edit the browser-auth route group in `cmd/server/main.go` and register the milestone 005 routes with `auth.BrowserMiddleware([]byte(jwtSecret))`:
  - `r.Post("/search", searchHandler.ServeHTTP)`
  - `r.Get("/recipe/new", recipeCreateHandler.ServeHTTP)`
  - `r.Post("/recipe/new", recipeCreateHandler.ServeHTTP)`
  - `r.Get("/recipe/{id}", recipeViewHandler.ServeHTTP)`
  - `r.Get("/recipe/{id}/edit", recipeEditHandler.ServeHTTP)`
  - `r.Patch("/recipe/{id}", recipeEditHandler.ServeHTTP)`
  - `r.Post("/recipe/{id}/delete-confirm", recipeDeleteHandler.ServeHTTP)`
  - `r.Post("/recipe/{id}/delete-cancel", recipeDeleteHandler.ServeHTTP)`
  - `r.Delete("/recipe/{id}", recipeDeleteHandler.ServeHTTP)`
- [x] Register `/recipe/new` before `/recipe/{id}` in `cmd/server/main.go` so the static `new` route is not shadowed by the path parameter route.
- [x] Keep `/search` and the delete-confirm/delete-cancel endpoints inside the browser-middleware group because Datastar requests originate from the browser and should use the existing cookie-based auth flow.
- [x] Run `go build ./cmd/server` after the route wiring change to verify the server entry point still compiles.

### Task 12: Run templ generate and full test suite

- [x] Run `make generate` from the project root so the new or updated `.templ` files generate `_templ.go` companions.
- [x] Run `go test ./...` and fix any failing store, handler, or generated-template tests before handoff.
- [x] Run `go vet ./...` and resolve any vet findings.
- [x] Run `go build ./...` as the final compile check.
- [x] Verify the generated code includes the expected new template outputs for:
  - `views/pages/home.templ`
  - `views/pages/recipe.templ`
  - `views/pages/recipe_form.templ`
