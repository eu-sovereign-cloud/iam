package service

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

func (s *Service) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	u, err := s.CreateUser.Do(r.Context(), req.Subject, req.DisplayName, req.Admin)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, userToResponse(u))
}

func (s *Service) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.ListUsers.Do(r.Context())
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

func (s *Service) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	var req patchUserRequest
	if err := decodeJSON(r, &req); err != nil || req.Admin == nil {
		writeError(w, http.StatusBadRequest, "body must set \"admin\"")
		return
	}
	u, err := s.SetUserAdmin.Do(r.Context(), r.PathValue("subject"), *req.Admin)
	if err != nil {
		writeError(w, statusFor(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, userToResponse(u))
}

func (s *Service) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if err := s.DeleteUser.Do(r.Context(), r.PathValue("subject")); err != nil {
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
