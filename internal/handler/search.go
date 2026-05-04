package handler

import (
	"bytes"
	"log"
	"net/http"
	"slices"

	"github.com/starfederation/datastar-go/datastar"
	"lab37/internal/auth"
	"lab37/internal/store"
	"lab37/views/pages"
)

type SearchSignals struct {
	SearchQuery        string `json:"searchQuery"`
	SelectedRestaurant string `json:"selectedRestaurant"`
}

type SearchHandler struct {
	DB store.DBTX
}

func (h *SearchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	claims, err := auth.ClaimsFromContext(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	signals := &SearchSignals{}
	if err := datastar.ReadSignals(r, signals); err != nil {
		http.Error(w, "invalid signals", http.StatusBadRequest)
		return
	}

	if signals.SelectedRestaurant == "" {
		var buf bytes.Buffer
		_ = pages.SearchResults(nil, "").Render(r.Context(), &buf)

		sse := datastar.NewSSE(w, r)
		sse.PatchElements(buf.String(), datastar.WithSelectorID("search-results"))
		return
	}

	if !slices.Contains(claims.RestaurantIDs, signals.SelectedRestaurant) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	recipes, err := store.SearchRecipes(h.DB, signals.SelectedRestaurant, signals.SearchQuery)
	if err != nil {
		log.Printf("error searching recipes: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	_ = pages.SearchResults(recipes, signals.SelectedRestaurant).Render(r.Context(), &buf)

	sse := datastar.NewSSE(w, r)
	sse.PatchElements(buf.String(), datastar.WithSelectorID("search-results"))
}
