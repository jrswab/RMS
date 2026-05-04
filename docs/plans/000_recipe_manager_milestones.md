# RMS — Recipe Management System Milestones

## Section 1: Research Findings

### Codebase Structure

The repository is a greenfield skeleton. No application code exists yet.

```
lab37/
├── AGENTS.md              # Project-level agent rules
├── README.md              # Quick start (go run ./cmd/server, localhost:8080)
├── Makefile               # Targets: run, test, tidy, clean
├── go.mod                 # module lab37, Go 1.22
├── go.sum
├── docs/plans/
│   └── RMS00-design.md    # Design document (source of truth)
├── migrations/            # Empty — to be populated
└── static/                # Empty — to be populated
```

**Key observations:**
- `go.mod` already declares two dependencies: `github.com/golang-jwt/jwt/v5 v5.2.1` and `github.com/mattn/go-sqlite3 v1.14.22`.
- `Makefile` expects the entry point at `cmd/server` (does not exist yet).
- `Makefile` has a `clean` target that removes `recipes.db` — confirms SQLite file-based DB at project root.
- No `cmd/`, `internal/`, `views/`, or `static/` directories exist yet.
- No `.gitignore` exists.

### Design Document Summary (RMS000-design.md)

**Goal:** Web system to create, edit, delete, and display recipes for restaurants.

**UX Requirements:**
- User login required.
- Search scoped per restaurant.
- Restaurant dropdown selector (filtered by user permissions).
- Recipe display with instructions and ingredients.
- Edit capability for users with access.
- Delete capability restricted to admin role (frontend hides button, API rejects unauthorized requests).

**Database Schema — 6 tables:**

| Table | Purpose | Key Fields |
|-------|---------|------------|
| `recipes` | Core recipe data | id (UUID), name, restaurant_id (FK), instructions, yield (int), created_at, updated_at |
| `restaurants` | Restaurant entities | id (UUID), name |
| `users` | User accounts | id (UUID), username, password (hashed), role (admin/manager/staff), created_at, updated_at |
| `user_restaurants` | User-to-restaurant access (many-to-many) | user_id (UUID), restaurant_id (UUID) |
| `food` | Global food catalog (future expansion) | id (UUID), name |
| `ingredients` | Recipe ingredients linking to food | id (UUID), recipe_id (FK), food_id (FK), quantity (REAL), unit (text), sort_order (int) |

**API Endpoints:**

| Endpoint | Method | Auth | Role Restriction | Notes |
|----------|--------|------|------------------|-------|
| `/login` | POST | No | — | Returns JWT with user ID + restaurant IDs |
| `/search?q={text}` | POST | Yes | Any | Search by recipe name, scoped to restaurant. Query text must be escaped. |
| `/recipe/{id}` | GET | Yes | Any (with access) | Fetch recipe data |
| `/recipe/{id}` | DELETE | Yes | admin, manager | Delete recipe. Frontend hides button for staff; API rejects unauthorized. |
| `/recipe/{id}` | PATCH | Yes | admin, manager | Edit recipe. Text must be escaped. |
| `/recipe/new` | POST | Yes | Any (with access) | Create recipe. Text must be escaped. |

### Key Decisions Made

1. **Tech stack locked:** Go + Templ + Datastar + SQLite. No debate on alternatives — this is the take-home constraint.
2. **No frontend build step:** Server-rendered HTML via Templ. Datastar adds reactivity without a JS bundler.
3. **State stays on backend:** Datastar pattern — frontend is a thin render layer, all state management server-side.
4. **SQLite for now, PostgreSQL later:** Design explicitly allows migration path if demand grows.
5. **JWT in Authorization header:** All routes except `/login` require a valid JWT.
6. **Roles are text, not enum:** `admin`, `manager`, `staff` stored as plain text in the `users` table.

### Approaches Considered and Rejected

| Approach | Rejected Because |
|----------|-----------------|
| Separate frontend SPA (React/Vue) | Design explicitly chooses server-rendered HTML for simplicity and single-repo deployment |
| Session-based auth (cookies) | Design specifies JWT in Authorization header |
| ORM (GORM, etc.) | Not in go.mod; design implies direct SQL with repository pattern |
| Auto-increment IDs | Design specifies UUIDs for all primary keys |
| Separate auth service | Overkill for small userbase; monolith is the stated architecture |

### Constraints and Assumptions

**Constraints:**
- Go 1.22 (specified in go.mod)
- SQLite as the database (file-based, `recipes.db` at project root per Makefile)
- Zero frontend build step — no npm, no webpack, no bundler
- JWT-based authentication with bcrypt password hashing (user confirmed: hash + salt on backend)
- Datastar is new to the team — Context7 MCP must be consulted before generating any Datastar code (per AGENTS.md)

**Assumptions:**
- Single binary deployment (embedded migrations, embedded static assets)
- Small userbase (dozens, not thousands)
- One server instance (no horizontal scaling concern with SQLite)
- Port 8080 default (per README.md)

### Open Questions — Resolved

| Question | Answer |
|----------|--------|
| How should configuration be handled? | **Environment variables** — JWT secret, DB path, server port via env vars (12-factor) |
| How should migrations be loaded? | **Embedded in binary** — Go `embed` package, run on startup |
| How should seed data be handled? | **Seed script** — Separate Makefile target that inserts test data (admin user, restaurant, test recipes) |
| Password storage? | **bcrypt hash + salt** on the backend (user confirmed) |
| Auth complexity? | **Keep it simple for demo** — JWT, users, passwords, securely stored. No registration UI, no password reset. |

### Open Questions — Remaining

| Question | Status | Impact |
|----------|--------|--------|
| Should the JWT include restaurant IDs in the claims, or should we fetch them per-request? | Design says "JWT includes user ID and restaurant IDs" — assume embedded in claims | Low — affects middleware only |
| What is the JWT expiration policy? | Not specified — assume reasonable default (e.g., 24h for demo) | Low |
| Should `user_restaurants` have a composite PK or separate id column? | Design shows user_id + restaurant_id only — assume composite PK | Low |
| How should recipe search handle partial matches? | Design says "search by recipe name" — assume SQL LIKE with wildcards | Low |
| Should ingredients be editable inline on the recipe edit page, or on a separate page? | Not specified — assume inline for better UX | Medium — affects frontend design |

---

## Section 2: Milestones

Ordered by dependency. Each milestone produces a spec file (`XXX_nnn_spec.md`) and implementation file (`XXX_nnn_implement.md`) in `docs/plans/`.

- [x] 001: Database schema, migrations, connection layer, and seed data — all 6 tables created, embedded migrations run on startup, seed script populates test data
- [x] 002: Data access layer — Go models and repository methods for all tables with TDD coverage (users, restaurants, recipes, ingredients, food, user_restaurants)
- [x] 003: Authentication and authorization — `/login` endpoint with bcrypt verification, JWT generation with user/restaurant claims, auth middleware, RBAC middleware for role-based endpoint protection
- [x] 004: Frontend foundation — Templ templates generating, base HTML layout, Datastar script inclusion, restaurant dropdown selector wired to user's accessible restaurants
- [x] 005: Recipe CRUD and search — search endpoint with Datastar reactivity, recipe view page, create/edit forms with ingredient management, delete with role enforcement
