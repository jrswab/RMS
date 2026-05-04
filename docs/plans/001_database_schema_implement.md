# Implementation Guide 001: Database Schema, Migrations, Connection Layer, and Seed Data

**Spec:** `docs/plans/001_database_schema_spec.md`

---

## Section 1: Context Summary

This milestone lays the foundation for the entire Recipe Management System. No application code exists yet — the repository is a greenfield skeleton with only `go.mod`, `Makefile`, and empty `migrations/` and `static/` directories. Before any HTTP handlers, templates, or business logic can be built, the system needs a database it can connect to, schema it can trust, and test data it can develop against. This implementation creates the SQLite schema (6 tables), a connection layer with proper pragmas, an embedded migration runner that applies schema on startup, and a seed script that populates reproducible test data. Every subsequent milestone depends on this one.

---

## Section 2: Implementation Checklist

### Task 1: Create migration file with all 6 tables

- [x] Create `migrations/001_create_tables.up.sql`
  - Define `schema_migrations` table: `version TEXT PRIMARY KEY`, `applied_at INTEGER NOT NULL`
  - Define `restaurants` table: `id TEXT PRIMARY KEY`, `name TEXT NOT NULL`
  - Define `users` table: `id TEXT PRIMARY KEY`, `username TEXT NOT NULL UNIQUE`, `password TEXT NOT NULL`, `role TEXT NOT NULL`, `created_at INTEGER NOT NULL`, `updated_at INTEGER NOT NULL`
  - Define `food` table: `id TEXT PRIMARY KEY`, `name TEXT NOT NULL`
  - Define `user_restaurants` table: `user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE`, `restaurant_id TEXT NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE`, `UNIQUE(user_id, restaurant_id)`
  - Define `recipes` table: `id TEXT PRIMARY KEY`, `name TEXT NOT NULL`, `restaurant_id TEXT NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE`, `instructions TEXT NOT NULL`, `yield INTEGER NOT NULL`, `created_at INTEGER NOT NULL`, `updated_at INTEGER NOT NULL`
  - Define `ingredients` table: `id TEXT PRIMARY KEY`, `recipe_id TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE`, `food_id TEXT NOT NULL REFERENCES food(id) ON DELETE RESTRICT`, `quantity REAL NOT NULL`, `unit TEXT NOT NULL`, `sort_order INTEGER NOT NULL`, `UNIQUE(recipe_id, sort_order)`
  - Create indexes: `idx_recipes_restaurant_id ON recipes(restaurant_id)`, `idx_recipes_name ON recipes(name)`, `idx_ingredients_recipe_id ON ingredients(recipe_id)`

### Task 2: Implement connection layer

- [x] Create `internal/db/db.go`
  - Function `Open(dbPath string) (*sql.DB, error)` — opens SQLite at the given path, applies `PRAGMA foreign_keys = ON` and `PRAGMA journal_mode = WAL`, returns the `*sql.DB` handle
  - Function `DefaultPath() string` — reads `DB_PATH` env var, returns `recipes.db` if unset
  - On open failure (missing directory, corrupt file), return a clear wrapped error

### Task 3: Test connection layer

- [x] Create `internal/db/db_test.go`
  - Test `TestOpen` — open an in-memory SQLite DB (`:memory:`), verify no error, verify `foreign_keys` pragma is `1` via `PRAGMA foreign_keys` query, verify `journal_mode` pragma is `wal` via `PRAGMA journal_mode` query
  - Test `TestOpen_InvalidPath` — open a path with a non-existent directory component, verify error is returned
  - Test `TestDefaultPath` — unset `DB_PATH`, verify returns `recipes.db`; set `DB_PATH` to `/tmp/test.db`, verify returns `/tmp/test.db`

### Task 4: Implement migration runner

