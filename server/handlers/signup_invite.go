package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/pkg"
	"github.com/akinalp/mqvi/pkg/ctxkeys"
	"github.com/akinalp/mqvi/services"
)

type SignupInviteHandler struct {
	service services.SignupInviteService
}

func NewSignupInviteHandler(service services.SignupInviteService) *SignupInviteHandler {
	return &SignupInviteHandler{service: service}
}

func (h *SignupInviteHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(ctxkeys.User).(*models.User)
	if !ok || user == nil {
		pkg.ErrorWithMessage(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	invites, err := h.service.List(r.Context(), user.ID)
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, http.StatusOK, invites)
}

func (h *SignupInviteHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(ctxkeys.User).(*models.User)
	if !ok || user == nil {
		pkg.ErrorWithMessage(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	var req struct {
		ExpiryDays int `json:"expiry_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.ErrorWithMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}

	invite, err := h.service.Create(r.Context(), user.ID, req.ExpiryDays)
	if err != nil {
		if errors.Is(err, services.ErrSignupInviteQuota) {
			pkg.ErrorWithMessage(w, http.StatusConflict, "signup invite creation quota reached")
			return
		}
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, http.StatusCreated, invite)
}

func (h *SignupInviteHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	user, ok := r.Context().Value(ctxkeys.User).(*models.User)
	if !ok || user == nil {
		pkg.ErrorWithMessage(w, http.StatusUnauthorized, "user not found in context")
		return
	}

	inviteID := r.PathValue("inviteId")
	if inviteID == "" {
		pkg.ErrorWithMessage(w, http.StatusBadRequest, "invite id is required")
		return
	}

	if err := h.service.Revoke(r.Context(), user.ID, inviteID); err != nil {
		if errors.Is(err, services.ErrSignupInviteInvalid) {
			pkg.ErrorWithMessage(w, http.StatusNotFound, "signup invite not found or unavailable")
			return
		}
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, http.StatusOK, map[string]string{"message": "signup invite revoked"})
}

func (h *SignupInviteHandler) AdminListRelationships(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.AdminListRelationships(r.Context())
	if err != nil {
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, http.StatusOK, items)
}

func (h *SignupInviteHandler) AdminSetLimit(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("id")
	if userID == "" {
		pkg.ErrorWithMessage(w, http.StatusBadRequest, "user id is required")
		return
	}
	var req struct {
		Limit int `json:"invite_limit"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		pkg.ErrorWithMessage(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.service.AdminSetLimit(r.Context(), userID, req.Limit); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			pkg.ErrorWithMessage(w, http.StatusNotFound, "user not found")
			return
		}
		pkg.Error(w, err)
		return
	}
	pkg.JSON(w, http.StatusOK, map[string]interface{}{"user_id": userID, "invite_limit": req.Limit})
}
