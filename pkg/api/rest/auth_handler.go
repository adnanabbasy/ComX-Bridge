package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Key string `json:"key"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 1. Verify API Key
	authConfig := s.engine.Config().API.Auth
	var role string
	valid := false

	// Check against users
	for _, u := range authConfig.Users {
		if u.Key == req.Key {
			valid = true
			role = u.Role
			break
		}
	}

	if !valid {
		respondError(w, http.StatusUnauthorized, "Invalid API Key")
		return
	}

	// 2. Generate JWT
	if authConfig.JWTSecret == "" {
		respondError(w, http.StatusInternalServerError, "JWT Secret not configured")
		return
	}

	claims := jwt.MapClaims{
		"sub":  req.Key, // Using Key as subject for now
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(authConfig.JWTSecret))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to sign token")
		return
	}

	respondJSON(w, http.StatusOK, LoginResponse{
		Token:     tokenString,
		ExpiresAt: int64(claims["exp"].(int64)),
	})
}
