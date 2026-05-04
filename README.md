# Recipe Manager — Lab 37

A recipe management system for restaurant kitchen staff. Create, edit, search, and display recipes per restaurant. Server-rendered HTML with zero frontend build step, backed by SQLite.

## Tech Stack

- **Go** with [Chi](https://github.com/go-chi/chi) router
- [Templ](https://templ.guide/) for type-safe HTML templates
- [Datastar](https://data-star.dev/) for frontend reactivity (state stays on the backend)
- **SQLite** via [go-sqlite3](https://github.com/mattn/go-sqlite3)
- JWT authentication

## Prerequisites

- **Go 1.24+**
- **GCC** (required by the CGo SQLite driver — `gcc` must be on your `PATH`)
- [Templ CLI](https://templ.guide/quick-start/installation) (`go install github.com/a-h/templ/cmd/templ@latest`)

## Quick Start

```bash
# 1. Clone and enter the repo
git clone https://github.com/jrswab/lab37.git
cd lab37

# 2. Install Go dependencies
go mod tidy

# 3. Generate templ files
templ generate

# 4. Seed the database with sample data
make seed

# 5. Start the server (requires JWT_SECRET)
JWT_SECRET=dev-secret make run

# 6. Open http://localhost:8080 in your browser
```

7. Use one of these demo logins to test each role:

| Username  | Password      | Role    |
|-----------|---------------|---------|
| `admin`   | `password123` | admin   |
| `manager` | `password123` | manager |
| `staff`   | `password123` | staff   |

## Seed Users

After running `make seed`, you can log in with any of these accounts (password for all: `password123`):

| Username | Role    | Restaurants                        |
|----------|---------|------------------------------------|
| admin    | admin   | The Rusty Spoon, Copper Kettle     |
| manager  | manager | The Rusty Spoon                    |
| staff    | staff   | The Rusty Spoon                    |

## Environment Variables

| Variable     | Required | Default | Description                        |
|--------------|----------|---------|------------------------------------|
| `JWT_SECRET` | Yes      | —       | Secret key for signing JWT tokens  |
| `PORT`       | No       | `8080`  | Port the HTTP server listens on    |

## Makefile Targets

| Target      | Description                          |
|-------------|--------------------------------------|
| `make run`  | Generate templ files and start server|
| `make seed` | Seed the database with sample data   |
| `make test` | Run all tests                        |
| `make tidy` | Run `go mod tidy`                    |
| `make clean`| Delete the SQLite database files     |

## Project Structure

```
.
├── cmd/
│   ├── server/       # HTTP server entry point
│   └── seed/         # Database seeder for local dev
├── docs/plans/       # Design docs and implementation specs
├── internal/
│   ├── auth/         # JWT middleware and helpers
│   ├── db/           # Database connection and migrations runner
│   ├── handler/      # HTTP handlers (login, search, recipe CRUD)
│   └── store/        # Data access layer
├── migrations/       # SQL migration files (embedded)
├── static/css/       # Stylesheets
└── views/
    ├── components/   # Reusable templ components
    ├── layouts/      # Page layout templates
    └── pages/        # Page-level templates
```

## License

Private — Lab 37 take-home project.
