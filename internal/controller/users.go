package controller

import (
	"net/http"

	"github.com/eu-sovereign-cloud/iam/internal/model"
)

type createUserRequest struct {
	Subject     string `json:"subject"`
	DisplayName string `json:"displayName"`
	Admin       bool   `json:"admin"`
}

type patchUserRequest struct {
	Admin *bool `json:"admin"`
}

type userResponse struct {
	Subject     string `json:"subject"`
	DisplayName string `json:"displayName"`
	Admin       bool   `json:"admin"`
	CreatedAt   string `json:"createdAt"`
}

func (c *Controller) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := c.Users.Create(r.Context(), req.Subject, req.DisplayName, req.Admin)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, userToResponse(u))
}

func (c *Controller) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := c.Users.List(r.Context())
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	out := make([]userResponse, 0, len(users))
	for _, u := range users {
		out = append(out, userToResponse(u))
	}
	writeJSON(w, http.StatusOK, out)
}

func (c *Controller) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	var req patchUserRequest
	if err := decodeJSON(r, &req); err != nil || req.Admin == nil {
		writeError(w, http.StatusBadRequest, "body must set \"admin\"")
		return
	}
	u, err := c.Users.SetAdmin(r.Context(), r.PathValue("subject"), *req.Admin)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, userToResponse(u))
}

func (c *Controller) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if err := c.Users.Delete(r.Context(), r.PathValue("subject")); err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func userToResponse(u model.User) userResponse {
	return userResponse{
		Subject:     u.Subject,
		DisplayName: u.DisplayName,
		Admin:       u.Admin,
		CreatedAt:   u.CreatedAt.Format(timeFormat),
	}
}
