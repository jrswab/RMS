package store

type User struct {
	ID        string
	Username  string
	Password  string
	Role      string
	CreatedAt int64
	UpdatedAt int64
}

type Restaurant struct {
	ID   string
	Name string
}

type Food struct {
	ID   string
	Name string
}

type UserRestaurant struct {
	UserID       string
	RestaurantID string
}

type Recipe struct {
	ID           string
	Name         string
	RestaurantID string
	Instructions string
	Yield        int64
	CreatedAt    int64
	UpdatedAt    int64
}

type Ingredient struct {
	ID        string
	RecipeID  string
	FoodID    string
	Quantity  float64
	Unit      string
	SortOrder int64
}
