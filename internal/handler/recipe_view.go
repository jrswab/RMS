package handler

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"lab37/internal/auth"
	"lab37/internal/store"
	"lab37/views/layouts"
	"lab37/views/pages"
)

type RecipeViewHandler struct {
	DB store.DBTX
}

func (h *RecipeViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	id := chi.URLParam(r, "id")

	recipe, err := store.GetRecipeByID(h.DB, id)
	if err != nil {
		if err == store.ErrNotFound {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return
		}

		log.Printf("error getting recipe: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	hasAccess := false
	for _, restaurantID := range claims.RestaurantIDs {
		if restaurantID == recipe.RestaurantID {
			hasAccess = true
			break
		}
	}
	if !hasAccess {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	ingredients, err := store.ListIngredientsByRecipeID(h.DB, id)
	if err != nil {
		log.Printf("error listing ingredients: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ingredientsWithFood := make([]pages.IngredientWithFood, 0, len(ingredients))
	for _, ingredient := range ingredients {
		food, err := store.GetFoodByID(h.DB, ingredient.FoodID)
		if err != nil {
			log.Printf("error getting food by id: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		ingredientsWithFood = append(ingredientsWithFood, pages.IngredientWithFood{
			FoodName:  food.Name,
			Quantity:  ingredient.Quantity,
			Unit:      ingredient.Unit,
			SortOrder: ingredient.SortOrder,
		})
	}

	canEdit := claims.Role == "admin" || claims.Role == "manager"
	canDelete := canEdit

	user, err := store.GetUserByID(h.DB, claims.UserID)
	if err != nil {
		log.Printf("error getting user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	restaurants, err := store.ListRestaurantsByUserID(h.DB, claims.UserID)
	if err != nil {
		log.Printf("error listing restaurants: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	_ = layouts.Base(recipe.Name, pages.RecipeView(user.Username, restaurants, *recipe, ingredientsWithFood, canEdit, canDelete)).Render(r.Context(), w)
}
