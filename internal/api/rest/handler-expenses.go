package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/rajware/expensetracker-go/internal/domain"
)

// --- Expense handlers ---

func (h *handler) handleAddExpense(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OccurredAt  time.Time `json:"occurred_at"`
		Description string    `json:"description"`
		Amount      float64   `json:"amount"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	userID := userIDFromContext(r.Context())
	expense, err := h.expenses.Add(r.Context(), userID, body.OccurredAt, body.Description, body.Amount)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSONWithStatus(w, http.StatusCreated, domain.NewExpenseView(*expense))
}

func (h *handler) handleQueryExpenses(w http.ResponseWriter, r *http.Request) {
	q, ok := parseExpenseQuery(w, r)
	if !ok {
		return
	}
	userID := userIDFromContext(r.Context())
	result, err := h.expenses.Query(r.Context(), userID, q)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, result)
}

func (h *handler) handleGetExpense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := userIDFromContext(r.Context())
	view, err := h.expenses.QueryByID(r.Context(), userID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, view)
}

func (h *handler) handleUpdateExpense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Description string    `json:"description"`
		OccurredAt  time.Time `json:"occurred_at"`
		Amount      float64   `json:"amount"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	userID := userIDFromContext(r.Context())
	expense, err := h.expenses.Update(r.Context(), userID, id, body.Description, body.OccurredAt, body.Amount)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, domain.NewExpenseView(*expense))
}

func (h *handler) handleDeleteExpense(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	userID := userIDFromContext(r.Context())
	if err := h.expenses.Delete(r.Context(), userID, id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseExpenseQuery(w http.ResponseWriter, r *http.Request) (domain.ExpenseQuery, bool) {
	var q domain.ExpenseQuery
	params := r.URL.Query()

	if s := params.Get("from"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			http.Error(w, "invalid 'from' date", http.StatusBadRequest)
			return q, false
		}
		q.From = &t
	}
	if s := params.Get("to"); s != "" {
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			http.Error(w, "invalid 'to' date", http.StatusBadRequest)
			return q, false
		}
		q.To = &t
	}
	switch params.Get("sort_by") {
	case "description":
		q.SortBy = domain.SortByDescription
	case "amount":
		q.SortBy = domain.SortByAmount
	default:
		q.SortBy = domain.SortByDate
	}
	if params.Get("sort_desc") == "true" {
		q.SortDesc = true
	}
	if s := params.Get("page"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			http.Error(w, "invalid 'page'", http.StatusBadRequest)
			return q, false
		}
		q.Page = n
	}
	if s := params.Get("page_size"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			http.Error(w, "invalid 'page_size'", http.StatusBadRequest)
			return q, false
		}
		q.PageSize = n
	}
	return q, true
}
