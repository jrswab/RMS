package handler

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/starfederation/datastar-go/datastar"
	"lab37/internal/auth"
	"lab37/internal/store"
	"lab37/views/layouts"
	"lab37/views/pages"
)

type IngredientSignal struct {
	FoodName string `json:"foodName"`
	Quantity string `json:"quantity"`
	Unit     string `json:"unit"`
}

type RecipeFormSignals struct {
	Name         string             `json:"name"`
	Yield        string             `json:"yield"`
	Instructions string             `json:"instructions"`
	Ingredients  []IngredientSignal `json:"ingredients"`
	RestaurantID string             `json:"selectedRestaurant"`
}

type RecipeCreateHandler struct {
	DB store.DBTX
}

type validatedIngredient struct {
	FoodName string
	Quantity float64
	Unit     string
}

func (h *RecipeCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.serveGet(w, r)
	case http.MethodPost:
		h.servePost(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *RecipeCreateHandler) serveGet(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	restaurantID := strings.TrimSpace(r.URL.Query().Get("restaurant"))
	if restaurantID == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if !slices.Contains(claims.RestaurantIDs, restaurantID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	user, restaurants, ok := h.loadRecipeFormPageData(w, claims)
	if !ok {
		return
	}

	_ = layouts.Base("Create Recipe", pages.RecipeForm(user.Username, restaurants, restaurantID, store.Recipe{}, []pages.IngredientRow{{}}, map[string]string{}, false)).Render(r.Context(), w)
}

func (h *RecipeCreateHandler) servePost(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	var signals RecipeFormSignals
	if err := datastar.ReadSignals(r, &signals); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	signals.RestaurantID = strings.TrimSpace(signals.RestaurantID)
	if signals.RestaurantID != "" && !slices.Contains(claims.RestaurantIDs, signals.RestaurantID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	recipe, ingredientRows, validatedIngredients, formErrors := validateRecipeCreateSignals(signals)
	if len(formErrors) > 0 {
		h.renderRecipeFormPatch(w, r, claims, recipe, ingredientRows, formErrors)
		return
	}

	database, ok := h.DB.(*sql.DB)
	if !ok {
		log.Printf("error starting transaction: DB type %T does not support Begin", h.DB)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	tx, err := database.Begin()
	if err != nil {
		log.Printf("error beginning transaction: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	foodIDs, err := resolveFoodIDs(tx, validatedIngredients)
	if err != nil {
		log.Printf("error resolving food ids: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	now := time.Now().Unix()
	recipe.ID = uuid.New().String()
	recipe.CreatedAt = now
	recipe.UpdatedAt = now

	if err := store.CreateRecipe(tx, &recipe); err != nil {
		log.Printf("error creating recipe: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ingredients := make([]store.Ingredient, 0, len(validatedIngredients))
	for i, ingredient := range validatedIngredients {
		ingredients = append(ingredients, store.Ingredient{
			ID:        uuid.New().String(),
			RecipeID:  recipe.ID,
			FoodID:    foodIDs[ingredient.FoodName],
			Quantity:  ingredient.Quantity,
			Unit:      ingredient.Unit,
			SortOrder: int64(i + 1),
		})
	}

	if err := store.ReplaceIngredients(tx, recipe.ID, ingredients); err != nil {
		log.Printf("error replacing ingredients: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("error committing transaction: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.Redirect("/recipe/" + recipe.ID)
}

func (h *RecipeCreateHandler) loadRecipeFormPageData(w http.ResponseWriter, claims *auth.Claims) (*store.User, []store.Restaurant, bool) {
	user, err := store.GetUserByID(h.DB, claims.UserID)
	if err != nil {
		log.Printf("error getting user: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, nil, false
	}

	restaurants, err := store.ListRestaurantsByUserID(h.DB, claims.UserID)
	if err != nil {
		log.Printf("error listing restaurants: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, nil, false
	}

	return user, restaurants, true
}

func (h *RecipeCreateHandler) renderRecipeFormPatch(w http.ResponseWriter, r *http.Request, claims *auth.Claims, recipe store.Recipe, ingredientRows []pages.IngredientRow, formErrors map[string]string) {
	user, restaurants, ok := h.loadRecipeFormPageData(w, claims)
	if !ok {
		return
	}

	var buf bytes.Buffer
	_ = pages.RecipeForm(user.Username, restaurants, recipe.RestaurantID, recipe, ingredientRows, formErrors, false).Render(r.Context(), &buf)

	sse := datastar.NewSSE(w, r)
	sse.PatchElements(buf.String(), datastar.WithSelectorID("recipe-form-page"))
}

func validateRecipeCreateSignals(signals RecipeFormSignals) (store.Recipe, []pages.IngredientRow, []validatedIngredient, map[string]string) {
	recipe := store.Recipe{
		Name:         strings.TrimSpace(signals.Name),
		RestaurantID: strings.TrimSpace(signals.RestaurantID),
		Instructions: strings.TrimSpace(signals.Instructions),
	}

	ingredientRows := make([]pages.IngredientRow, 0, len(signals.Ingredients))
	validatedIngredients := make([]validatedIngredient, 0, len(signals.Ingredients))
	formErrors := make(map[string]string)

	if recipe.Name == "" {
		formErrors["name"] = "Name is required."
	}
	if recipe.RestaurantID == "" {
		formErrors["restaurant"] = "Restaurant is required."
	}

	yield, err := strconv.ParseInt(strings.TrimSpace(signals.Yield), 10, 64)
	if err != nil {
		formErrors["yield"] = "Yield must be a valid integer."
	} else {
		recipe.Yield = yield
	}

	for i, signal := range signals.Ingredients {
		foodName := strings.TrimSpace(signal.FoodName)
		quantityText := strings.TrimSpace(signal.Quantity)
		unit := strings.TrimSpace(signal.Unit)

		ingredientRows = append(ingredientRows, pages.IngredientRow{
			FoodName: foodName,
			Quantity: quantityText,
			Unit:     unit,
		})

		if foodName == "" && quantityText == "" && unit == "" {
			continue
		}

		if foodName == "" {
			if _, ok := formErrors["ingredients"]; !ok {
				formErrors["ingredients"] = fmt.Sprintf("Ingredient %d food name is required.", i+1)
			}
			continue
		}

		quantity, err := strconv.ParseFloat(quantityText, 64)
		if err != nil {
			if _, ok := formErrors["ingredients"]; !ok {
				formErrors["ingredients"] = fmt.Sprintf("Ingredient %d quantity must be a valid number.", i+1)
			}
			continue
		}

		if unit == "" {
			if _, ok := formErrors["ingredients"]; !ok {
				formErrors["ingredients"] = fmt.Sprintf("Ingredient %d unit is required.", i+1)
			}
			continue
		}

		validatedIngredients = append(validatedIngredients, validatedIngredient{
			FoodName: foodName,
			Quantity: quantity,
			Unit:     unit,
		})
	}

	return recipe, ingredientRows, validatedIngredients, formErrors
}

func resolveFoodIDs(tx *sql.Tx, ingredients []validatedIngredient) (map[string]string, error) {
	foodIDs := make(map[string]string, len(ingredients))

	for _, ingredient := range ingredients {
		if _, ok := foodIDs[ingredient.FoodName]; ok {
			continue
		}

		food, err := store.GetFoodByName(tx, ingredient.FoodName)
		if err != nil {
			if !errors.Is(err, store.ErrNotFound) {
				return nil, fmt.Errorf("lookup food %q: %w", ingredient.FoodName, err)
			}

			food = &store.Food{ID: uuid.New().String(), Name: ingredient.FoodName}
			if err := store.CreateFood(tx, food); err != nil {
				return nil, fmt.Errorf("create food %q: %w", ingredient.FoodName, err)
			}
		}

		foodIDs[ingredient.FoodName] = food.ID
	}

	return foodIDs, nil
}