- [x] Create `internal/db/migrate.go`
  - Embed the `migrations/` directory using `//go:embed migrations/*.up.sql` (note: the embed directive must reference the relative path from the package; place the embed in a package that can reach `migrations/` — see edge case below)
  - Function `RunMigrations(db *sql.DB) error` — reads embedded `.up.sql` files, sorts by filename, creates `schema_migrations` if not exists, queries applied versions, skips already-applied, executes new migrations in order within a transaction, inserts into `schema_migrations` on success. Returns error on any failure; does not partial-apply.
  - **Embed path edge case:** Go's `//go:embed` directive cannot reference files outside the package directory. Since `migrations/` is at the project root and `internal/db/` is nested, the embed must live at the project root level. Create `migrations.go` at the project root (package `lab37`) or use a dedicated `migrations/embed.go` file in a `migrations` package that exports the embedded FS. The `internal/db/migrate.go` function should accept the embedded `fs.FS` as a parameter: `RunMigrations(db *sql.DB, migrationFS fs.FS) error`. This keeps the runner testable and decoupled from the embed location.
  - Create `migrations/embed.go` (package `migrations`): `//go:embed *.up.sql` directive on a `var FS embed.FS` variable that exports the embedded filesystem.

### Task 5: Test migration runner

- [x] Create `internal/db/migrate_test.go`
  - Test `TestRunMigrations` — open `:memory:` DB, run migrations using the real embedded FS, verify `schema_migrations` table exists and has one row, verify all 6 tables exist by querying `sqlite_master`, verify column counts match spec
  - Test `TestRunMigrations_Idempotent` — run migrations twice on the same DB, verify `schema_migrations` still has exactly one row (not duplicated), verify no error on second run
  - Test `TestRunMigrations_EmptyFS` — pass an `fstest.MapFS{}` (empty), verify no error (no-op), verify `schema_migrations` table is created but empty
  - Test `TestRunMigrations_ForeignKeys` — after migration, insert a `user_restaurants` row referencing a non-existent user, verify it fails with a foreign key error (confirms `PRAGMA foreign_keys = ON` is effective at migration time — note: the connection layer sets this, so the test must use a DB opened via `db.Open()`)

### Task 6: Implement seed script

- [x] Create `cmd/seed/main.go`
  - `main()` opens DB via `internal/db.Open(internal/db.DefaultPath())`, runs migrations via `internal/db.RunMigrations(db, migrations.FS)`, then calls `seed(db)` and exits
  - Function `seed(db *sql.DB) error` — inserts all test data using `INSERT OR IGNORE` for idempotency
  - Insert 3 users: `admin` (role `admin`), `manager` (role `manager`), `staff` (role `staff`) — all with bcrypt hash of `password123`, deterministic UUIDs (e.g., `uuid.NewSHA1(uuid.NameSpaceURL, []byte("admin"))` or similar)
  - Insert 2 restaurants: `The Rusty Spoon`, `Copper Kettle` — deterministic UUIDs
  - Insert user-restaurant links: admin → both, manager → spoon, staff → spoon
  - Insert 8 food items: flour, sugar, butter, eggs, salt, milk, chicken, rice — deterministic UUIDs
  - Insert 3–5 recipes belonging to `The Rusty Spoon`, each with 3–6 ingredients referencing the food items, with varying quantities, units, and sort orders
  - All timestamps (`created_at`, `updated_at`) use a fixed value (e.g., `time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Unix()`) for reproducibility

### Task 7: Test seed script

- [x] Create `cmd/seed/seed_test.go`
  - Test `TestSeed` — open `:memory:` DB via `db.Open(":memory:")`, run migrations, call `seed(db)`, verify: 3 users exist with correct usernames/roles, 2 restaurants exist with correct names, 6 user_restaurant links exist, 8 food items exist, 3–5 recipes exist all linked to `The Rusty Spoon`, each recipe has 3–6 ingredients
  - Test `TestSeed_Idempotent` — call `seed(db)` twice, verify row counts are unchanged (no duplicates)

### Task 8: Update Makefile

- [x] Update `Makefile`
  - Add `seed` target: `go run ./cmd/seed`
  - Add `seed` to `.PHONY`
  - Existing targets (`run`, `test`, `tidy`, `clean`) remain unchanged
