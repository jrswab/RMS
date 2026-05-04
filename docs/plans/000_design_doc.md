# Recipe Management System

Goal: design a web system to create, edit, delete, and display recipes for restaurants.

## UX
- Users must be able to log in.
- Search must work on a per restaurant basis.
- Dropdowns selector for restaurant (based on the user's permissons)
- Display a recipe for a user to read; including instructions and ingredients.
- The user must be able to edit a recipe they have access to.
- The admin user must be allowed to delete a recipe they have access to.

## Tech Stack
- Go & Templ
- Datastar
- SQL Lite

Using Go allows us to have bot the front end and the backend for the system in one repostiory. Templ is a templating package for Go with React like variable substitutions which allows for reusability across the site.

Datastar provides frontend reactivity and keeps all state on the backend; the frontend simply renders.

Using SQL lite for it's relational database since the expected userbase is small. This does allow for scaling into Postresql in the future if demand requires it.

## Database

### Tables

#### Recipes
- id: UUID
- name: text
- restaurant_id: UUID (from the restaurants table)
- instructions: text
- yield: int
- created_at: unix timestamp
- updated_at: unix timestamp

#### Restaurants
- id: UUID
- name: text

#### Users
- id: UUID
- username: text
- password: hashed text
- role: text (`admin`, `manager`, `staff`)
- created_at: unix timestamp
- updated_at: unix timestamp

#### User Restaurants
This table is for storing the restaurants a user has access to. Likely to be one but can also be multiple.

- user_id: UUID
- restaurant_id: UUID

#### Food
This table is for food IDs to be used within the ingredients table but allows for expansion into a global food catalog for building recipes in the future.

- id: UUID
- name: text

#### Ingredients
List of ingredients, their food id, and what recipe it belongs to. Multipl recipes can share the same food_id while having various quantities, units, and in different prep orders.

- id: UUID
- recipe_id: UUID (from recipe table and tied to a restaurant)
- food_id: UUID (from food tabel)
- quantity: REAL
- unit: text
- sort_order: int (Controls display order of ingredients within a recipe.)


## Endpoints
- `/login`
    - HTTP POST for user name and password.
    - This creates a JWT that includes their user ID and restaurant IDs.
    - All routes except /login require a valid JWT in the Authorization header.
- `/search?q={text}`
    - HTTP POST performs the recipe search by recipe name for a specific restaurant.
    - Query text must be escaped before using on the backend.
- `/recipe/{id}`
    - HTTP GET to fetch the recipe data (for loading in the ui or supplying the data)
    - HTTP DELETE to remove the recipe (if user role is `admin` or `manager`). The frontend hides the delete button if the user is not in the premmitted roles and direct API requests are rejected.
    - HTTP PATCH to edit a recipe. Text must be escaped before storing. (if user role is `admin` or `manager`)
- `/recipe/new`
    - HTTP POST to create a new recipe
    - Text must be escaped before storing.
