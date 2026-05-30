package api

import "net/http"

// RegisterRoutes attaches all v1 API routes to mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)

	mux.HandleFunc("GET /v1/systems", h.listSystems)
	mux.HandleFunc("POST /v1/systems", h.createSystem)
	mux.HandleFunc("GET /v1/systems/{id}", h.getSystem)
	mux.HandleFunc("PUT /v1/systems/{id}", h.updateSystem)
	mux.HandleFunc("DELETE /v1/systems/{id}", h.deleteSystem)

	mux.HandleFunc("GET /v1/repositories", h.listRepositories)
	mux.HandleFunc("POST /v1/repositories", h.createRepository)
	mux.HandleFunc("GET /v1/repositories/{id}", h.getRepository)
	mux.HandleFunc("PUT /v1/repositories/{id}", h.updateRepository)
	mux.HandleFunc("DELETE /v1/repositories/{id}", h.deleteRepository)

	mux.HandleFunc("GET /v1/policies", h.listPolicies)
	mux.HandleFunc("POST /v1/policies", h.createPolicy)
	mux.HandleFunc("GET /v1/policies/{id}", h.getPolicy)
	mux.HandleFunc("PUT /v1/policies/{id}", h.updatePolicy)
	mux.HandleFunc("DELETE /v1/policies/{id}", h.deletePolicy)

	mux.HandleFunc("GET /v1/jobs", h.listJobs)
	mux.HandleFunc("POST /v1/jobs", h.createJob)
	mux.HandleFunc("GET /v1/jobs/{id}", h.getJob)
	mux.HandleFunc("PUT /v1/jobs/{id}", h.updateJob)
	mux.HandleFunc("DELETE /v1/jobs/{id}", h.deleteJob)

	mux.HandleFunc("GET /v1/snapshots", h.listSnapshots)
	mux.HandleFunc("POST /v1/snapshots", h.createSnapshot)
	mux.HandleFunc("GET /v1/snapshots/{id}", h.getSnapshot)
	mux.HandleFunc("DELETE /v1/snapshots/{id}", h.deleteSnapshot)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
