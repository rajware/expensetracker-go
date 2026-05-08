package rest

import (
	"net/http"
)

func (h *handler) handleAddCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	userID := userIDFromContext(r.Context())
	category, err := h.categories.Add(r.Context(), userID, body.Name)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSONWithStatus(w, http.StatusCreated, category)
}

func (h *handler) handleQueryCategories(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")

	categories, err := h.categories.Query(r.Context(), prefix)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, categories)
}

func (h *handler) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}

	userID := userIDFromContext(r.Context())
	category, err := h.categories.Update(r.Context(), userID, id, body.Name)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, category)
}

func (h *handler) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := userIDFromContext(r.Context())

	if err := h.categories.Delete(r.Context(), userID, id); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
