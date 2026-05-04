package pages

type IngredientWithFood struct {
	FoodName  string
	Quantity  float64
	Unit      string
	SortOrder int64
}

type IngredientRow struct {
	FoodName string `json:"foodName"`
	Quantity string `json:"quantity"`
	Unit     string `json:"unit"`
}
