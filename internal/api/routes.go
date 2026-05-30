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
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
