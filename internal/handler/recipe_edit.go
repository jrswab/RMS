package handler

import (
	"bytes"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/starfederation/datastar-go/datastar"
	"lab37/internal/auth"
	"lab37/internal/store"
	"lab37/views/layouts"
	"lab37/views/pages"
)

type RecipeEditHandler struct {
	DB store.DBTX
}

func (h *RecipeEditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.serveGet(w, r)
	case http.MethodPatch:
		h.servePatch(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *RecipeEditHandler) serveGet(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	recipe, ok := h.loadAuthorizedRecipe(w, chi.URLParam(r, "id"), claims)
	if !ok {
		return
	}

	ingredientRows, ok := h.loadIngredientRows(w, recipe.ID)
	if !ok {
		return
	}

	if !h.renderRecipeFormPage(w, r, claims, *recipe, ingredientRows, map[string]string{}) {
		return
	}
}

func (h *RecipeEditHandler) servePatch(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	recipe, ok := h.loadAuthorizedRecipe(w, chi.URLParam(r, "id"), claims)
	if !ok {
		return
	}

	var signals RecipeFormSignals
	if err := datastar.ReadSignals(r, &signals); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	signals.RestaurantID = recipe.RestaurantID

	updatedRecipe, ingredientRows, validatedIngredients, formErrors := validateRecipeCreateSignals(signals)
	updatedRecipe.ID = recipe.ID
	updatedRecipe.RestaurantID = recipe.RestaurantID

	if len(formErrors) > 0 {
		h.renderRecipeFormPatch(w, r, claims, updatedRecipe, ingredientRows, formErrors)
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

	recipe.Name = updatedRecipe.Name
	recipe.Instructions = updatedRecipe.Instructions
	recipe.Yield = updatedRecipe.Yield
	recipe.UpdatedAt = time.Now().Unix()

	if err := store.UpdateRecipe(tx, recipe); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return
		}

		log.Printf("error updating recipe: %v", err)
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

func (h *RecipeEditHandler) loadAuthorizedRecipe(w http.ResponseWriter, recipeID string, claims *auth.Claims) (*store.Recipe, bool) {
	recipe, err := store.GetRecipeByID(h.DB, recipeID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return nil, false
		}

		log.Printf("error getting recipe: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, false
	}

	if !slices.Contains(claims.RestaurantIDs, recipe.RestaurantID) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}

	if claims.Role != "admin" && claims.Role != "manager" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}

	return recipe, true
}

func (h *RecipeEditHandler) loadIngredientRows(w http.ResponseWriter, recipeID string) ([]pages.IngredientRow, bool) {
	ingredients, err := store.ListIngredientsByRecipeID(h.DB, recipeID)
	if err != nil {
		log.Printf("error listing ingredients: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return nil, false
	}

	ingredientRows := make([]pages.IngredientRow, 0, len(ingredients))
	for _, ingredient := range ingredients {
		food, err := store.GetFoodByID(h.DB, ingredient.FoodID)
		if err != nil {
			log.Printf("error getting food by id: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return nil, false
		}

		ingredientRows = append(ingredientRows, pages.IngredientRow{
			FoodName: food.Name,
			Quantity: strconv.FormatFloat(ingredient.Quantity, 'f', -1, 64),
			Unit:     ingredient.Unit,
		})
	}

	return ingredientRows, true
}

func (h *RecipeEditHandler) renderRecipeFormPage(w http.ResponseWriter, r *http.Request, claims *auth.Claims, recipe store.Recipe, ingredientRows []pages.IngredientRow, formErrors map[string]string) bool {
	pageDataLoader := &RecipeCreateHandler{DB: h.DB}
	user, restaurants, ok := pageDataLoader.loadRecipeFormPageData(w, claims)
	if !ok {
		return false
	}

	_ = layouts.Base("Edit Recipe", pages.RecipeForm(user.Username, restaurants, recipe.RestaurantID, recipe, ingredientRows, formErrors, true)).Render(r.Context(), w)
	return true
}

func (h *RecipeEditHandler) renderRecipeFormPatch(w http.ResponseWriter, r *http.Request, claims *auth.Claims, recipe store.Recipe, ingredientRows []pages.IngredientRow, formErrors map[string]string) {
	pageDataLoader := &RecipeCreateHandler{DB: h.DB}
	user, restaurants, ok := pageDataLoader.loadRecipeFormPageData(w, claims)
	if !ok {
		return
	}

	var buf bytes.Buffer
	_ = pages.RecipeForm(user.Username, restaurants, recipe.RestaurantID, recipe, ingredientRows, formErrors, true).Render(r.Context(), &buf)

	sse := datastar.NewSSE(w, r)
	sse.PatchElements(buf.String(), datastar.WithSelectorID("recipe-form-page"))
}
