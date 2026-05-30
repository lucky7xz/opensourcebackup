package api

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func (h *Handler) listSystems(w http.ResponseWriter, r *http.Request) {
	systems, err := h.systems.List(r.Context())
	if err != nil {
		h.log.Error("list systems", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if systems == nil {
		systems = []catalog.System{}
	}
	writeJSON(w, http.StatusOK, systems)
}

func (h *Handler) createSystem(w http.ResponseWriter, r *http.Request) {
	var s catalog.System
	if err := decode(r, &s); err != nil {
		handleDecodeError(w, err)
		return
	}
	if s.Hostname == "" {
		writeError(w, http.StatusBadRequest, "hostname is required")
		return
	}
	if s.RiskClass == "" {
		s.RiskClass = "standard"
	}
	if err := h.systems.Create(r.Context(), &s); err != nil {
		h.log.Error("create system", "error", err)
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) getSystem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	s, err := h.systems.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) updateSystem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var s catalog.System
	if err := decode(r, &s); err != nil {
		handleDecodeError(w, err)
		return
	}
	s.ID = id
	if err := h.systems.Update(r.Context(), &s); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) deleteSystem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.systems.Delete(r.Context(), id); err != nil {
		writeError(w, httpStatusForError(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
