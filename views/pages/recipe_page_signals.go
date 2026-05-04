package pages

import "encoding/json"

type recipeFormPageSignals struct {
	SelectedRestaurant    string          `json:"selectedRestaurant"`
	Ingredients           []IngredientRow `json:"ingredients"`
	IngredientRowsVersion int             `json:"_ingredientRowsVersion"`
}

type recipeViewPageSignals struct {
	SelectedRestaurant string `json:"selectedRestaurant"`
}

func recipeFormSignals(restaurantID string, ingredients []IngredientRow) string {
	if ingredients == nil {
		ingredients = []IngredientRow{}
	}

	return mustPageSignalsJSON(recipeFormPageSignals{
		SelectedRestaurant:    restaurantID,
		Ingredients:           ingredients,
		IngredientRowsVersion: 0,
	})
}

func recipeViewSignals(restaurantID string) string {
	return mustPageSignalsJSON(recipeViewPageSignals{SelectedRestaurant: restaurantID})
}

func mustPageSignalsJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return string(b)
}
