package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
	"goFrameworkFirstTry/repository"
)

type UserHandler struct {
	Repo *repository.UserRepository
}

func NewUserHandler(repo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		Repo: repo,
	}
}

func (h *UserHandler) GetUsers(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(h.Repo.GetAll())
}

func (h *UserHandler) GetUserByID(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	id, err := strconv.Atoi(ps.ByName("id"))

	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	user, found := h.Repo.GetByID(id)

	if !found {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
