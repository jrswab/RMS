package handler

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"lab37/internal/auth"
	"lab37/internal/store"
	"lab37/views/pages"
)

type RecipeDeleteHandler struct {
	DB store.DBTX
}

func (h *RecipeDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodDelete:
		h.serveDelete(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/delete-confirm"):
		h.serveDeleteConfirm(w, r)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/delete-cancel"):
		h.serveDeleteCancel(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *RecipeDeleteHandler) serveDeleteConfirm(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	recipe, ok := h.loadAuthorizedRecipe(w, chi.URLParam(r, "id"), claims)
	if !ok {
		return
	}

	var buf bytes.Buffer
	_ = pages.RecipeDeleteDialog(*recipe).Render(r.Context(), &buf)

	sse := datastar.NewSSE(w, r)
	sse.PatchElements(buf.String(), datastar.WithSelectorID("delete-confirmation"), datastar.WithModeInner())
}

func (h *RecipeDeleteHandler) serveDeleteCancel(w http.ResponseWriter, r *http.Request) {
	_, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.PatchElements(`<div id="delete-confirmation"></div>`, datastar.WithSelectorID("delete-confirmation"))
}

func (h *RecipeDeleteHandler) serveDelete(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	recipe, ok := h.loadAuthorizedRecipe(w, chi.URLParam(r, "id"), claims)
	if !ok {
		return
	}

	if err := store.DeleteRecipe(h.DB, recipe.ID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, "recipe not found", http.StatusNotFound)
			return
		}

		log.Printf("error deleting recipe: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	sse := datastar.NewSSE(w, r)
	sse.Redirect("/")
}

func (h *RecipeDeleteHandler) loadAuthorizedRecipe(w http.ResponseWriter, recipeID string, claims *auth.Claims) (*store.Recipe, bool) {
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
