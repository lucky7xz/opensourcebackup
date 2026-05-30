package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func (h *Handler) listRepositories(w http.ResponseWriter, r *http.Request) {
	repos, err := h.repositories.List(r.Context())
	if err != nil {
		h.log.Error("list repositories", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if repos == nil {
		repos = []catalog.BackupRepository{}
	}
	writeJSON(w, http.StatusOK, repos)
}

func (h *Handler) createRepository(w http.ResponseWriter, r *http.Request) {
	var repo catalog.BackupRepository
	if err := decode(r, &repo); err != nil {
		handleDecodeError(w, err)
		return
	}
	if repo.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if repo.Location == "" {
		writeError(w, http.StatusBadRequest, "location is required")
		return
	}
	if err := h.repositories.Create(r.Context(), &repo); err != nil {
		h.log.Error("create repository", "error", err)
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, repo)
}

func (h *Handler) getRepository(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	repo, err := h.repositories.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

func (h *Handler) updateRepository(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var repo catalog.BackupRepository
	if err := decode(r, &repo); err != nil {
		handleDecodeError(w, err)
		return
	}
	repo.ID = id
	if err := h.repositories.Update(r.Context(), &repo); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

func (h *Handler) deleteRepository(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repositories.Delete(r.Context(), id); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
