CREATE TABLE restaurants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    role TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE food (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE user_restaurants (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    restaurant_id TEXT NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    UNIQUE(user_id, restaurant_id)
);

CREATE TABLE recipes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    restaurant_id TEXT NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    instructions TEXT NOT NULL,
    yield INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE ingredients (
    id TEXT PRIMARY KEY,
    recipe_id TEXT NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    food_id TEXT NOT NULL REFERENCES food(id) ON DELETE RESTRICT,
    quantity REAL NOT NULL,
    unit TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    UNIQUE(recipe_id, sort_order)
);

CREATE INDEX idx_recipes_restaurant_id ON recipes(restaurant_id);
CREATE INDEX idx_recipes_name ON recipes(name);
CREATE INDEX idx_ingredients_recipe_id ON ingredients(recipe_id);
