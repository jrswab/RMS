# Spec 005: Recipe CRUD and Search

## Section 1: Context & Constraints

### Milestone Entry

> 005: Recipe CRUD and search — search endpoint with Datastar reactivity, recipe view page, create/edit forms with ingredient management, delete with role enforcement

### Research Findings — Relevant Context

**Codebase state (post-milestone 004):**

```text
lab37/
├── cmd/server/main.go                  # login/logout/static/home routes only
├── internal/auth/                      # JWT claims, API auth, browser auth, RBAC
├── internal/handler/                   # login/home handlers; no recipe handlers yet
├── internal/store/
│   ├── recipe.go                       # recipe lookup/search/create/update/delete
│   ├── ingredient.go                   # ingredient listing and full replacement
│   ├── food.go                         # food lookup/list only
│   └── models.go
├── views/
│   ├── pages/home.templ                # placeholder home page
│   ├── pages/login.templ
│   └── components/header.templ         # restaurant dropdown binds selected-restaurant
└── static/css/style.css
```

- `GET /` already renders an authenticated full HTML home page behind browser auth, but it currently contains only a welcome message and no recipe/search UI.
- The header already exposes the restaurant selector with `data-bind:selected-restaurant`; per Datastar's naming rules this is available to expressions and requests as `$selectedRestaurant`.
- The current server registers login/logout/browser routes, static assets, and an authenticated API route group, but milestone 005 recipe/search routes do not exist yet.
- `auth.Claims` already contains `UserID`, `Role`, and `RestaurantIDs`, which is enough to make restaurant-access and role decisions from the presented JWT claims.
- API auth accepts either `Authorization: Bearer <token>` or the `auth_token` cookie; browser page auth already redirects unauthenticated users to `/login`.
- The store layer already provides recipe retrieval, recipe search, recipe create/update/delete, ingredient list retrieval in saved order, full ingredient replacement for an existing recipe, and food lookup by ID plus full food listing.
- The current store layer does **not** yet provide food lookup by name or food creation, so inline food creation is new milestone 005 behavior.
- Existing recipe search behavior is already restaurant-scoped and treats `%`, `_`, and `\` as literal search characters. An empty search string matches all recipes for the selected restaurant.
- The schema makes food global (`food` is not restaurant-scoped), ingredients ordered (`sort_order`), recipe delete cascade into ingredients, and food delete restricted when ingredients reference it.
- `food.name` is not unique at the schema level, so duplicate prevention/reuse is an application behavior rather than a database-enforced constraint.
- `go.mod` currently targets Go `1.23.0` with `templ`, `chi`, `jwt`, `uuid`, `sqlite3`, and `x/crypto`; no frontend build tooling exists.
- The repository currently includes the Datastar browser script through the base layout, but no server-side Datastar response handling has been added yet.

**Datastar research findings (Context7):**

- `data-bind` creates two-way bindings between form controls and signals. Hyphenated names like `selected-restaurant` become `$selectedRestaurant`.
- Datastar backend requests can use GET, POST, PATCH, and DELETE. GET sends signals as query parameters; non-GET requests send signals in the request body by default.
- Datastar responses can stream multiple SSE events that patch HTML fragments, patch signals, and redirect the browser in a single request cycle.
- HTML fragment patching is the normal way to update part of the page without a full reload.
- Signals must be read from the request before the server opens the SSE response stream.

### Decisions Already Made (do not re-evaluate)

1. The stack remains Go + Templ + Datastar + SQLite.
2. The frontend remains server-rendered with no JavaScript build step.
3. Backend-rendered state remains the source of truth for interactive UI behavior.
4. Search is scoped to one restaurant at a time.
5. Access checks use the restaurant IDs carried in the authenticated user's JWT.
6. The existing endpoint contract remains the authority for mutation permissions: create requires restaurant access; edit and delete require both restaurant access and a permitted role (`admin` or `manager`).
7. Users who are not allowed to edit/delete do not get those actions in the normal UI, but server-side authorization still decides every request.
8. Recipe/search text is treated as literal user input and must not gain wildcard or markup behavior through the transport layer.

### Constraints

- Current repository constraint: Go `1.23.0` with toolchain `go1.24.3`.
- SQLite remains the system of record.
- No milestone 005 behavior can depend on a frontend bundler or SPA runtime.
- Search must return HTML fragments over Datastar SSE rather than JSON payloads.
- Restaurant context for search must come from the Datastar signal `$selectedRestaurant`.
- "Access" means the recipe's `restaurant_id` is present in the authenticated user's JWT `restaurant_ids` claim.
- Authorization decisions are based on the claims in the JWT presented with the request.
- The Recipe View page must be a normal full page load, not a Datastar partial.
- Delete confirmation state must remain backend-driven, not client-authored.

### User-Confirmed Decisions

| Decision | Choice |
|----------|--------|
| Search response format | HTML fragments via Datastar SSE |
| Search restaurant context | Restaurant ID comes from Datastar signal `$selectedRestaurant` |
| Ingredient row UX | Inline add/remove rows |
| Food field UX | Food name is free text |
| New food behavior | Server creates a food record when a non-empty typed food name does not already exist |
| Delete UX | Confirmation dialog is Datastar-driven and backed by server state |
| Post-create redirect | Redirect to the Recipe View page |
| Post-edit redirect | Redirect to the Recipe View page |
| Post-delete redirect | Redirect to the Home page |
| Access definition | Recipe access = recipe `restaurant_id` exists in JWT `restaurant_ids` |
| Recipe View rendering model | Full page load, not a Datastar partial |

## Section 2: Requirements

### 2.1 Behaviors

#### 2.1.1 Home and Search

- The authenticated home page must expose a recipe search experience instead of a placeholder-only state.
- Search must always be scoped to the restaurant currently identified by `$selectedRestaurant`.
- Search must update the search results area without a full page reload.
- The search response must be an HTML fragment suitable for Datastar SSE patching.
- Search results must only contain recipes whose names match the submitted search text within the selected restaurant.
- Each search result must, at minimum, show the recipe name and provide normal navigation to that recipe's full Recipe View page.
- When the selected restaurant changes, the displayed results must update so the page no longer shows stale recipes from the previously selected restaurant.
- The home page must provide a create-recipe entry point that operates within the current restaurant context.
- If the user has no accessible restaurants, the home page must show a no-restaurant state instead of exposing recipe search results or a valid create flow.

#### 2.1.2 Recipe View

- The Recipe View page must be reachable by direct URL and by navigating from search results or post-mutation redirects.
- The Recipe View page must render as a full page load.
- The page must display the recipe name, yield, instructions, and ingredient list in saved order.
- The page must support read-only viewing for any authenticated user who has access to the recipe's restaurant.
- The page must expose edit and delete actions only when the current user is authorized for those actions.
- If a recipe has no ingredients, the page must render a clear empty-ingredients state rather than a broken or missing section.

#### 2.1.3 Create Recipe

- The create flow must require an authenticated user and an accessible restaurant context.
- Users with restaurant access must be able to create a recipe for that restaurant, including users whose role does not permit edit/delete of existing recipes.
- The create form must capture recipe name, restaurant context, yield, instructions, and zero or more ingredient rows.
- Each ingredient row must capture food name, quantity, and unit.
- Ingredient rows must be managed inline on the form without leaving the page.
- Adding a row must append a new blank ingredient row.
- Removing a row must remove that row and preserve the remaining rows' relative order and entered values.
- Each ingredient row must accept a free-text food name.
- On submission, each non-empty food name must resolve to an existing global food item or cause a new global food item to be created.
- A successful create submission must produce one complete saved recipe state: recipe data and its submitted ingredient list.
- On successful create, the user must be redirected to the new recipe's Recipe View page.
- If validation fails, the user must remain on the create form and see the current form values and validation errors.

#### 2.1.4 Edit Recipe

- The edit flow must load the existing recipe and ingredient data into an editable form.
- Only users with both recipe access and a permitted role (`admin` or `manager`) may edit a recipe.
- Editing must allow recipe field changes plus inline add/remove/update of ingredient rows.
- The submitted ingredient list becomes the recipe's new saved ingredient list and order.
- Food-name resolution on edit must follow the same rules as create: existing food is reused; new food is created when needed.
- A successful edit must update the targeted recipe only.
- On successful edit, the user must be redirected to that recipe's Recipe View page.
- If validation fails, the user must remain on the edit form and see the current form values and validation errors.

#### 2.1.5 Delete Recipe

- Delete must be a two-step action: initiating delete opens a confirmation dialog; no recipe is removed until the user explicitly confirms.
- The confirmation dialog must be driven by Datastar interactions whose authoritative state lives on the server.
- Only users with both recipe access and a permitted role (`admin` or `manager`) may delete a recipe.
- Canceling the dialog must leave the recipe unchanged and return the page to its non-confirming state.
- Confirming the dialog must delete the targeted recipe and its ingredient list.
- On successful delete, the user must be redirected to the Home page.

### 2.2 Interfaces

| Interface | Method | Result |
|-----------|--------|--------|
| Home page | `GET /` | Full HTML page with restaurant-aware search UI and create entry point |
| Search | `POST /search?q={text}` | Datastar SSE response that patches the search results area with HTML fragments; request includes `$selectedRestaurant` |
| Recipe View page | `GET /recipe/{id}` | Full HTML page for one recipe |
| Create Recipe page | `GET /recipe/new` | Full HTML page with blank recipe form in an accessible restaurant context |
| Create Recipe submit | `POST /recipe/new` | Creates a recipe, then redirects to `GET /recipe/{id}` on success |
| Edit Recipe page | `GET /recipe/{id}/edit` | Full HTML page with populated recipe form |
| Update Recipe submit | `PATCH /recipe/{id}` | Updates a recipe, then redirects to `GET /recipe/{id}` on success |
| Delete confirmation interaction | Datastar-backed page interaction | Opens/closes the confirmation dialog without leaving the current page |
| Delete Recipe submit | `DELETE /recipe/{id}` | Deletes a recipe, then redirects to `GET /` on success |

**Interface rules:**

- All interfaces require authentication.
- Search, view, edit, and delete must validate that the addressed restaurant context or recipe belongs to one of the authenticated user's current JWT `restaurant_ids` claims.
- Create, edit, and delete must validate authorization again at submit time using the JWT claims presented with that request, even if the user previously loaded the page successfully.
- Search and delete confirmation interactions return HTML-oriented Datastar responses, not JSON result payloads.
- Recipe View, Create, and Edit pages are browser pages, not API-only payloads.

### 2.3 Data Flow

#### 2.3.1 Search Flow

1. The authenticated user lands on the Home page.
2. The user selects a restaurant, which sets `$selectedRestaurant`.
3. The user supplies search text.
4. The search request sends the search text plus the current restaurant context.
5. The server validates authentication and confirms the requested restaurant is present in the user's current JWT `restaurant_ids` claims.
6. The server returns an HTML fragment over Datastar SSE containing the search-results state for that restaurant and query.
7. The browser updates the results area without a full page reload.
8. When the user chooses a result, the browser performs normal navigation to that recipe's full Recipe View page.

#### 2.3.2 View Flow

1. The user requests `GET /recipe/{id}` directly or via navigation from search/create/edit.
2. The server loads the recipe and its ingredients.
3. The server validates that the recipe's `restaurant_id` is present in the user's current JWT `restaurant_ids` claims.
4. The server renders a full Recipe View page.
5. The page shows only the actions the current user is permitted to take under the current JWT claims.

#### 2.3.3 Create and Edit Flow

1. The user opens the create or edit form in an authorized restaurant context.
2. The user enters or updates recipe fields and manages ingredient rows inline.
3. The user submits the form.
4. The server re-validates authentication, access, and any role requirement using the JWT claims presented with the submission request.
5. The server validates the submitted recipe fields and ingredient rows.
6. For each ingredient row, the server resolves the submitted food name to an existing global food item or creates a new global food item when needed.
7. The server applies the recipe and ingredient changes as one complete outcome: either the submitted recipe state is saved, or the submission is rejected without a partial recipe state becoming visible.
8. On success, the browser is redirected to the Recipe View page for the saved recipe.
9. On failure, the browser stays on the form page and sees errors with the submitted data preserved.

#### 2.3.4 Delete Flow

1. The user initiates delete from a page that exposes the action.
2. The server returns a Datastar-driven confirmation dialog state for the targeted recipe.
3. The user either cancels or confirms.
4. On cancel, the server returns the non-confirming page state and no data changes occur.
5. On confirm, the server re-validates authentication, access, and delete authorization using the JWT claims presented with the confirmation request.
6. If the delete is still authorized and the recipe still exists, the recipe is deleted.
7. The browser is redirected to the Home page.
8. If the delete is no longer valid, no recipe is deleted and the user receives a failure state for the current interaction.

### 2.4 Edge Cases

| Case | Expected Behavior |
|------|-------------------|
| Authentication expires before search, create, edit, or delete | The request fails authentication; no protected data or mutation is returned/applied |
| User has no accessible restaurants | Home page shows a no-restaurant state; search does not return recipe data; create flow is not available until an accessible restaurant exists |
| Search runs with no selected restaurant | No recipe results are returned; the page shows a prompt to select a restaurant |
| Search targets a restaurant ID outside the user's current JWT `restaurant_ids` claims | The request is rejected as an access failure and no recipe data is returned |
| Search text is empty | All recipes for the selected restaurant are eligible to appear in results |
| Search text contains `%`, `_`, or `\` | Those characters are treated as literal search text, not wildcard controls |
| Search text or recipe text contains HTML/script markup | The text is handled as literal user content and does not execute as markup/script |
| Search finds no matching recipes | The results area shows a no-results state |
| User changes restaurants after seeing results | The UI updates so only the current restaurant's results remain visible |
| User requests a recipe ID that does not exist | The request returns a not-found state |
| User requests a recipe that belongs to a restaurant they cannot access under the current JWT claims | The request returns a forbidden state and does not expose the recipe data |
| User with `staff` role tries to load the edit page or submit an edit directly | The request is rejected and no recipe is changed |
| User with `staff` role tries to open or confirm delete directly | The request is rejected and no recipe is deleted |
| User opens create flow without an accessible restaurant context | The request is rejected and no recipe is created |
| Create or edit submission omits the recipe name | The form is re-rendered with a validation error and no changes are saved |
| Create or edit submission uses a non-integer or otherwise invalid yield value | The form is re-rendered with a validation error and no changes are saved |
| Ingredient row is submitted with no food name or only whitespace | The form is re-rendered with a validation error and no recipe/ingredient changes are saved |
| Ingredient row is submitted with a missing or invalid quantity | The form is re-rendered with a validation error and no recipe/ingredient changes are saved |
| Ingredient row is submitted with a missing or blank unit | The form is re-rendered with a validation error and no recipe/ingredient changes are saved |
| Ingredient rows are all removed before submit | The recipe may be saved with zero ingredients |
| Multiple ingredient rows use the same already-existing food name | The existing global food item is reused for each matching row |
| Multiple ingredient rows use the same new food name in one submission | The food is created once and all matching rows resolve to that food |
| A submitted food name does not yet exist | The submission creates the new food item before linking the ingredient row |
| Any ingredient row fails validation during create/edit | The submission fails as a whole; the user does not see a partially updated recipe |
| The recipe is deleted by another action before an edit or delete submission completes | The request returns a not-found state and no further changes are applied |
| The user retries an action with a JWT that no longer grants the required restaurant access | The request is rejected using the claims in that JWT and no mutation occurs |
| The user retries edit/delete with a JWT that no longer grants the required role | The request is rejected using the claims in that JWT and no mutation occurs |
| The user's database access or role changes but the presented JWT still contains the old claims | Authorization continues to follow the presented JWT claims until the token is replaced or expires |
| User opens delete confirmation and then cancels | The dialog closes and the recipe remains unchanged |
| User confirms delete after the recipe is already gone | The request returns a not-found state and no other recipe is affected |
| User confirms delete without a valid server-side confirmation state | No delete occurs; the interaction returns a failure state rather than deleting speculatively |
| Recipe being viewed has zero ingredients | The Recipe View page shows an empty-ingredients state instead of an empty broken list |
