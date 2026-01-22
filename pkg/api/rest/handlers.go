package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := s.engine.Status()
	respondJSON(w, http.StatusOK, status)
}

func (s *Server) handleListGateways(w http.ResponseWriter, r *http.Request) {
	// We need a way to get list of gateways from engine
	// Adding ListGateways() to Engine (already exists)
	gateways := s.engine.ListGateways()

	// Get detailed info for each
	gwInfos := make([]map[string]interface{}, 0)
	for _, name := range gateways {
		if gw, err := s.engine.GetGateway(name); err == nil {
			// Gateway Status is available
			gwInfos = append(gwInfos, map[string]interface{}{
				"name":   name,
				"status": gw.Status(),
			})
		}
	}

	respondJSON(w, http.StatusOK, gwInfos)
}

// handleSendGatewayRequest represents the payload for sending data.
type handleSendGatewayRequest struct {
	Data string `json:"data"` // Handles text or base64? For now assume text/raw
}

func (s *Server) handleSendGateway(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	gw, err := s.engine.GetGateway(name)
	if err != nil {
		respondError(w, http.StatusNotFound, "Gateway not found")
		return
	}

	var req handleSendGatewayRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Send data using SendRaw
	// This will use the configured transport to send the data
	_, err = gw.SendRaw(r.Context(), []byte(req.Data))
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to send data: %v", err))
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"status": "sent",
		"bytes":  fmt.Sprintf("%d", len(req.Data)),
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
