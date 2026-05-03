package rest

import "net/http"

// --- User handlers ---

func (h *handler) handleSignUp(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	_, err := h.users.SignUp(r.Context(), body.Username, body.DisplayName, body.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *handler) handleSignIn(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	user, err := h.users.SignIn(r.Context(), body.Username, body.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	token, err := h.auth.IssueToken(user.ID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.transport.SetToken(w, token)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) handleGetMe(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	view, err := h.users.QueryByID(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, view)
}

func (h *handler) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	var body struct {
		DisplayName string `json:"display_name"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	userID := userIDFromContext(r.Context())
	view, err := h.users.UpdateDisplayName(r.Context(), userID, body.DisplayName)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, view)
}

func (h *handler) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if err := h.users.CloseAccountByID(r.Context(), userID); err != nil {
		writeError(w, err)
		return
	}
	h.transport.ClearToken(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) handleKeepalive(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	token, err := h.auth.IssueToken(userID)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h.transport.SetToken(w, token)
	w.WriteHeader(http.StatusNoContent)
}

func (h *handler) handleSignOut(w http.ResponseWriter, r *http.Request) {
	h.transport.ClearToken(w)
	w.WriteHeader(http.StatusNoContent)
}
