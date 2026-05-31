package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

// handleRBACLogin handles POST /auth/login with user+password from the users table.
// Falls back to the legacy ADMIN_PASSWORD single-user auth if no users exist.
func (h *Handler) handleRBACLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := security.ClientIPHashed(r)

	// ── Multi-user path ──────────────────────────────────────────────────────
	if h.users != nil && req.Email != "" {
		user, err := h.users.GetByEmail(r.Context(), req.Email)
		if err != nil || !user.IsActive() || !auth.VerifyPassword(req.Password, user.PasswordHash) {
			_ = h.auditStore.Append(r.Context(), audit.Event(
				audit.ActionAuthLoginFail, audit.ResourceAuth, "").
				By(audit.ActorAdmin).IP(ip).
				Details(fmt.Sprintf("email=%s", req.Email)).
				Severity(audit.SeverityWarning).Failed().Build())
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		token, err := h.sessions.Create(user.ID, user.Email, user.Role, ip)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		_ = h.auditStore.Append(r.Context(), audit.Event(
			audit.ActionAuthLogin, audit.ResourceAuth, user.ID.String()).
			By(audit.ActorAdmin).Actor(user.Email).IP(ip).UA(r.UserAgent()).Build())
		h.sessions.SetCookie(w, r, token)
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"user":   safeUser(user),
		})
		return
	}

	// ── Legacy single-password fallback ──────────────────────────────────────
	if h.webAuth != nil {
		token, err := h.webAuth.Login(req.Password, ip)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidCredentials) {
				_ = h.auditStore.Append(r.Context(), audit.Event(
					audit.ActionAuthLoginFail, audit.ResourceAuth, "").
					By(audit.ActorAdmin).IP(ip).
					Severity(audit.SeverityWarning).Failed().Build())
				writeError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		_ = h.auditStore.Append(r.Context(), audit.Event(
			audit.ActionAuthLogin, audit.ResourceAuth, "").
			By(audit.ActorAdmin).Actor("admin").IP(ip).UA(r.UserAgent()).Build())
		auth.SetCookie(w, r, token)
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
			"user":   map[string]string{"email": "admin", "role": "admin"},
		})
		return
	}

	writeError(w, http.StatusServiceUnavailable, "authentication not configured")
}

// handleRBACLogout handles POST /auth/logout.
func (h *Handler) handleRBACLogout(w http.ResponseWriter, r *http.Request) {
	token := auth.TokenFromRequest(r)
	if token != "" {
		if h.sessions != nil {
			h.sessions.Revoke(token)
			h.sessions.ClearCookie(w)
		} else if h.webAuth != nil {
			h.webAuth.Logout(token)
			auth.ClearCookie(w)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAuthMe handles GET /auth/me — returns the current user's info.
func (h *Handler) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"user_id":       session.UserID,
		"email":         session.UserEmail,
		"role":          session.Role,
	})
}

// ── User management (admin only) ──────────────────────────────────────────────

// handleListUsers handles GET /v1/users — admin only.
func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if h.users == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	users, err := h.users.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	safe := make([]map[string]any, len(users))
	for i, u := range users {
		safe[i] = safeUser(&u)
	}
	writeJSON(w, http.StatusOK, safe)
}

// handleCreateUser handles POST /v1/users — admin only.
func (h *Handler) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if h.users == nil {
		writeError(w, http.StatusServiceUnavailable, "user management not available")
		return
	}
	var req struct {
		Email       string    `json:"email"`
		Password    string    `json:"password"`
		Role        auth.Role `json:"role"`
		DisplayName string    `json:"display_name"`
	}
	if err := decode(r, &req); err != nil {
		handleDecodeError(w, err)
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if req.Role == "" {
		req.Role = auth.RoleViewer
	}
	if !req.Role.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid role — must be admin, operator, or viewer")
		return
	}
	if len(req.Password) < 10 {
		writeError(w, http.StatusBadRequest, "password must be at least 10 characters")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	user, err := h.users.Create(r.Context(), req.Email, string(hash), req.Role, req.DisplayName)
	if err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionCreate, audit.ResourceAuth, user.ID.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		Details(fmt.Sprintf("email=%s role=%s", user.Email, user.Role)).Build())
	writeJSON(w, http.StatusCreated, safeUser(user))
}

// handleUpdateUserRole handles PUT /v1/users/{id}/role — admin only.
func (h *Handler) handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if h.users == nil {
		writeError(w, http.StatusServiceUnavailable, "user management not available")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	var req struct{ Role auth.Role }
	if err := decode(r, &req); err != nil {
		handleDecodeError(w, err)
		return
	}
	if !req.Role.IsValid() {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}
	// Prevent demoting the last admin
	if req.Role != auth.RoleAdmin {
		count, err := h.users.CountByRole(r.Context(), auth.RoleAdmin)
		if err == nil && count <= 1 {
			target, _ := h.users.GetByID(r.Context(), id)
			if target != nil && target.Role == auth.RoleAdmin {
				writeError(w, http.StatusConflict, "cannot demote the last admin user")
				return
			}
		}
	}
	if err := h.users.UpdateRole(r.Context(), id, req.Role); err != nil {
		writeError(w, httpStatusForError(err), "update failed")
		return
	}
	// Invalidate all sessions for this user so new role takes effect immediately
	if h.sessions != nil {
		h.sessions.RevokeByUser(id)
	}
	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionUpdate, audit.ResourceAuth, id.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		Details(fmt.Sprintf("role changed to %s", req.Role)).
		Severity(audit.SeverityWarning).Build())
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleDeleteUser handles DELETE /v1/users/{id} — admin only.
func (h *Handler) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if h.users == nil {
		writeError(w, http.StatusServiceUnavailable, "user management not available")
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}
	// Prevent deleting the last admin
	count, err := h.users.CountByRole(r.Context(), auth.RoleAdmin)
	if err == nil && count <= 1 {
		target, _ := h.users.GetByID(r.Context(), id)
		if target != nil && target.Role == auth.RoleAdmin {
			writeError(w, http.StatusConflict, "cannot delete the last admin user")
			return
		}
	}
	if h.sessions != nil {
		h.sessions.RevokeByUser(id)
	}
	if err := h.users.Delete(r.Context(), id); err != nil {
		writeError(w, httpStatusForError(err), "delete failed")
		return
	}
	_ = h.auditStore.Append(r.Context(), audit.Event(
		audit.ActionDelete, audit.ResourceAuth, id.String()).
		By(audit.ActorAdmin).
		IP(security.ClientIPHashed(r)).
		Severity(audit.SeverityWarning).Build())
	w.WriteHeader(http.StatusNoContent)
}

// safeUser strips the password hash before sending to the client.
func safeUser(u *auth.User) map[string]any {
	m := map[string]any{
		"id":           u.ID,
		"email":        u.Email,
		"role":         u.Role,
		"display_name": u.DisplayName,
		"created_at":   u.CreatedAt,
		"active":       u.IsActive(),
	}
	if u.DisabledAt != nil {
		m["disabled_at"] = u.DisabledAt
	}
	return m
}
